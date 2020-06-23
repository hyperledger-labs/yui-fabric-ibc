module github.com/datachainlab/fabric-ibc

go 1.14

require (
	github.com/VictoriaMetrics/fastcache v1.5.7 // indirect
	github.com/confio/ics23/go v0.0.0-20200604202538-6e2c36a74465
	github.com/cosmos/cosmos-sdk v0.34.4-0.20200622203133-4716260a6e2d
	github.com/fsouza/go-dockerclient v1.6.5 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/mux v1.7.4
	github.com/hyperledger/fabric v1.4.0-rc1.0.20200416031218-eff2f9306191
	github.com/hyperledger/fabric-amcl v0.0.0-20200424173818-327c9e2cf77a // indirect
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-config v0.0.0-20200514142724-e1bf69d9f3fe // indirect
	github.com/hyperledger/fabric-contract-api-go v1.0.0
	github.com/hyperledger/fabric-lib-go v1.0.0 // indirect
	github.com/hyperledger/fabric-protos-go v0.0.0-20200506201313-25f6564b9ac4
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/sykesm/zap-logfmt v0.0.3 // indirect
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00 // indirect
	github.com/tendermint/tendermint v0.33.5
	github.com/tendermint/tm-db v0.5.1
	github.com/willf/bitset v1.1.10 // indirect
	google.golang.org/grpc v1.29.1
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4

replace github.com/cosmos/cosmos-sdk => github.com/datachainlab/cosmos-sdk v0.34.4-0.20200623040429-5c9fe2c0d7e5
