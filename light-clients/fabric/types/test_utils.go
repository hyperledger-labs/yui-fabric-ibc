package types

import (
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode/commitment"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	tmtime "github.com/tendermint/tendermint/types/time"
)

type MSPTestFixture struct {
	MSPID   string
	MSPConf *msppb.MSPConfig
	MSP     msp.MSP
	Signer  msp.SigningIdentity
}

func getMSPFixture(mspsDir string, mspID string) (*MSPTestFixture, error) {
	mspConf, err := getLocalVerifyingMspConfig(mspsDir, mspID)
	if err != nil {
		return nil, err
	}
	msp, err := getLocalMsp(mspsDir, mspID)
	if err != nil {
		return nil, err
	}
	signer, err := msp.GetDefaultSigningIdentity()
	if err != nil {
		return nil, err
	}

	return &MSPTestFixture{
		MSPID:   mspID,
		MSPConf: mspConf,
		MSP:     msp,
		Signer:  signer,
	}, nil
}

func GetLocalMspConfigForTest(mspsDir string, mspID string) (*msppb.MSPConfig, *factory.FactoryOpts, error) {
	bccspConf := getDefaultBccspConfig()
	mconf, err := getLocalMspConfigWithBccspConfig(mspsDir, mspID, bccspConf)
	return mconf, bccspConf, err
}

// return MSPConfig with no signing identity
func getLocalVerifyingMspConfig(mspsDir string, mspID string) (*msppb.MSPConfig, error) {
	mconf, _, err := GetLocalMspConfigForTest(mspsDir, mspID)
	if err != nil {
		return nil, err
	}
	err = GetVerifyingConfig(mconf)
	if err != nil {
		return nil, err
	}
	return mconf, nil
}

func getLocalMsp(mspsDir string, mspID string) (msp.MSP, error) {
	mspConf, bccspConf, err := GetLocalMspConfigForTest(mspsDir, mspID)
	if err != nil {
		return nil, err
	}
	return SetupLocalMspForTest(mspConf, bccspConf)
}

func SetupLocalMspForTest(mspConf *msppb.MSPConfig, bccspConf *factory.FactoryOpts) (msp.MSP, error) {
	cryptoProvider, err := (&factory.SWFactory{}).Get(bccspConf)
	if err != nil {
		return nil, err
	}
	opts := msp.Options[msp.ProviderTypeToString(msp.FABRIC)]
	m, err := msp.New(opts, cryptoProvider)
	if err != nil {
		return nil, err
	}
	if err := m.Setup(mspConf); err != nil {
		return nil, err
	}
	return m, nil
}

// get MSPConfig for verifying
func GetVerifyingConfig(mconf *msppb.MSPConfig) error {
	var conf msppb.FabricMSPConfig
	err := proto.Unmarshal(mconf.Config, &conf)
	if err != nil {
		return err
	}
	conf.SigningIdentity = nil
	confb, err := proto.Marshal(&conf)
	if err != nil {
		return err
	}
	mconf.Config = confb
	return nil
}

func getLocalMspConfigWithBccspConfig(mspsDir string, mspID string, bccspConfig *factory.FactoryOpts) (*msppb.MSPConfig, error) {
	dir := filepath.Join(mspsDir, mspID)
	return msp.GetLocalMspConfig(dir, bccspConfig, mspID)
}

func getDefaultBccspConfig() *factory.FactoryOpts {
	return factory.GetDefaultOpts()
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

func MakeCommitmentProofForTest(signer protoutil.Signer, key string, value []byte) (*CommitmentProof, error) {
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
