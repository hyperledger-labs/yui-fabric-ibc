package app

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// Query implements the ABCI interface. It delegates to CommitMultiStore if it
// implements Queryable.
func (app *BaseApp) Query(req abci.RequestQuery) abci.ResponseQuery {

	// handle gRPC routes first rather than calling splitPath because '/' characters
	// are used as part of gRPC paths
	if grpcHandler := app.grpcQueryRouter.Route(req.Path); grpcHandler != nil {
		return app.handleQueryGRPC(grpcHandler, req)
	}

	path := splitPath(req.Path)
	if len(path) == 0 {
		return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "no query path provided"))
	}

	switch path[0] {
	case "store":
		return handleQueryStore(app, path, req)
	case "custom":
		return handleQueryCustom(app, path, req)
	}

	return sdkerrors.QueryResult(sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unknown query path"))
}

func (app *BaseApp) handleQueryGRPC(handler GRPCQueryHandler, req abci.RequestQuery) abci.ResponseQuery {
	ctx, err := app.createQueryContext(req.Height, req.Prove)
	if err != nil {
		return sdkerrors.QueryResult(err)
	}

	res, err := handler(ctx, req)
	if err != nil {
		res = sdkerrors.QueryResult(gRPCErrorToSDKError(err))
		res.Height = req.Height
		return res
	}

	return res
}

func handleQueryStore(app *BaseApp, path []string, req abci.RequestQuery) abci.ResponseQuery {
	// "/store" prefix for store queries
	req.Path = "/" + strings.Join(path[1:], "/")
	return app.cms.Query(req)
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

	ctx, err := app.createQueryContext(req.Height, req.Prove)
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

func (app *BaseApp) createQueryContext(height int64, prove bool) (sdk.Context, error) {
	cacheMS := app.cms.CacheMultiStore()

	// cache wrap the commit-multistore for safety
	ctx := sdk.NewContext(
		cacheMS, tmproto.Header{}, true, app.logger,
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

func gRPCErrorToSDKError(err error) error {
	status, ok := grpcstatus.FromError(err)
	if !ok {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}

	switch status.Code() {
	case codes.NotFound:
		return sdkerrors.Wrap(sdkerrors.ErrKeyNotFound, err.Error())
	case codes.InvalidArgument:
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	case codes.FailedPrecondition:
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	case codes.Unauthenticated:
		return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, err.Error())
	default:
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
	}
}
