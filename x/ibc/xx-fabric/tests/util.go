package tests

import (
	"path/filepath"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/msp"
)

func GetLocalMsp(mspsDir string, mspID string) (msp.MSP, error) {
	dir := filepath.Join(mspsDir, mspID)
	bccspConfig := factory.GetDefaultOpts()
	mspConf, err := msp.GetLocalMspConfig(dir, bccspConfig, mspID)
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
	return m, nil
}
