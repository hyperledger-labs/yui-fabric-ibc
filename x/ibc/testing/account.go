package testing

import (
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type Account struct {
	authtypes.AccountI

	creatorHash []byte
}

var _ authtypes.AccountI = (*Account)(nil)

func NewAccount(base authtypes.AccountI, creator []byte) *Account {
	h := sha256.Sum256(creator)
	return &Account{AccountI: base, creatorHash: h[:20]}
}

func (acc Account) GetAddress() sdk.AccAddress {
	return acc.creatorHash
}
