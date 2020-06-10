package types

import (
	"fmt"
	"time"

	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	"github.com/hyperledger/fabric-protos-go/peer"
)

var _ clientexported.Header = (*Header)(nil)

type Header struct {
	ChaincodeHeader *ChaincodeHeader `json:"chaincode_header" yaml:"chaincode_header"`
	ChaincodeInfo   *ChaincodeInfo   `json:"chaincode_info" yaml:"chaincode_info"`
}

func NewHeader(cheader ChaincodeHeader, cinfo ChaincodeInfo) Header {
	return Header{ChaincodeHeader: &cheader, ChaincodeInfo: &cinfo}
}

func (h Header) GetHeight() uint64 {
	return uint64(h.ChaincodeHeader.Sequence)
}

func (h Header) ClientType() clientexported.ClientType {
	return Fabric
}

// every transactor can update this via endorsement
type ChaincodeHeader struct {
	Sequence  int64
	Timestamp time.Time
	Proof     Proof
}

func NewChaincodeHeader(seq int64, timestamp time.Time, proof Proof) ChaincodeHeader {
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

type ChaincodeInfo struct {
	ChannelID   string
	ChaincodeID peer.ChaincodeID
	PolicyBytes []byte
	Signatures  [][]byte // signatures are created by last endorsers
}

func NewChaincodeInfo(chanID string, ccID peer.ChaincodeID, policyBytes []byte, sigs [][]byte) ChaincodeInfo {
	return ChaincodeInfo{
		ChannelID:   chanID,
		ChaincodeID: ccID,
		PolicyBytes: policyBytes,
		Signatures:  sigs,
	}
}

func (ci ChaincodeInfo) ValidateBasic() error {
	return nil
}

func (ci ChaincodeInfo) ChainID() string {
	return fmt.Sprintf("%v/%v", ci.ChannelID, ci.ChaincodeID.String())
}
