package chaincode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	abci "github.com/tendermint/tendermint/abci/types"
)

// TODO make key prefixes configurable
const (
	packetEventKeyPrefix = "/packets"
)

// QueryPacket returns a packet that matches given arguments
func (c *IBCChaincode) QueryPacket(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (string, error) {
	k := makePacketKey(packetEventKeyPrefix, portID, channelID, sequence)
	bz, err := ctx.GetStub().GetState(k)
	if err != nil {
		return "", err
	} else if bz == nil {
		return "", fmt.Errorf("key not found: %v", k)
	}
	var p channeltypes.Packet
	if err := p.Unmarshal(bz); err != nil {
		return "", err
	}
	bz, err = json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(bz), nil
}

// HandlePacketEvent handles events and put packets which are parsed from events on state DB
func HandlePacketEvent(ctx contractapi.TransactionContextInterface, events []abci.Event) (continue_ bool, err error) {
	packets, err := getPacketsFromEvents(events)
	if err != nil {
		return true, nil
	}

	for _, p := range packets {
		k := makePacketKey(packetEventKeyPrefix, p.SourcePort, p.SourceChannel, p.Sequence)
		bz, err := p.Marshal()
		if err != nil {
			return false, err
		}
		if err := ctx.GetStub().PutState(k, bz); err != nil {
			return false, err
		}
	}

	return true, nil
}

func makePacketKey(keyPrefix string, portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%v/%v/%v/%v", keyPrefix, portID, channelID, sequence)
}

func getPacketsFromEvents(events []abci.Event) ([]channeltypes.Packet, error) {
	var packets []channeltypes.Packet
	sevs := sdk.StringifyEvents(events)
	for _, ev := range sevs {
		if ev.Type == channeltypes.EventTypeSendPacket {
			// NOTE: Attributes of packet are included in one event.
			var packet channeltypes.Packet
			for _, attr := range ev.Attributes {
				switch attr.Key {
				case channeltypes.AttributeKeyData:
					// AttributeKeyData key indicates a start of packet attributes
					packet = channeltypes.Packet{}
					packet.Data = []byte(attr.Value)
				case channeltypes.AttributeKeyTimeoutHeight:
					parts := strings.Split(attr.Value, "-")
					packet.TimeoutHeight = clienttypes.NewHeight(
						strToUint64(parts[0]),
						strToUint64(parts[1]),
					)
				case channeltypes.AttributeKeyTimeoutTimestamp:
					packet.TimeoutTimestamp = strToUint64(attr.Value)
				case channeltypes.AttributeKeySequence:
					packet.Sequence = strToUint64(attr.Value)
				case channeltypes.AttributeKeySrcPort:
					packet.SourcePort = attr.Value
				case channeltypes.AttributeKeySrcChannel:
					packet.SourceChannel = attr.Value
				case channeltypes.AttributeKeyDstPort:
					packet.DestinationPort = attr.Value
				case channeltypes.AttributeKeyDstChannel:
					packet.DestinationChannel = attr.Value
				case channeltypes.AttributeKeyChannelOrdering:
					// AttributeKeyChannelOrdering key indicates the end of packet atrributes
					if err := packet.ValidateBasic(); err != nil {
						return nil, err
					}
					packets = append(packets, packet)
				}
			}
		}
	}
	return packets, nil
}

func strToUint64(s string) uint64 {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return uint64(v)
}
