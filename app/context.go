package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/datachainlab/fabric-ibc/auth/ante"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

func setupContext(ctx sdk.Context, stub shim.ChaincodeStubInterface) sdk.Context {
	return ante.ContextWithStub(ctx, stub)
}
