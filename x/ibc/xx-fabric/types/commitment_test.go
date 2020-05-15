package types

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/policydsl"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	msptesttools "github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	tmtime "github.com/tendermint/tendermint/types/time"
)

func TestMSPSetup(t *testing.T) {
	assert := assert.New(t)

	// setup the MSP manager so that we can sign/verify
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		assert.FailNow(err.Error())
	}
	cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	if err != nil {
		assert.FailNow(err.Error())
	}
	lcMSP := mspmgmt.GetLocalMSP(cryptoProvider)
	signer, err := lcMSP.GetDefaultSigningIdentity()
	if err != nil {
		assert.FailNow(err.Error())
	}

	mm := mspmgmt.NewDeserializersManager(cryptoProvider).GetLocalDeserializer()

	pr := &Proof{}
	{
		result := &rwset.TxReadWriteSet{
			DataModel: rwset.TxReadWriteSet_KV,
			NsRwset: []*rwset.NsReadWriteSet{
				{
					Namespace: "lscc",
					Rwset: MarshalOrPanic(&kvrwset.KVRWSet{
						Writes: []*kvrwset.KVWrite{
							{
								Key:   "commitment/{channel}/{port}/{seq}",
								Value: []byte("true"),
							},
						},
					}),
				},
			},
		}
		bz, err := proto.Marshal(result)
		if err != nil {
			assert.FailNow(err.Error())
		}
		res, err := makeProposalResponse(signer, bz)
		if err != nil {
			assert.FailNow(err.Error())
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
		pr.NSIndex = 0
		pr.WriteSetIndex = 0
	}

	if err := pr.ValidateBasic(); err != nil {
		assert.FailNow(err.Error())
	}

	var mspids []string
	mspid, err := lcMSP.GetIdentifier()
	if err != nil {
		assert.FailNow(err.Error())
	}
	mspids = append(
		mspids,
		mspid,
	)
	clientState := makeClientState(mspids)

	var p pb.ApplicationPolicy
	if err := proto.Unmarshal(clientState.LastChaincodeInfo.PolicyBytes, &p); err != nil {
		assert.FailNow(err.Error())
	}
	ap := p.Type.(*pb.ApplicationPolicy_SignaturePolicy)

	var sigSet []*protoutil.SignedData
	for i := 0; i < len(pr.Signatures); i++ {
		msg := make([]byte, len(pr.Proposal)+len(pr.Identities[i]))
		copy(msg[:len(pr.Proposal)], pr.Proposal)
		copy(msg[len(pr.Proposal):], pr.Identities[i])

		sigSet = append(
			sigSet,
			&protoutil.SignedData{
				Data:      msg,
				Identity:  pr.Identities[i],
				Signature: pr.Signatures[i],
			},
		)
	}

	pp := cauthdsl.EnvelopeBasedPolicyProvider{Deserializer: mm}
	policy, err := pp.NewPolicy(ap.SignaturePolicy)
	if err != nil {
		assert.FailNow(err.Error())
	}
	if err := policy.EvaluateSignedData(sigSet); err != nil {
		assert.FailNow(err.Error())
	}

	// TODO unmarshal and validate each fields: ChannelID, ChaincodeID, Namespace
	var payload pb.ProposalResponsePayload
	if err := proto.Unmarshal(pr.Proposal, &payload); err != nil {
		assert.FailNow(err.Error())
	}
	var cact pb.ChaincodeAction
	if err := proto.Unmarshal(payload.Extension, &cact); err != nil {
		assert.FailNow(err.Error())
	}
	var result rwset.TxReadWriteSet
	if err := proto.Unmarshal(cact.Results, &result); err != nil {
		assert.FailNow(err.Error())
	}
	targetKey := "commitment/{channel}/{port}/{seq}"
	ok, err := EnsureWriteSetIncludesCommitment(result.GetNsRwset(), pr.NSIndex, pr.WriteSetIndex, targetKey, []byte("true"))
	if err != nil {
		assert.FailNow(err.Error())
	}
	assert.True(ok)
}

func makeClientState(mspids []string) ClientState {
	policy := protoutil.MarshalOrPanic(&common.ApplicationPolicy{
		Type: &common.ApplicationPolicy_SignaturePolicy{
			SignaturePolicy: policydsl.SignedByNOutOfGivenRole(int32(len(mspids)/2+1), msppb.MSPRole_MEMBER, mspids),
		},
	})
	cs := ClientState{
		LastChaincodeHeader: ChaincodeHeader{
			Sequence:  1,
			Timestamp: tmtime.Now(),
			Proof:     Proof{}, // TODO fix
		},
		LastChaincodeInfo: ChaincodeInfo{
			ChannelID: "dummyChannel",
			ChaincodeID: pb.ChaincodeID{
				Name:    "dummyCC",
				Version: "dummyVer",
			},
			PolicyBytes: policy,
			Signatures:  nil, // TODO fix
		},
	}
	return cs
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
