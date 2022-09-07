package main

import (
	"log"

	fabric "github.com/hyperledger-labs/yui-fabric-ibc/relay/module"
	tendermint "github.com/hyperledger-labs/yui-relayer/chains/tendermint/module"
	"github.com/hyperledger-labs/yui-relayer/cmd"
	mock "github.com/hyperledger-labs/yui-relayer/provers/mock/module"
)

func main() {
	if err := cmd.Execute(
		tendermint.Module{},
		fabric.Module{},
		mock.Module{},
	); err != nil {
		log.Fatal(err)
	}
}
