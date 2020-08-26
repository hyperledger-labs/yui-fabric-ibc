package chaincode

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (c *IBCChaincode) RunMsg(stub shim.ChaincodeStubInterface, txBytes []byte) ([]abci.Event, error) {
	return c.runner.RunMsg(stub, txBytes)
}

func (c *IBCChaincode) GetIgnoredFunctions() []string {
	return []string{"RunMsg"}
}
