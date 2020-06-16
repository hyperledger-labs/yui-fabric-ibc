package types

import (
	"fmt"

	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
)

var _ clientexported.Header = (*Header)(nil)

func NewHeader(cheader ChaincodeHeader, cinfo ChaincodeInfo) Header {
	return Header{ChaincodeHeader: &cheader, ChaincodeInfo: &cinfo}
}

func (h Header) GetHeight() uint64 {
	return h.ChaincodeHeader.Sequence.Value
}

func (h Header) ClientType() clientexported.ClientType {
	return Fabric
}

func NewChaincodeHeader(seq uint64, timestamp int64, proof Proof) ChaincodeHeader {
	return ChaincodeHeader{
		Sequence: commitment.NewSequence(seq, timestamp),
		Proof:    proof,
	}
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
