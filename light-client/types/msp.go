package types

import (
	"errors"

	"github.com/golang/protobuf/proto"
	msppb "github.com/hyperledger/fabric-protos-go/msp"
)

// XXX we need better alias name
type MSPPBConfig = msppb.MSPConfig

func NewMSPInfo(mspID string, config, policy []byte) MSPInfo {
	return MSPInfo{
		MSPID:  mspID,
		Config: config,
		Policy: policy,
	}
}

func (mi MSPInfos) GetMSPPBConfigs() ([]MSPPBConfig, error) {
	configs := []MSPPBConfig{}
	for _, mi := range mi.Infos {
		// freezed MSPInfo is skipped
		if mi.Freezed {
			continue
		}
		if mi.Config == nil {
			// valid MSPInfo should have a config
			return nil, errors.New("a MSPInfo has no config")
		}
		var mspConfig MSPPBConfig
		if err := proto.Unmarshal(mi.Config, &mspConfig); err != nil {
			return nil, err
		}
		configs = append(configs, mspConfig)
	}
	return configs, nil
}

// return true whether the target MSPInfo is freezed or not
func (mi MSPInfos) HasMSPID(mspID string) bool {
	idx := indexOfMSPID(mi, mspID)
	return idx != -1
}

func (mi MSPInfos) IndexOf(mspID string) int {
	idx := indexOfMSPID(mi, mspID)
	return idx
}

func (mi MSPInfos) FindMSPInfo(mspID string) (*MSPInfo, error) {
	idx := indexOfMSPID(mi, mspID)
	if idx < 0 {
		return nil, errors.New("MSPInfo not found")
	}
	return &mi.Infos[idx], nil
}

// assume header.ValidateBasic() == nil
func generateMSPInfos(header Header) (*MSPInfos, error) {
	var infos MSPInfos
	for _, mh := range header.MSPHeaders.Headers {
		if mh.Type != MSPHeaderTypeCreate {
			// TODO we should return an error if got an unexpected header?
			continue
		}
		infos.Infos = append(infos.Infos, MSPInfo{
			MSPID:   mh.MSPID,
			Config:  mh.Config,
			Policy:  mh.Policy,
			Freezed: false,
		})
	}
	return &infos, nil
}

// return the index of the mspID or -1 if mspID is not present.
func indexOfMSPID(mi MSPInfos, mspID string) int {
	for i, info := range mi.Infos {
		if info.MSPID == mspID {
			return i
		}
	}
	return -1
}
