package testing

import (
	"fmt"
	"strconv"
	"strings"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/hyperledger-labs/yui-fabric-ibc/chaincode"
	"github.com/hyperledger-labs/yui-fabric-ibc/commitment"
	"github.com/hyperledger-labs/yui-fabric-ibc/tests"
	fabrictypes "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/types"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func endorseCommitment(ctx contractapi.TransactionContextInterface, chain *TestChain, entry *commitment.CommitmentEntry) (*fabrictypes.CommitmentProof, error) {
	e, err := commitment.EntryFromCommitment(entry)
	if err != nil {
		return nil, err
	}
	return tests.MakeCommitmentProof(chain.endorser, e.Key, e.Value)
}

func queryEndorseCommitment(ctx contractapi.TransactionContextInterface, chain *TestChain, key []byte) ([]byte, error) {
	e, err := queryCommitment(ctx, chain.CC, key)
	if err != nil {
		return nil, err
	}
	c, err := endorseCommitment(ctx, chain, e)
	if err != nil {
		return nil, err
	}
	return chain.App.AppCodec().Marshal(c)
}

func queryCommitment(ctx contractapi.TransactionContextInterface, cc *chaincode.IBCChaincode, key []byte) (*commitment.CommitmentEntry, error) {
	k := string(key)
	parts := strings.Split(k, "/")

	if strings.HasSuffix(k, "/"+string(host.ClientStateKey())) {
		clientID := parts[1]
		return cc.EndorseClientState(ctx, clientID)
	} else if strings.HasPrefix(k, string(host.KeyConnectionPrefix)+"/") {
		connectionID := parts[1]
		return cc.EndorseConnectionState(ctx, connectionID)
	} else if strings.HasPrefix(k, string(host.KeyClientStorePrefix)) && strings.Contains(k, string(host.KeyConsensusStatePrefix)+"/") {
		clientID := parts[1]
		height, err := clienttypes.ParseHeight(parts[3])
		if err != nil {
			return nil, err
		}
		return cc.EndorseConsensusStateCommitment(ctx, clientID, height.RevisionHeight)
	} else if strings.HasPrefix(k, host.KeyChannelEndPrefix+"/") {
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
