package testing

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	fabricauthtypes "github.com/hyperledger-labs/yui-fabric-ibc/x/auth/types"
	"github.com/hyperledger/fabric-protos-go/msp"
)

type Account struct {
	authtypes.AccountI

	creatorAddress []byte
}

var _ authtypes.AccountI = (*Account)(nil)

func NewAccount(base authtypes.AccountI, sid *msp.SerializedIdentity) *Account {
	addr, err := fabricauthtypes.MakeCreatorAddressWithSerializedIdentity(sid)
	if err != nil {
		panic(err)
	}
	return &Account{AccountI: base, creatorAddress: addr}
}

func (acc Account) GetAddress() sdk.AccAddress {
	return acc.creatorAddress
}
