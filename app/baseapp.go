package app

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
	ibckeeper "github.com/cosmos/ibc-go/modules/core/keeper"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger-labs/yui-fabric-ibc/store"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
	dbm "github.com/tendermint/tm-db"
)

type Application interface {
	InitChain(appStateBytes []byte) error
	RunTx(stub shim.ChaincodeStubInterface, txBytes []byte) (result *sdk.Result, err error)
	Query(req abci.RequestQuery) abci.ResponseQuery

	MakeCacheContext(header tmproto.Header) (ctx sdk.Context, writer func())
	AppCodec() codec.Codec
	GetIBCKeeper() ibckeeper.Keeper
}

type BaseApp struct {
	logger           log.Logger
	name             string                    // application name from abci.Info
	db               dbm.DB                    // common DB backend
	cms              *store.Store              // Main (uncached) state
	router           sdk.Router                // handle any kind of message
	queryRouter      sdk.QueryRouter           // router for redirecting query calls
	grpcQueryRouter  *GRPCQueryRouter          // router for redirecting gRPC query calls
	msgServiceRouter *baseapp.MsgServiceRouter // router for redirecting Msg service messages
	txDecoder        sdk.TxDecoder             // unmarshal []byte into sdk.Tx

	anteHandler   sdk.AnteHandler // ante handler for fee and auth
	initChainer   InitChainer
	blockProvider BlockProvider

	// paramStore is used to query for ABCI consensus parameters from an
	// application parameter store.
	paramStore ParamStore

	// flag for sealing options and parameters to a BaseApp
	sealed bool
}

func NewBaseApp(
	name string, logger log.Logger, db dbm.DB, txDecoder sdk.TxDecoder,
) *BaseApp {
	app := &BaseApp{
		logger:           logger,
		name:             name,
		db:               db,
		cms:              store.NewStore(db),
		router:           baseapp.NewRouter(),
		queryRouter:      baseapp.NewQueryRouter(),
		grpcQueryRouter:  NewGRPCQueryRouter(),
		msgServiceRouter: baseapp.NewMsgServiceRouter(),
		txDecoder:        txDecoder,
	}

	return app
}

// Name returns the name of the BaseApp.
func (app *BaseApp) Name() string {
	return app.name
}

// Logger returns the logger of the BaseApp.
func (app *BaseApp) Logger() log.Logger {
	return app.logger
}

// MsgServiceRouter returns the MsgServiceRouter of a BaseApp.
func (app *BaseApp) MsgServiceRouter() *baseapp.MsgServiceRouter { return app.msgServiceRouter }

// Router returns the router of the BaseApp.
func (app *BaseApp) Router() sdk.Router {
	if app.sealed {
		// We cannot return a Router when the app is sealed because we can't have
		// any routes modified which would cause unexpected routing behavior.
		panic("Router() on sealed BaseApp")
	}

	return app.router
}

// QueryRouter returns the QueryRouter of a BaseApp.
func (app *BaseApp) QueryRouter() sdk.QueryRouter { return app.queryRouter }

// Seal seals a BaseApp. It prohibits any further modifications to a BaseApp.
func (app *BaseApp) Seal() { app.sealed = true }

// IsSealed returns true if the BaseApp is sealed and false otherwise.
func (app *BaseApp) IsSealed() bool { return app.sealed }

// MountStores mounts all IAVL or DB stores to the provided keys in the BaseApp
// multistore.
func (app *BaseApp) MountKVStores(keys map[string]*sdk.KVStoreKey) {
	for _, key := range keys {
		app.MountStore(key, sdk.StoreTypeDB)
	}
}

// MountStores mounts all IAVL or DB stores to the provided keys in the BaseApp
// multistore.
func (app *BaseApp) MountTransientStores(keys map[string]*sdk.TransientStoreKey) {
	for _, key := range keys {
		app.MountStore(key, sdk.StoreTypeTransient)
	}
}

// MountMemoryStores mounts all in-memory KVStores with the BaseApp's internal
// commit multi-store.
func (app *BaseApp) MountMemoryStores(keys map[string]*sdk.MemoryStoreKey) {
	for _, memKey := range keys {
		app.MountStore(memKey, sdk.StoreTypeMemory)
	}
}

// MountStoreWithDB mounts a store to the provided key in the BaseApp
// multistore, using a specified DB.
func (app *BaseApp) MountStoreWithDB(key sdk.StoreKey, typ sdk.StoreType, db dbm.DB) {
	app.cms.MountStoreWithDB(key, typ, db)
}

