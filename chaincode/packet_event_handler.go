package chaincode

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	abci "github.com/tendermint/tendermint/abci/types"
)

// TODO make key prefixes configurable
const (
	packetEventKeyPrefix                = "/packets"
	packetAcknowledgementEventKeyPrefix = "/acks"
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

// QueryPacketAcknowledgement returns a packet acknowledgement that matches given arguments
func (c *IBCChaincode) QueryPacketAcknowledgement(ctx contractapi.TransactionContextInterface, portID, channelID string, sequence uint64) (string, error) {
	k := makePacketAcknowledgementKey(packetAcknowledgementEventKeyPrefix, portID, channelID, sequence)
	bz, err := ctx.GetStub().GetState(k)
	if err != nil {
		return "", err
	} else if bz == nil {
		return "", fmt.Errorf("key not found: %v", k)
	}
	return base64.StdEncoding.EncodeToString(bz), nil
}

// HandlePacketEvent handles events and put packets which are parsed from events on state DB
func HandlePacketEvent(ctx contractapi.TransactionContextInterface, events []abci.Event) (continue_ bool, err error) {
	packets, err := getPacketsFromEvents(events)
	if err != nil {
		return false, err
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

// HandlePacketAcknowledgementEvent handles events and put acks which are parsed from events on state DB
func HandlePacketAcknowledgementEvent(ctx contractapi.TransactionContextInterface, events []abci.Event) (continue_ bool, err error) {
	acks, err := getPacketAcknowledgementsFromEvents(events)
	if err != nil {
		return false, err
	}

	for _, a := range acks {
		k := makePacketAcknowledgementKey(packetAcknowledgementEventKeyPrefix, a.dstPortID, a.dstChannelID, a.sequence)
		if err := ctx.GetStub().PutState(k, a.data); err != nil {
			return false, err
		}
	}

	return true, nil
}

func makePacketAcknowledgementKey(keyPrefix string, portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%v/%v/%v/%v", keyPrefix, portID, channelID, sequence)
}

func getPacketsFromEvents(events []abci.Event) ([]channeltypes.Packet, error) {
	var packets []channeltypes.Packet
	for _, ev := range events {
		if ev.Type != channeltypes.EventTypeSendPacket {
			continue
		}
		// NOTE: Attributes of packet are included in one event.
		var (
			packet channeltypes.Packet
			err    error
		)
		for i, attr := range ev.Attributes {
			v := string(attr.Value)
			switch string(attr.Key) {
			case channeltypes.AttributeKeyData:
				// AttributeKeyData key indicates a start of packet attributes
				packet = channeltypes.Packet{}
				packet.Data = []byte(attr.Value)
				err = assertIndex(i, 0)
			case channeltypes.AttributeKeyDataHex:
				var bz []byte
				bz, err = hex.DecodeString(string(attr.Value))
				if err != nil {
					panic(err)
				}
				packet.Data = bz
				err = assertIndex(i, 1)
			case channeltypes.AttributeKeyTimeoutHeight:
				parts := strings.Split(v, "-")
				packet.TimeoutHeight = clienttypes.NewHeight(
					strToUint64(parts[0]),
					strToUint64(parts[1]),
				)
				err = assertIndex(i, 2)
			case channeltypes.AttributeKeyTimeoutTimestamp:
				packet.TimeoutTimestamp = strToUint64(v)
				err = assertIndex(i, 3)
			case channeltypes.AttributeKeySequence:
				packet.Sequence = strToUint64(v)
				err = assertIndex(i, 4)
			case channeltypes.AttributeKeySrcPort:
				packet.SourcePort = v
				err = assertIndex(i, 5)
			case channeltypes.AttributeKeySrcChannel:
				packet.SourceChannel = v
				err = assertIndex(i, 6)
			case channeltypes.AttributeKeyDstPort:
				packet.DestinationPort = v
				err = assertIndex(i, 7)
			case channeltypes.AttributeKeyDstChannel:
				packet.DestinationChannel = v
				err = assertIndex(i, 8)
			}
			if err != nil {
				return nil, err
			}
		}
		if err := packet.ValidateBasic(); err != nil {
			return nil, err
		}
		packets = append(packets, packet)
	}
	return packets, nil
}

type packetAcknowledgement struct {
	srcPortID    string
	srcChannelID string
	dstPortID    string
	dstChannelID string
	sequence     uint64
	data         []byte
}

func getPacketAcknowledgementsFromEvents(events []abci.Event) ([]packetAcknowledgement, error) {
	var acks []packetAcknowledgement
	for _, ev := range events {
		if ev.Type != channeltypes.EventTypeWriteAck {
			continue
		}
		var (
			ack packetAcknowledgement
			err error
		)
		for i, attr := range ev.Attributes {
			v := string(attr.Value)
			switch string(attr.Key) {
			case channeltypes.AttributeKeySequence:
				ack.sequence = strToUint64(v)
				err = assertIndex(i, 4)
			case channeltypes.AttributeKeySrcPort:
				ack.srcPortID = v
				err = assertIndex(i, 5)
			case channeltypes.AttributeKeySrcChannel:
				ack.srcChannelID = v
				err = assertIndex(i, 6)
			case channeltypes.AttributeKeyDstPort:
				ack.dstPortID = v
				err = assertIndex(i, 7)
			case channeltypes.AttributeKeyDstChannel:
				ack.dstChannelID = v
				err = assertIndex(i, 8)
			case channeltypes.AttributeKeyAck:
				ack.data = attr.Value
				err = assertIndex(i, 9)
			}
			if err != nil {
				return nil, err
			}
		}
		acks = append(acks, ack)
	}
	return acks, nil
}

func assertIndex(actual, expected int) error {
	if actual == expected {
		return nil
	} else {
		return fmt.Errorf("%v != %v", actual, expected)
	}
}

func strToUint64(s string) uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}
