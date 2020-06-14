package types

import (
	"fmt"

	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
)

var _ clientexported.Header = (*Header)(nil)

func NewHeader(cheader ChaincodeHeader, cinfo ChaincodeInfo) Header {
	return Header{ChaincodeHeader: &cheader, ChaincodeInfo: &cinfo}
}

func (h Header) GetHeight() uint64 {
	return uint64(h.ChaincodeHeader.Sequence)
}

func (h Header) ClientType() clientexported.ClientType {
	return Fabric
}

func NewChaincodeHeader(seq int64, timestamp uint64, proof Proof) ChaincodeHeader {
	return ChaincodeHeader{
		Sequence:  seq,
		Timestamp: timestamp,
		Proof:     proof,
	}
}

func (h ChaincodeHeader) GetEndorseBytes() []byte {
	h2 := ChaincodeHeader{
		Sequence:  h.Sequence,
		Timestamp: h.Timestamp,
	}
	return ModuleCdc.MustMarshalJSON(h2)
}

func (h ChaincodeHeader) ValidateBasic() error {
	return nil
}

func NewChaincodeInfo(chanID string, ccID ChaincodeID, policyBytes []byte, sigs [][]byte) ChaincodeInfo {
	return ChaincodeInfo{
		ChannelId:         chanID,
		ChaincodeId:       ccID,
		EndorsementPolicy: policyBytes,
		Signatures:        sigs,
	}
}

func (ci ChaincodeInfo) ValidateBasic() error {
	return nil
}

func (ci ChaincodeInfo) ChainID() string {
	return fmt.Sprintf("%v/%v", ci.ChannelId, ci.ChaincodeId.String())
}
