module github.com/hyperledger-labs/yui-fabric-ibc

go 1.15

require (
	github.com/VictoriaMetrics/fastcache v1.5.7 // indirect
	github.com/confio/ics23/go v0.6.3
	github.com/cosmos/cosmos-sdk v0.40.0-rc3
	github.com/fsouza/go-dockerclient v1.6.5 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.15.2
	github.com/hyperledger/fabric v1.4.0-rc1.0.20200416031218-eff2f9306191
	github.com/hyperledger/fabric-amcl v0.0.0-20200424173818-327c9e2cf77a // indirect
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200511190512-bcfeb58dd83a
	github.com/hyperledger/fabric-contract-api-go v1.0.0
	github.com/hyperledger/fabric-protos-go v0.0.0-20200506201313-25f6564b9ac4
	github.com/regen-network/cosmos-proto v0.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/sykesm/zap-logfmt v0.0.3 // indirect
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00 // indirect
	github.com/tendermint/tendermint v0.34.0-rc6
	github.com/tendermint/tm-db v0.6.2
	google.golang.org/genproto v0.0.0-20201111145450-ac7456db90a6 // indirect
	google.golang.org/grpc v1.33.2
	google.golang.org/protobuf v1.25.0
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
