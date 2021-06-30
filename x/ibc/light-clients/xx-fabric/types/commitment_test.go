package types

import (
	"path/filepath"
	"testing"

	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	fabrictests "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/tests"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/require"
	tmtime "github.com/tendermint/tendermint/types/time"
)

func TestCommitment(t *testing.T) {
	require := require.New(t)

	// setup the MSP manager so that we can sign/verify
	config, err := DefaultConfig()
	require.NoError(err)
	lcMSP, err := fabrictests.GetLocalMsp(config.MSPsDir, "SampleOrgMSP")
	require.NoError(err)
	signer, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(err)

	targetKey := "commitment/{channel}/{port}/{seq}"
	targetValue := []byte("true")

	pr, err := makeCommitmentProof(signer, targetKey, targetValue)
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

	deserializer, err := loadVerifyingMsps(config)
	require.NoError(err)

	pp := cauthdsl.EnvelopeBasedPolicyProvider{Deserializer: deserializer}
	policy, err := pp.NewPolicy(ap.SignaturePolicy)
	require.NoError(err)
	require.NoError(policy.EvaluateSignedData(sigSet))

	// TODO unmarshal and validate each fields: ChannelID, ChaincodeID, Namespace
	var payload pb.ProposalResponsePayload
	require.NoError(proto.Unmarshal(pr.Proposal, &payload))

	var cact pb.ChaincodeAction
	require.NoError(proto.Unmarshal(payload.Extension, &cact))

	result := &rwsetutil.TxRwSet{}
	require.NoError(result.FromProtoBytes(cact.Results))

	ok, err := ensureWriteSetIncludesCommitment(result.NsRwSets, pr.NsIndex, pr.WriteSetIndex, targetKey, targetValue)
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
			Sequence: commitment.NewSequence(1, tmtime.Now().UnixNano()),
			Proof:    CommitmentProof{}, // TODO fix
		},
		LastChaincodeInfo: ChaincodeInfo{
			ChannelId: "dummyChannel",
			ChaincodeId: ChaincodeID{
				Name:    "dummyCC",
				Version: "dummyVer",
			},
			EndorsementPolicy: policy,
			Proof:             nil, // TODO fix
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

func makeCommitmentProof(signer protoutil.Signer, key string, value []byte) (*CommitmentProof, error) {
	pr := &CommitmentProof{}
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

func MarshalOrPanic(msg proto.Message) []byte {
	b, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func loadVerifyingMsps(conf Config) (msp.MSPManager, error) {
	msps := []msp.MSP{}
	// if in local, this would depend `peer.localMspType` config
	for _, id := range conf.MSPIDs {
		dir := filepath.Join(conf.MSPsDir, id)
		bccspConfig := factory.GetDefaultOpts()
		mspConf, err := msp.GetVerifyingMspConfig(dir, id, msp.ProviderTypeToString(msp.FABRIC))
		if err != nil {
			return nil, err
		}
		opts := msp.Options[msp.ProviderTypeToString(msp.FABRIC)]
		cryptoProvider, err := (&factory.SWFactory{}).Get(bccspConfig)
		if err != nil {
			return nil, err
		}
		m, err := msp.New(opts, cryptoProvider)
		if err != nil {
			return nil, err
		}
		if err := m.Setup(mspConf); err != nil {
			return nil, err
		}
		msps = append(msps, m)
	}
	mgr := msp.NewMSPManager()
	err := mgr.Setup(msps)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}
