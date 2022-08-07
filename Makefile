MODULES=chaincode light-client simapp relay

all: test build rly

.PHONY: build
build:
	@go build -o build/fabibc ./simapp/cmd/fabibc

.PHONY: test
test:
	@for m in $(MODULES); do \
		FABRIC_IBC_MSPS_DIR=${PWD}/tests/fixtures/organizations/peerOrganizations go test ./$$m/...;\
	done

rly:
	@go build -o build/rly ./relay/bin

###############################################################################
###                                Protobuf                                 ###
###############################################################################

.PHONY: proto-gen
proto-gen:
	@echo "Generating Protobuf files"
	docker run -v $(CURDIR):/workspace --workdir /workspace tendermintdev/sdk-proto-gen:v0.3 sh ./scripts/protocgen.sh

.PHONY: cryptogen
cryptogen:
	./scripts/cryptogen.sh
