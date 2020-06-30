module github.com/datachainlab/cmd/fabibc

go 1.14

require (
	github.com/datachainlab/fabric-ibc v0.0.0-20200627140743-0bd5b1d26ddf
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a // indirect
	github.com/hyperledger/fabric-contract-api-go v1.1.0
	github.com/hyperledger/fabric-protos-go v0.0.0-20200506201313-25f6564b9ac4 // indirect
)

replace (
	github.com/cosmos/cosmos-sdk => github.com/datachainlab/cosmos-sdk v0.34.4-0.20200623040429-5c9fe2c0d7e5
	github.com/datachainlab/fabric-ibc => ../../
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
)
