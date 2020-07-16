package tests

import (
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/protoutil"
)

func MakeProof(signer protoutil.Signer, key string, value []byte) (*fabric.Proof, error) {
	pr := &fabric.Proof{}
	result := &rwset.TxReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsRwset: []*rwset.NsReadWriteSet{
			{
				Namespace: "lscc",
				Rwset: MarshalOrPanic(&kvrwset.KVRWSet{
					Writes: []*kvrwset.KVWrite{
						{
							Key:   key,
							Value: value,
						},
					},
				}),
			},
		},
	}
	bz, err := proto.Marshal(result)
	if err != nil {
		return nil, err
	}
	res, err := makeProposalResponse(signer, bz)
	if err != nil {
		return nil, err
	}

	pr.Signatures = append(
		pr.Signatures,
		res.Endorsement.Signature,
	)
	pr.Identities = append(
		pr.Identities,
		res.Endorsement.Endorser,
	)
	pr.Proposal = res.Payload
	pr.NsIndex = 0
	pr.WriteSetIndex = 0
	if err := pr.ValidateBasic(); err != nil {
		return nil, err
	}
	return pr, nil
}

func makeProposalResponse(signer protoutil.Signer, results []byte) (*pb.ProposalResponse, error) {
	channelID := "dummyChannel"
	ccid := &pb.ChaincodeID{
		Name:    "dummyCC",
		Version: "dummyVer",
	}

	ss, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	prop, _, err := protoutil.CreateChaincodeProposal(
		common.HeaderType_ENDORSER_TRANSACTION,
		channelID,
		&pb.ChaincodeInvocationSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				ChaincodeId: ccid,
			},
		},
		ss,
	)
	if err != nil {
		return nil, err
	}

	res, err := protoutil.CreateProposalResponse(
		prop.Header,
		prop.Payload,
		&pb.Response{Status: 200},
		results,
		nil,
		ccid,
		signer,
	)
	return res, err
}

func MarshalOrPanic(msg proto.Message) []byte {
	b, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
