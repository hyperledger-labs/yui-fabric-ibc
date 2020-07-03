build:
	@go build ./cmd/fabibc

test:
	FABRIC_IBC_MSPS_DIR=${PWD}/tests/fixtures/organizations/peerOrganizations go test ./...

###############################################################################
###                                Protobuf                                 ###
###############################################################################

proto-gen:
	@./scripts/protocgen.sh

proto-update-deps:
	# Copy from https://github.com/cosmos/cosmos-sdk/blob/65ea305336c0da689ecf5f8c864d0f2e0370c71e/Makefile#L291

.PHONY: cryptogen
cryptogen:
	./scripts/cryptogen.sh