// MountStore mounts a store to the provided key in the BaseApp multistore,
// using the default DB.
func (app *BaseApp) MountStore(key sdk.StoreKey, typ sdk.StoreType) {
	app.cms.MountStoreWithDB(key, typ, nil)
}

func (app *BaseApp) LoadLatestVersion() error {
	if err := app.cms.LoadStores(); err != nil {
		return err
	}
	return nil
}

func (app *BaseApp) init() error {
	if app.sealed {
		panic("cannot call initFromMainStore: BaseApp already sealed")
	}

	// needed for the export command which inits from store but never calls initchain
	app.Seal()

	return nil
}

func (app *BaseApp) InitChain(appStateBytes []byte) error {
	if app.initChainer == nil {
		return nil
	}
	ms := app.cms.CacheMultiStore()
	ctx := sdk.NewContext(ms, tmproto.Header{}, false, app.logger)
	if err := app.initChainer(ctx, appStateBytes); err != nil {
		return err
	}
	ms.Write()
	return nil
}

func (app *BaseApp) getBlockHeader() tmproto.Header {
	block := app.blockProvider()
	return tmproto.Header{
		ChainID: app.name,
		Height:  block.Height() + 1,
		Time:    tmtime.Now(),
	}
}

func (app *BaseApp) RunTx(stub shim.ChaincodeStubInterface, txBytes []byte) (result *sdk.Result, err error) {
	tx, err := app.txDecoder(txBytes)
	if err != nil {
		return nil, err
	}

	ms := app.cms.CacheMultiStore()
	ctx := setupContext(sdk.NewContext(ms, app.getBlockHeader(), false, app.logger), stub)

	msgs := tx.GetMsgs()
	if err := validateBasicTxMsgs(msgs); err != nil {
		return nil, err
	}

	var events sdk.Events
	if app.anteHandler != nil {
		var (
			anteCtx sdk.Context
			msCache sdk.CacheMultiStore
		)

		// Cache wrap context before AnteHandler call in case it aborts.
		// This is required for both CheckTx and DeliverTx.
		// Ref: https://github.com/cosmos/cosmos-sdk/issues/2772
		//
		// NOTE: Alternatively, we could require that AnteHandler ensures that
		// writes do not happen if aborted/failed.  This may have some
		// performance benefits, but it'll be more difficult to get right.
		anteCtx, msCache = app.cacheTxContext(ctx, txBytes)
		anteCtx = anteCtx.WithEventManager(sdk.NewEventManager())
		newCtx, err := app.anteHandler(anteCtx, tx, false)

		if !newCtx.IsZero() {
			// At this point, newCtx.MultiStore() is cache-wrapped, or something else
			// replaced by the AnteHandler. We want the original multistore, not one
			// which was cache-wrapped for the AnteHandler.
			//
			// Also, in the case of the tx aborting, we need to track gas consumed via
			// the instantiated gas meter in the AnteHandler, so we update the context
			// prior to returning.
			ctx = newCtx.WithMultiStore(ms)
		}

		events = ctx.EventManager().Events()

		if err != nil {
			return nil, err
		}

		msCache.Write()
	}
	// Create a new Context based off of the existing Context with a cache-wrapped
	// MultiStore in case message processing fails. At this point, the MultiStore
	// is doubly cached-wrapped.
	runMsgCtx, msCache := app.cacheTxContext(ctx, txBytes)
	// Attempt to execute all messages and only update state if all messages pass
	// and we're in DeliverTx. Note, runMsgs will never return a reference to a
	// Result if any single message fails or does not have a registered Handler.
	result, err = app.runMsgs(runMsgCtx, msgs)
	if err == nil {
		msCache.Write()

		if len(events) > 0 {
			// append the events in the order of occurrence
			result.Events = append(events.ToABCIEvents(), result.Events...)
		}
	} else {
		return result, err
	}

	// commit
	ms.Write()
	return result, nil
}

