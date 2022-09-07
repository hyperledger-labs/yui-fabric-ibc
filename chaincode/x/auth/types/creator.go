package types

import (
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/msp"
)

func MakeCreatorAddressWithSerializedIdentity(sid *msp.SerializedIdentity) (sdk.AccAddress, error) {
	bz, err := proto.Marshal(sid)
	if err != nil {
		return nil, err
	}
	return MakeCreatorAddressWithBytes(bz)
}

func MakeCreatorAddressWithBytes(creator []byte) (sdk.AccAddress, error) {
	h := sha256.Sum256(creator)
	return h[:20], nil
}
