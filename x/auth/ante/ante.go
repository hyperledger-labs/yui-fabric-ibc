package ante

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-fabric-ibc/x/auth/types"
)

// Verify a transaction using Identity given from fabric stub and return an error if it is invalid. Note,
// the FabricIDVerificationDecorator decorator will not get executed on ReCheck.
//
// CONTRACT: Tx must implement sdk.Tx interface
type FabricIDVerificationDecorator struct{}

func NewFabricIDVerificationDecorator() FabricIDVerificationDecorator {
	return FabricIDVerificationDecorator{}
}

func (dec FabricIDVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// no need to verify signatures on recheck tx
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	stub := types.StubFromContext(ctx)
	creator, err := stub.GetCreator()
	if err != nil {
		return ctx, err
	}
	addr, err := types.MakeCreatorAddressWithBytes(creator)
	if err != nil {
		return ctx, err
	}

	signers := getSigners(tx.GetMsgs()...)
	if len(signers) != 1 {
		return ctx, fmt.Errorf("the number of signers must be 1")
	}

	if !bytes.Equal(signers[0], addr) {
		return ctx, fmt.Errorf("got unexpected signer: expected=%X actual=%X", signers[0], addr)
	}

	return next(ctx, tx, simulate)
}

func getSigners(msgs ...sdk.Msg) []sdk.AccAddress {
	var signers []sdk.AccAddress
	seen := map[string]bool{}

	for _, msg := range msgs {
		for _, addr := range msg.GetSigners() {
			if !seen[addr.String()] {
				signers = append(signers, addr)
				seen[addr.String()] = true
			}
		}
	}

	return signers
}
