package testing

import (
	"fmt"
	"strconv"
	"strings"

	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"github.com/datachainlab/fabric-ibc/chaincode"
	"github.com/datachainlab/fabric-ibc/commitment"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func queryEndorseCommitment(ctx contractapi.TransactionContextInterface, cc *chaincode.IBCChaincode, key []byte) (*commitment.CommitmentEntry, error) {
	k := string(key)
	parts := strings.Split(k, "/")

	if strings.HasSuffix(k, "/"+string(host.KeyClientState())) {
		clientID := parts[1]
		return cc.EndorseClientState(ctx, clientID)
	} else if strings.HasPrefix(k, string(host.KeyConnectionPrefix)+"/") {
		connectionID := parts[1]
		return cc.EndorseConnectionState(ctx, connectionID)
	} else if strings.HasPrefix(k, string(host.KeyClientStorePrefix)) && strings.Contains(k, string(host.KeyConsensusStatesPrefix)+"/") {
		clientID := parts[1]
		height, err := clienttypes.ParseHeight(parts[3])
		if err != nil {
			return nil, err
		}
		return cc.EndorseConsensusStateCommitment(ctx, clientID, height.VersionHeight)
	} else if strings.HasPrefix(k, host.KeyChannelPrefix+"/") {
		portID := parts[2]
		channelID := parts[4]
		return cc.EndorseChannelState(ctx, portID, channelID)
	} else if strings.HasPrefix(k, host.KeyPacketCommitmentPrefix+"/") {
		portID := parts[2]
		channelID := parts[4]
		seq, err := strconv.Atoi(parts[6])
		if err != nil {
			return nil, err
		}
		return cc.EndorsePacketCommitment(ctx, portID, channelID, uint64(seq))
	} else if strings.HasPrefix(k, host.KeyPacketAckPrefix+"/") {
		portID := parts[2]
		channelID := parts[4]
		seq, err := strconv.Atoi(parts[6])
		if err != nil {
			return nil, err
		}
		return cc.EndorsePacketAcknowledgement(ctx, portID, channelID, uint64(seq))
	} else {
		return nil, fmt.Errorf("unknown key: '%v'", k)
	}
}
