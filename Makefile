all: test build yrly

.PHONY: build
build:
	@go build -o build/fabibc ./simapp/cmd/fabibc

.PHONY: test
test:
	FABRIC_IBC_MSPS_DIR=${PWD}/tests/fixtures/organizations/peerOrganizations go test ./...

yrly:
	@go build -o build/yrly ./relay/bin

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
