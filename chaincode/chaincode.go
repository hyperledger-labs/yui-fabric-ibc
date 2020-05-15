package chaincode

import (
	"os"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/tendermint/tendermint/libs/log"
)

type IBCChaincode struct {
	contractapi.Contract
	runner AppRunner
}

func (c *IBCChaincode) HandleIBCMsg(ctx contractapi.TransactionContextInterface, msgJSON string) error {
	return c.runner.RunMsg(ctx.GetStub(), msgJSON)
}

func NewIBCChaincode() *IBCChaincode {
	logger := log.NewTMLogger(os.Stdout)
	runner := NewAppRunner(logger, DefaultDBProvider)
	c := &IBCChaincode{
		runner: runner,
	}
	return c
}
