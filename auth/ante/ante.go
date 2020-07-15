package ante

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/datachainlab/fabric-ibc/auth/types"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
)

// Verify a transaction using Identity given from fabric stub and return an error if it is invalid. Note,
// the FabricIDVerificationDecorator decorator will not get executed on ReCheck.
//
// CONTRACT: Tx must implement StdTx interface
type FabricIDVerificationDecorator struct{}

func NewFabricIDVerificationDecorator() FabricIDVerificationDecorator {
	return FabricIDVerificationDecorator{}
}

func (dec FabricIDVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	stdTx := tx.(types.StdTx)

	// no need to verify signatures on recheck tx
	if ctx.IsReCheckTx() {
		return next(ctx, stdTx, simulate)
	}

	stub := StubFromContext(ctx)
	ci, err := cid.New(stub)
	if err != nil {
		return ctx, err
	}
	id, err := ci.GetID()
	if err != nil {
		return ctx, err
	}
	signer := stdTx.GetSigners()[0]

	if !bytes.Equal(signer, []byte(id)) {
		return ctx, fmt.Errorf("got unexpected signer: expected=%X actual=%v", signer, []byte(id))
	}

	return next(ctx, stdTx, simulate)
}