func (app *BaseApp) runMsgs(ctx sdk.Context, msgs []sdk.Msg) (*sdk.Result, error) {
	msgLogs := make(sdk.ABCIMessageLogs, 0, len(msgs))
	events := sdk.EmptyEvents()
	txMsgData := &sdk.TxMsgData{
		Data: make([]*sdk.MsgData, 0, len(msgs)),
	}

	// NOTE: GasWanted is determined by the AnteHandler and GasUsed by the GasMeter.
	for i, msg := range msgs {
		var (
			msgResult    *sdk.Result
			eventMsgName string // name to use as value in event `message.action`
			err          error
		)

		if handler := app.msgServiceRouter.Handler(msg); handler != nil {
			// ADR 031 request type routing
			msgResult, err = handler(ctx, msg)
			eventMsgName = sdk.MsgTypeURL(msg)
		} else if legacyMsg, ok := msg.(legacytx.LegacyMsg); ok {
			// legacy sdk.Msg routing
			// Assuming that the app developer has migrated all their Msgs to
			// proto messages and has registered all `Msg services`, then this
			// path should never be called, because all those Msgs should be
			// registered within the `msgServiceRouter` already.
			msgRoute := legacyMsg.Route()
			eventMsgName = legacyMsg.Type()
			handler := app.router.Route(ctx, msgRoute)
			if handler == nil {
				return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized message route: %s; message index: %d", msgRoute, i)
			}

			msgResult, err = handler(ctx, msg)
		} else {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "can't route message %+v", msg)
		}

		if err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to execute message; message index: %d", i)
		}

		msgEvents := sdk.Events{
			sdk.NewEvent(sdk.EventTypeMessage, sdk.NewAttribute(sdk.AttributeKeyAction, eventMsgName)),
		}
		msgEvents = msgEvents.AppendEvents(msgResult.GetEvents())

		// append message events, data and logs
		//
		// Note: Each message result's data must be length-prefixed in order to
		// separate each result.
		events = events.AppendEvents(msgEvents)

		txMsgData.Data = append(txMsgData.Data, &sdk.MsgData{MsgType: sdk.MsgTypeURL(msg), Data: msgResult.Data})
		msgLogs = append(msgLogs, sdk.NewABCIMessageLog(uint32(i), msgResult.Log, msgEvents))
	}

	data, err := proto.Marshal(txMsgData)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to marshal tx data")
	}

	return &sdk.Result{
		Data:   data,
		Log:    strings.TrimSpace(msgLogs.String()),
		Events: events.ToABCIEvents(),
	}, nil
}

// validateBasicTxMsgs executes basic validator calls for messages.
func validateBasicTxMsgs(msgs []sdk.Msg) error {
	if len(msgs) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "must contain at least one message")
	}

	for _, msg := range msgs {
		err := msg.ValidateBasic()
		if err != nil {
			return err
		}
	}

	return nil
}

// cacheTxContext returns a new context based off of the provided context with
// a cache wrapped multi-store.
func (app *BaseApp) cacheTxContext(ctx sdk.Context, txBytes []byte) (sdk.Context, sdk.CacheMultiStore) {
	ms := ctx.MultiStore()
	// TODO: https://github.com/cosmos/cosmos-sdk/issues/2824
	msCache := ms.CacheMultiStore()
	if msCache.TracingEnabled() {
		msCache = msCache.SetTracingContext(
			sdk.TraceContext(
				map[string]interface{}{
					"txHash": fmt.Sprintf("%X", tmhash.Sum(txBytes)),
				},
			),
		).(sdk.CacheMultiStore)
	}

	return ctx.WithMultiStore(msCache), msCache
}

//----------------------------------------
// +Options

// AnteHandlerProvider provides an anteHandler
type AnteHandlerProvider func(
	ibcKeeper ibckeeper.Keeper,
	sigGasConsumer ante.SignatureVerificationGasConsumer,
) sdk.AnteHandler

func (app *BaseApp) SetAnteHandler(ah sdk.AnteHandler) {
	if app.sealed {
		panic("SetAnteHandler() on sealed BaseApp")
	}

	app.anteHandler = ah
}

// SetRouter allows us to customize the router.
func (app *BaseApp) SetRouter(router sdk.Router) {
	if app.sealed {
		panic("SetRouter() on sealed BaseApp")
	}
	app.router = router
}

type InitChainer func(ctx sdk.Context, appStateBytes []byte) error

func (app *BaseApp) SetInitChainer(initChainer InitChainer) {
	app.initChainer = initChainer
}

type Block interface {
	Height() int64
	Timestamp() int64
}

type BlockProvider func() Block

func (app *BaseApp) SetBlockProvider(blockProvider BlockProvider) {
	app.blockProvider = blockProvider
}

func (app BaseApp) BlockProvider() BlockProvider {
	return app.blockProvider
}

//----------------------------------------
// +Helpers

func (app *BaseApp) MakeCacheContext(header tmproto.Header) (ctx sdk.Context, writer func()) {
	ms := app.cms.CacheMultiStore()
	return sdk.NewContext(ms, header, false, app.logger), ms.Write
}
