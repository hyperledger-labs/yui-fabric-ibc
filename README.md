# Fabric-IBC

## What's this?

Fabric-IBC includes the Chaincode and Cosmos modules that enable IBC between Fabric and Cosmos/Tendermint, or between Fabric and Fabric. Of course, it'll be able to communicate with other Blockchains and DLTs that support IBC!

## Development

Currently, x/ibc/client module in cosmos-sdk cannot use a fabric client implemented in external modules without forking the source code. Therefore, we have copied and modified the following modules from cosmos-sdk.

- x/ibc/02-client(https://github.com/cosmos/cosmos-sdk/blob/b5a658729170e577b0e6e809e73d5e0756b90e07/x/ibc/02-client): Excludes x/ibc/02-client/{client,exported,simulation,types}
- x/ibc(https://github.com/cosmos/cosmos-sdk/blob/b5a658729170e577b0e6e809e73d5e0756b90e07/x/ibc): Excludes x/ibc/xx-fabric

In addition, we have rewritten the import path for each cosmos modules from `github.com/cosmos/cosmos/cosmos-sdk/x/ibc/02-client` to `github.com/datachainlab/fabric-ibc/x/ibc/02-client`.
