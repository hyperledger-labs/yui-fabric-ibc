package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-fabric-ibc/x/auth/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

func setupContext(ctx sdk.Context, stub shim.ChaincodeStubInterface) sdk.Context {
	return types.ContextWithStub(ctx, stub)
}
