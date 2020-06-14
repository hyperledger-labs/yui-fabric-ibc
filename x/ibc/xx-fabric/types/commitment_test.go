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
	"github.com/stretchr/testify/require"
	tmtime "github.com/tendermint/tendermint/types/time"
)

func TestCommitment(t *testing.T) {
	require := require.New(t)

	// setup the MSP manager so that we can sign/verify
	err := msptesttools.LoadMSPSetupForTesting()
	require.NoError(err)
	cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	require.NoError(err)
	lcMSP := mspmgmt.GetLocalMSP(cryptoProvider)
	signer, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(err)

	mm := mspmgmt.NewDeserializersManager(cryptoProvider).GetLocalDeserializer()

	pr, err := makeProof(signer)
	require.NoError(err)

	var mspids []string
	mspid, err := lcMSP.GetIdentifier()
	require.NoError(err)
	mspids = append(
		mspids,
		mspid,
	)
	clientState := makeClientState(mspids)

	var p pb.ApplicationPolicy
	require.NoError(proto.Unmarshal(clientState.LastChaincodeInfo.EndorsementPolicy, &p))
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
	require.NoError(err)
	require.NoError(policy.EvaluateSignedData(sigSet))

	// TODO unmarshal and validate each fields: ChannelID, ChaincodeID, Namespace
	var payload pb.ProposalResponsePayload
	require.NoError(proto.Unmarshal(pr.Proposal, &payload))

	var cact pb.ChaincodeAction
	require.NoError(proto.Unmarshal(payload.Extension, &cact))

	var result rwset.TxReadWriteSet
	require.NoError(proto.Unmarshal(cact.Results, &result))

	targetKey := "commitment/{channel}/{port}/{seq}"
	ok, err := EnsureWriteSetIncludesCommitment(result.GetNsRwset(), pr.NsIndex, pr.WriteSetIndex, targetKey, []byte("true"))
	require.NoError(err)
	require.True(ok)
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
			Timestamp: uint64(tmtime.Now().UnixNano()),
			Proof:     Proof{}, // TODO fix
		},
		LastChaincodeInfo: ChaincodeInfo{
			ChannelId: "dummyChannel",
			ChaincodeId: ChaincodeID{
				Name:    "dummyCC",
				Version: "dummyVer",
			},
			EndorsementPolicy: policy,
			Signatures:        nil, // TODO fix
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

func makeProof(signer protoutil.Signer) (*Proof, error) {
	pr := &Proof{}
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

func MarshalOrPanic(msg proto.Message) []byte {
	b, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
