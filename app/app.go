package app

import (
	"io"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	transfer "github.com/cosmos/cosmos-sdk/x/ibc-transfer"
	ibctransferkeeper "github.com/cosmos/cosmos-sdk/x/ibc-transfer/keeper"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	port "github.com/cosmos/cosmos-sdk/x/ibc/05-port"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/datachainlab/fabric-ibc/x/ibc"
	client "github.com/datachainlab/fabric-ibc/x/ibc/02-client"
	ibcante "github.com/datachainlab/fabric-ibc/x/ibc/ante"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

const appName = "FabricIBC"

var (
	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		ibc.AppModuleBasic{},
		transfer.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:  nil,
		distrtypes.ModuleName:       nil,
		ibctransfertypes.ModuleName: {authtypes.Minter, authtypes.Burner},
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName: true,
	}
)

type IBCApp struct {
	*BaseApp
	cdc      *codec.Codec
	appCodec codec.Marshaler

	txDecoder sdk.TxDecoder

	// keys to access the substores
	keys    map[string]*sdk.KVStoreKey
	memKeys map[string]*sdk.MemoryStoreKey

	// subspaces
	subspaces map[string]paramstypes.Subspace

	// keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	IBCKeeper        *ibc.Keeper
	TransferKeeper   ibctransferkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper capabilitykeeper.ScopedKeeper

	// the module manager
	mm *module.Manager
}

func NewIBCApp(logger log.Logger, db dbm.DB, traceStore io.Writer, cskProvider SelfConsensusStateKeeperProvider, blockProvider BlockProvider) (*IBCApp, error) {
	appCodec, cdc := MakeCodecs()
	bApp := NewBaseApp(appName, logger, db, JSONTxDecoder(cdc))
	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey,
		stakingtypes.StoreKey, paramstypes.StoreKey, ibc.StoreKey, ibctransfertypes.StoreKey, capabilitytypes.StoreKey,
	)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)

	app := &IBCApp{
		BaseApp:   bApp,
		cdc:       cdc,
		appCodec:  appCodec,
		keys:      keys,
		memKeys:   memKeys,
		txDecoder: JSONTxDecoder(cdc),
		subspaces: make(map[string]paramstypes.Subspace),
	}

	// init params keeper and subspaces
	app.ParamsKeeper = paramskeeper.NewKeeper(appCodec, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	app.subspaces[authtypes.ModuleName] = app.ParamsKeeper.Subspace(authtypes.DefaultParamspace)
	app.subspaces[banktypes.ModuleName] = app.ParamsKeeper.Subspace(banktypes.DefaultParamspace)

	// // set the BaseApp's parameter store
	// bApp.SetParamStore(app.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(std.ConsensusParamsKeyTable()))

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibc.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], app.subspaces[authtypes.ModuleName], authtypes.ProtoBaseAccount, maccPerms,
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.AccountKeeper, app.subspaces[banktypes.ModuleName], make(map[string]bool),
	)
	app.IBCKeeper = ibc.NewKeeper(
		app.cdc, appCodec, keys[ibc.StoreKey], nil, cskProvider(), scopedIBCKeeper, // TODO set stakingKeeper
	)

	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey],
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		app.AccountKeeper, app.BankKeeper, scopedTransferKeeper,
	)
	transferModule := transfer.NewAppModule(app.TransferKeeper)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := port.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)
	app.IBCKeeper.SetRouter(ibcRouter)

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		auth.NewAppModule(appCodec, app.AccountKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		params.NewAppModule(app.ParamsKeeper),
		transferModule,
	)

	app.mm.RegisterRoutes(app.Router(), app.QueryRouter())

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetAnteHandler(NewAnteHandler(*app.IBCKeeper, ante.DefaultSigVerificationGasConsumer))
	app.SetInitChainer(app.InitChainer)
	app.SetBlockProvider(blockProvider)

	if err := app.LoadLatestVersion(); err != nil {
		return nil, err
	}

	// Initialize and seal the capability keeper so all persistent capabilities
	// are loaded in-memory and prevent any further modules from creating scoped
	// sub-keepers.
	ms := app.cms.CacheMultiStore()
	ctx := sdk.NewContext(ms, abci.Header{}, false, app.logger)
	app.CapabilityKeeper.InitializeAndSeal(ctx)
	ms.Write()

	app.ScopedIBCKeeper = scopedIBCKeeper

	return app, nil
}

// This key was created for ICS-20 testing
var MasterAccount crypto.PrivKey

func init() {
	MasterAccount = secp256k1.GenPrivKey()
}

func (app *IBCApp) InitChainer(ctx sdk.Context, appStateBytes []byte) error {
	var genesisState simapp.GenesisState
	app.cdc.MustUnmarshalJSON(appStateBytes, &genesisState)

	// res := app.mm.InitGenesis(ctx, app.cdc, genesisState)
	// https://github.com/cosmos/cosmos-sdk/blob/24b9be0ef841303a2e2b6f60042b5da3b74af2ef/simapp/cmd/simd/genaccounts.go#L73
	// FIXME these states should be moved into genesisState
	auth.InitGenesis(
		ctx,
		app.AccountKeeper,
		authtypes.NewGenesisState(
			authtypes.DefaultParams(),
			authtypes.GenesisAccounts{
				authtypes.NewBaseAccount(
					sdk.AccAddress(MasterAccount.PubKey().Address()),
					MasterAccount.PubKey(),
					1,
					1,
				),
			},
		),
	)
	addr := sdk.AccAddress(MasterAccount.PubKey().Address())
	coins := sdk.NewCoins(sdk.NewCoin("ftk", sdk.NewInt(1000)))
	balances := banktypes.Balance{Address: addr, Coins: coins.Sort()}
	bankState := banktypes.DefaultGenesisState()
	bankState.Balances = append(bankState.Balances, balances)
	bank.InitGenesis(ctx, app.BankKeeper, bankState)
	transfer.InitGenesis(ctx, app.TransferKeeper, ibctransfertypes.DefaultGenesisState())

	return nil
}

func (app *IBCApp) Codec() *codec.Codec {
	return app.cdc
}

// MakeCodecs constructs the *std.Codec and *codec.Codec instances used by
// simapp. It is useful for tests and clients who do not want to construct the
// full simapp
func MakeCodecs() (codec.Marshaler, *codec.Codec) {
	config := MakeEncodingConfig()
	return config.Marshaler, config.Amino
}

// MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
//
// TODO: this file should add a "+build test_amino" flag for #6190 and a proto.go file with a protobuf configuration
func MakeEncodingConfig() simappparams.EncodingConfig {
	encodingConfig := simappparams.MakeEncodingConfig()
	std.RegisterCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaceModules(encodingConfig.InterfaceRegistry)
	return encodingConfig
}

func JSONTxDecoder(cdc *codec.Codec) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		var tx = authtypes.StdTx{}

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

type SelfConsensusStateKeeperProvider func() client.SelfConsensusStateKeeper

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
