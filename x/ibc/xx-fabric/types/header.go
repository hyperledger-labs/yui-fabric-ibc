package types

import (
	"fmt"

	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
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

func NewChaincodeHeader(seq uint64, timestamp int64, proof CommitmentProof) ChaincodeHeader {
	return ChaincodeHeader{
		Sequence: commitment.NewSequence(seq, timestamp),
		Proof:    proof,
	}
}

func (h ChaincodeHeader) ValidateBasic() error {
	return nil
}

func NewChaincodeInfo(chanID string, ccID ChaincodeID, endorsementPolicy, ibcPolicy []byte, proof *MessageProof) ChaincodeInfo {
	return ChaincodeInfo{
		ChannelId:         chanID,
		ChaincodeId:       ccID,
		EndorsementPolicy: endorsementPolicy,
		IbcPolicy:         ibcPolicy,
		Proof:             proof,
	}
}

func (ci ChaincodeInfo) ValidateBasic() error {
	return nil
}

func (ci ChaincodeInfo) GetChainID() string {
	return fmt.Sprintf("%v/%v", ci.ChannelId, ci.ChaincodeId.String())
}

func (ci ChaincodeInfo) GetFabricChaincodeID() peer.ChaincodeID {
	return peer.ChaincodeID{
		Path:    ci.ChaincodeId.Path,
		Name:    ci.ChaincodeId.Name,
		Version: ci.ChaincodeId.Version,
	}
}

func (ci ChaincodeInfo) GetSignBytes() []byte {
	ci.Proof = nil
	bz, err := proto.Marshal(&ci)
	if err != nil {
		panic(err)
	}
	return bz
}
