package types

import (
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/require"
)

func TestCommitment(t *testing.T) {
	require := require.New(t)

	// setup the MSP manager so that we can sign/verify
	config, err := DefaultConfig()
	require.NoError(err)
	lcMSP, err := getLocalMsp(config.MSPsDir, "SampleOrgMSP")
	require.NoError(err)
	signer, err := lcMSP.GetDefaultSigningIdentity()
	require.NoError(err)

	targetKey := "commitment/{channel}/{port}/{seq}"
	targetValue := []byte("true")

	pr, err := MakeCommitmentProofForTest(signer, targetKey, targetValue)
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
