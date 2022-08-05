package app

import "github.com/cosmos/cosmos-sdk/codec/types"

// SetParamStore sets a parameter store on the BaseApp.
func (app *BaseApp) SetParamStore(ps ParamStore) {
	if app.sealed {
		panic("SetParamStore() on sealed BaseApp")
	}

	app.paramStore = ps
}

// SetInterfaceRegistry sets the InterfaceRegistry.
func (app *BaseApp) SetInterfaceRegistry(registry types.InterfaceRegistry) {
	// app.interfaceRegistry = registry
	app.grpcQueryRouter.SetInterfaceRegistry(registry)
	app.msgServiceRouter.SetInterfaceRegistry(registry)
}
