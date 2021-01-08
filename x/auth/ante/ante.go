package ante

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/datachainlab/fabric-ibc/x/auth/types"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
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
	ci, err := cid.New(stub)
	if err != nil {
		return ctx, err
	}
	_ = ci
	// id, err := ci.GetID()
	// if err != nil {
	// 	return ctx, err
	// }
	creator, err := stub.GetCreator()
	if err != nil {
		return ctx, err
	}
	creatorHash := sha256.Sum256(creator)

	signers := getSigners(tx.GetMsgs()...)
	if len(signers) != 1 {
		return ctx, fmt.Errorf("the number of signers must be 1")
	}

	if !bytes.Equal(signers[0], []byte(creatorHash[:20])) {
		return ctx, fmt.Errorf("got unexpected signer: expected=%X actual=%X", signers[0], []byte(creatorHash[:20]))
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
