package main

import (
	"fmt"

	"github.com/datachainlab/fabric-ibc/chaincode"
	"github.com/datachainlab/fabric-ibc/example"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc := chaincode.NewIBCChaincode(example.NewIBCApp, chaincode.DefaultDBProvider)
	chaincode, err := contractapi.NewChaincode(cc)

	if err != nil {
		fmt.Printf("Error create IBC chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting IBC chaincode: %s", err.Error())
	}
}
