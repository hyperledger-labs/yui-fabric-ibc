package chaincode

import (
	"errors"
	"io"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/capability"
	port "github.com/cosmos/cosmos-sdk/x/ibc/05-port"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/datachainlab/fabric-ibc/x/compat"
	"github.com/datachainlab/fabric-ibc/x/ibc"
	ibcante "github.com/datachainlab/fabric-ibc/x/ibc/ante"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	db "github.com/tendermint/tm-db"
	dbm "github.com/tendermint/tm-db"
)

const appName = "FabricIBC"

var (
	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		ibc.AppModuleBasic{},
	)
)

type DBProvider func(shim.ChaincodeStubInterface) dbm.DB

func DefaultDBProvider(stub shim.ChaincodeStubInterface) db.DB {
	return compat.NewDB(stub)
}

type AppRunner struct {
	logger     log.Logger
	traceStore io.Writer
	dbProvider DBProvider
}

func NewAppRunner(logger log.Logger, dbProvider DBProvider) AppRunner {
	return AppRunner{
		logger:     logger,
		dbProvider: dbProvider,
	}
}

func (r AppRunner) RunMsg(stub shim.ChaincodeStubInterface, msgJSON string) error {
	// FIXME can we reuse single instance instead of making new app per request?
	db := r.dbProvider(stub)
	app, err := NewApp(r.logger, db, r.traceStore, true)
	if err != nil {
		return err
	}

	_ = app.InitChain(abci.RequestInitChain{AppStateBytes: []byte("{}")})

	res := app.DeliverTx(
		abci.RequestDeliverTx{
			Tx: []byte(msgJSON),
		},
	)
	if res.IsErr() {
		return errors.New(res.String())
	}
	_ = app.Commit()
	return nil
}

type App struct {
	*bam.BaseApp
	cdc *codec.Codec

	txDecoder sdk.TxDecoder

	// keys to access the substores
	keys    map[string]*sdk.KVStoreKey
	memKeys map[string]*sdk.MemoryStoreKey

	// subspaces
	subspaces map[string]params.Subspace

	// keepers
	CapabilityKeeper *capability.Keeper
	IBCKeeper        *ibc.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper capability.ScopedKeeper

	// the module manager
	mm *module.Manager
}

func JSONTxDecoder(cdc *codec.Codec) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		var tx = auth.StdTx{}

		if len(txBytes) == 0 {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "tx bytes are empty")
		}

		// StdTx.Msg is an interface. The concrete types
		// are registered by MakeTxCodec
		err := cdc.UnmarshalJSON(txBytes, &tx)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrTxDecode, err.Error())
		}

		return tx, nil
	}
}

// NewSimApp returns a reference to an initialized SimApp.
func NewApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool, baseAppOptions ...func(*bam.BaseApp),
) (*App, error) {
	appCodec, cdc := MakeCodecs()

	bApp := bam.NewBaseApp(appName, logger, db, JSONTxDecoder(cdc), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetAppVersion(version.Version)

	keys := sdk.NewKVStoreKeys(
		staking.StoreKey, ibc.StoreKey, capability.StoreKey,
	)
	memKeys := sdk.NewMemoryStoreKeys(capability.MemStoreKey)

	app := &App{
		BaseApp:   bApp,
		cdc:       cdc,
		keys:      keys,
		memKeys:   memKeys,
		subspaces: make(map[string]params.Subspace),
	}

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capability.NewKeeper(appCodec, keys[capability.StoreKey], memKeys[capability.MemStoreKey])
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibc.ModuleName)
	app.IBCKeeper = ibc.NewKeeper(
		app.cdc, appCodec, keys[ibc.StoreKey], nil, scopedIBCKeeper, // TODO set stakingKeeper
	)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := port.NewRouter()
	/*
		ibcRouter.AddRoute(transfer.ModuleName, transferModule)
		ibcRouter.AddRoute(cross.ModuleName, crossModule)
	*/
	app.IBCKeeper.SetRouter(ibcRouter)

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		ibc.NewAppModule(app.IBCKeeper),
	)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter())

	// initialize stores
	app.MountKVStores(keys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetAnteHandler(NewAnteHandler(*app.IBCKeeper, ante.DefaultSigVerificationGasConsumer))
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		err := app.LoadLatestVersion()
		if err != nil {
			return nil, err
		}
	}

	// Initialize and seal the capability keeper so all persistent capabilities
	// are loaded in-memory and prevent any further modules from creating scoped
	// sub-keepers.
	ctx := app.BaseApp.NewContext(true, abci.Header{})
	app.CapabilityKeeper.InitializeAndSeal(ctx)

	app.ScopedIBCKeeper = scopedIBCKeeper

	return app, nil
}

// MakeCodecs constructs the *std.Codec and *codec.Codec instances used by
// simapp. It is useful for tests and clients who do not want to construct the
// full simapp
func MakeCodecs() (*std.Codec, *codec.Codec) {
	cdc := std.MakeCodec(ModuleBasics)
	interfaceRegistry := types.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaceModules(interfaceRegistry)
	appCodec := std.NewAppCodec(cdc, interfaceRegistry)
	return appCodec, cdc
}

// Name returns the name of the App
func (app *App) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *App) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *App) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *App) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState simapp.GenesisState
	app.cdc.MustUnmarshalJSON(req.AppStateBytes, &genesisState)

	res := app.mm.InitGenesis(ctx, app.cdc, genesisState)

	// // Set Historical infos in InitChain to ignore genesis params
	// stakingParams := staking.DefaultParams()
	// stakingParams.HistoricalEntries = 1000
	// app.StakingKeeper.SetParams(ctx, stakingParams)

	return res
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func NewAnteHandler(
	ibcKeeper ibc.Keeper,
	sigGasConsumer ante.SignatureVerificationGasConsumer,
) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		// ante.NewMempoolFeeDecorator(),
		ante.NewValidateBasicDecorator(),
		// ante.NewValidateMemoDecorator(ak),
		// ante.NewConsumeGasForTxSizeDecorator(ak),
		// NewSetPubKeyDecorator(ak), // SetPubKeyDecorator must be called before all signature verification decorators
		// NewValidateSigCountDecorator(ak),
		// NewDeductFeeDecorator(ak, bankKeeper),
		// NewSigGasConsumeDecorator(ak, sigGasConsumer),
		// NewSigVerificationDecorator(ak),
		// NewIncrementSequenceDecorator(ak),
		ibcante.NewProofVerificationDecorator(ibcKeeper.ClientKeeper, ibcKeeper.ChannelKeeper), // innermost AnteDecorator
	)
}
