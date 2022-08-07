MODULES=chaincode light-client simapp

all: test build

go.work:
	@go work init && go work use ${MODULES}

build: go.work
	@go build ./simapp/cmd/fabibc

test: go.work
	@for m in $(MODULES); do \
		FABRIC_IBC_MSPS_DIR=${PWD}/tests/fixtures/organizations/peerOrganizations go test ./$$m/...;\
	done

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
