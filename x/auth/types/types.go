package types

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Tx = (*StdTx)(nil)

// StdTx is a standard way to wrap Msgs.
type StdTx struct {
	Msgs []sdk.Msg `json:"msg" yaml:"msg"`
}

func NewStdTx(msgs []sdk.Msg) StdTx {
	return StdTx{
		Msgs: msgs,
	}
}

// GetMsgs returns the all the transaction's messages.
func (tx StdTx) GetMsgs() []sdk.Msg { return tx.Msgs }

// ValidateBasic does a simple and lightweight validation check that doesn't
// require access to any other information.
func (tx StdTx) ValidateBasic() error {
	if len(tx.Msgs) == 0 {
		return errors.New("StdTx must cointain at least one Msg")
	}
	signers := tx.GetSigners()
	if len(signers) != 1 {
		return fmt.Errorf("StdTx must cointain one signer")
	}
	return nil
}

// GetSigners returns the identities that must sign the transaction.
// Identities are returned in a deterministic order.
// They are accumulated from the GetSigners method for each Msg
// in the order they appear in tx.GetMsgs().
// Duplicate identities will be omitted.
func (tx StdTx) GetSigners() []sdk.AccAddress {
	var signers []sdk.AccAddress
	seen := map[string]bool{}

	for _, msg := range tx.GetMsgs() {
		for _, addr := range msg.GetSigners() {
			if !seen[addr.String()] {
				signers = append(signers, addr)
				seen[addr.String()] = true
			}
		}
	}

	return signers
}
