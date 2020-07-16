package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type stubKey struct{}

func ContextWithStub(ctx sdk.Context, stub shim.ChaincodeStubInterface) sdk.Context {
	return ctx.WithValue(stubKey{}, stub)
}

func StubFromContext(ctx sdk.Context) shim.ChaincodeStubInterface {
	return ctx.Value(stubKey{}).(shim.ChaincodeStubInterface)
}
