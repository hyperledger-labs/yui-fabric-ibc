package types

import (
	"errors"
	"fmt"
	"strings"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	"github.com/hyperledger/fabric-protos-go/peer"
)

var _ exported.Header = (*Header)(nil)

func NewHeader(cheader *ChaincodeHeader, cinfo *ChaincodeInfo, mheaders *MSPHeaders) *Header {
	return &Header{ChaincodeHeader: cheader, ChaincodeInfo: cinfo, MSPHeaders: mheaders}
}

func (h Header) GetHeight() exported.Height {
	return clienttypes.NewHeight(0, h.ChaincodeHeader.Sequence.Value)
}

func (h Header) ClientType() string {
	return Fabric
}

func (h Header) ValidateBasic() error {
	if h.ChaincodeHeader == nil && h.ChaincodeInfo == nil && h.MSPHeaders == nil {
		return errors.New("either ChaincodeHeader, ChaincodeInfo or MSPHeaders must be non-nil value")
	}
	if h.ChaincodeHeader != nil {
		if err := h.ChaincodeHeader.ValidateBasic(); err != nil {
			return err
		}
	}
	if h.ChaincodeInfo != nil {
		if err := h.ChaincodeInfo.ValidateBasic(); err != nil {
			return err
		}
	}
	if h.MSPHeaders != nil {
		if err := h.MSPHeaders.ValidateBasic(); err != nil {
			return err
		}
	}
	return nil
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

func NewMSPHeader(mhType MSPHeaderType, mspID string, config []byte, policy []byte, proof *MessageProof) MSPHeader {
	return MSPHeader{
		Type:   mhType,
		MSPID:  mspID,
		Config: config,
		Policy: policy,
		Proof:  proof,
	}
}

func (mh MSPHeader) ValidateBasic() error {
	if mh.MSPID == "" {
		return errors.New("MSPID is empty")
	}

	var validConfigPolicy bool
	switch mh.Type {
	case MSPHeaderTypeCreate:
		validConfigPolicy = (mh.Config != nil && mh.Policy != nil)
	case MSPHeaderTypeUpdatePolicy:
		validConfigPolicy = (mh.Config == nil && mh.Policy != nil)
	case MSPHeaderTypeUpdateConfig:
		validConfigPolicy = (mh.Config != nil && mh.Policy == nil)
	case MSPHeaderTypeFreeze:
		validConfigPolicy = (mh.Config == nil && mh.Policy == nil)
	}
	if !validConfigPolicy {
		return errors.New("config or policy is invalid")
	}
	if mh.Proof == nil {
		return errors.New("proof is empty")
	}
	return nil
}

func (mh MSPHeader) GetSignBytes() []byte {
	mh.Proof = nil
	bz, err := proto.Marshal(&mh)
	if err != nil {
		panic(err)
	}
	return bz
}

func NewMSPHeaders(headers []MSPHeader) MSPHeaders {
	return MSPHeaders{
		Headers: headers,
	}
}

func (mhs MSPHeaders) ValidateBasic() error {
	m := map[string]bool{}
	prevID := ""

	for _, mh := range mhs.Headers {
		if err := mh.ValidateBasic(); err != nil {
			return err
		}
		if m[mh.MSPID] {
			return errors.New("some MSPHeaders are duplicated")
		}
		if CompareMSPID(prevID, mh.MSPID) > 0 {
			return errors.New("MSPID must be sorted by ascending order")
		}
		m[mh.MSPID] = true
		prevID = mh.MSPID
	}
	return nil
}

// returns an integer comparing two MSPIDs.
// the result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func CompareMSPID(a, b string) int {
	return strings.Compare(a, b)
}
