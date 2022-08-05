build:
	@go build ./simapp/cmd/fabibc

test:
	export FABRIC_IBC_MSPS_DIR=${PWD}/tests/fixtures/organizations/peerOrganizations \
	&& go test ./chaincode/... ./light-client/... ./simapp/...

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
