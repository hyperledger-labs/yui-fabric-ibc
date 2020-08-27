package app

import (
	"strings"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Query implements the ABCI interface. It delegates to CommitMultiStore if it
// implements Queryable.
func (app *BaseApp) Query(req abci.RequestQuery) abci.ResponseQuery {
	path := splitPath(req.Path)
	if len(path) == 0 {
		return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "no query path provided"))
	}

	switch path[0] {
	// "/app" prefix for special application queries
	// case "app":
	// 	return handleQueryApp(app, path, req)

	// case "store":
	// 	return handleQueryStore(app, path, req)

	// case "p2p":
	// 	return handleQueryP2P(app, path)

	case "custom":
		return handleQueryCustom(app, path, req)
	}

	return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unknown query path"))
}

func handleQueryCustom(app *BaseApp, path []string, req abci.RequestQuery) abci.ResponseQuery {
	// path[0] should be "custom" because "/custom" prefix is required for keeper
	// queries.
	//
	// The QueryRouter routes using path[1]. For example, in the path
	// "custom/gov/proposal", QueryRouter routes using "gov".
	if len(path) < 2 || path[1] == "" {
		return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "no route for custom query specified"))
	}

	querier := app.queryRouter.Route(path[1])
	if querier == nil {
		return sdkerrors.QueryResult(sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "no custom querier found for route %s", path[1]))
	}

	ctx, err := app.createQueryContext(req)
	if err != nil {
		return sdkerrors.QueryResult(err)
	}

	// Passes the rest of the path as an argument to the querier.
	//
	// For example, in the path "custom/gov/proposal/test", the gov querier gets
	// []string{"proposal", "test"} as the path.
	resBytes, err := querier(ctx, path[2:], req)
	if err != nil {
		res := sdkerrors.QueryResult(err)
		res.Height = req.Height
		return res
	}

	return abci.ResponseQuery{
		Height: req.Height,
		Value:  resBytes,
	}
}

func (app *BaseApp) createQueryContext(req abci.RequestQuery) (sdk.Context, error) {
	// when a client did not provide a query height, manually inject the latest
	// if req.Height == 0 {
	// 	req.Height = app.LastBlockHeight()
	// }

	// if req.Height <= 1 && req.Prove {
	// 	return sdk.Context{},
	// 		sdkerrors.Wrap(
	// 			sdkerrors.ErrInvalidRequest,
	// 			"cannot query with proof when height <= 1; please provide a valid height",
	// 		)
	// }

	cacheMS := app.cms.CacheMultiStore()
	// cacheMS, err := app.cms.CacheMultiStoreWithVersion(req.Height)
	// if err != nil {
	// 	return sdk.Context{},
	// 		sdkerrors.Wrapf(
	// 			sdkerrors.ErrInvalidRequest,
	// 			"failed to load state at height %d; %s (latest height: %d)", req.Height, err, app.LastBlockHeight(),
	// 		)
	// }

	// cache wrap the commit-multistore for safety
	ctx := sdk.NewContext(
		cacheMS, abci.Header{}, true, app.logger,
	)

	return ctx, nil
}

// splitPath splits a string path using the delimiter '/'.
//
// e.g. "this/is/funny" becomes []string{"this", "is", "funny"}
func splitPath(requestPath string) (path []string) {
	path = strings.Split(requestPath, "/")

	// first element is empty string
	if len(path) > 0 && path[0] == "" {
		path = path[1:]
	}

	return path
}
