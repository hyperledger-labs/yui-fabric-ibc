package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

var (
	_ clientexported.MsgCreateClient = MsgCreateClient{}
	_ clientexported.MsgUpdateClient = MsgUpdateClient{}
)

// MsgCreateClient defines a message to create an IBC client
type MsgCreateClient struct {
	ClientID string         `json:"client_id" yaml:"client_id"`
	Header   Header         `json:"header" yaml:"header"`
	Signer   sdk.AccAddress `json:"address" yaml:"address"`
}

// this is a constant to satisfy the linter
const TODO = "TODO"

// dummy implementation of proto.Message
func (msg MsgCreateClient) Reset()         {}
func (msg MsgCreateClient) String() string { return TODO }
func (msg MsgCreateClient) ProtoMessage()  {}

// NewMsgCreateClient creates a new MsgCreateClient instance
func NewMsgCreateClient(
	id string, header Header, signer sdk.AccAddress,
) MsgCreateClient {

	return MsgCreateClient{
		ClientID: id,
		Header:   header,
		Signer:   signer,
	}
}

// Route implements sdk.Msg
func (msg MsgCreateClient) Route() string {
	return host.RouterKey
}

// Type implements sdk.Msg
func (msg MsgCreateClient) Type() string {
	// return clientexported.TypeMsgCreateClient
	return "create_client"
}

// ValidateBasic implements sdk.Msg
func (msg MsgCreateClient) ValidateBasic() error {
	return host.ClientIdentifierValidator(msg.ClientID)
}

// GetSignBytes implements sdk.Msg
func (msg MsgCreateClient) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners implements sdk.Msg
func (msg MsgCreateClient) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Signer}
}

// GetClientID implements clientexported.MsgCreateClient
func (msg MsgCreateClient) GetClientID() string {
	return msg.ClientID
}

// GetClientType implements clientexported.MsgCreateClient
func (msg MsgCreateClient) GetClientType() string {
	return ClientTypeFabric
}

// GetConsensusState implements clientexported.MsgCreateClient
func (msg MsgCreateClient) GetConsensusState() clientexported.ConsensusState {
	// Construct initial consensus state from provided Header
	return ConsensusState{
		Timestamp: msg.Header.ChaincodeHeader.Sequence.Timestamp,
		Height:    msg.Header.ChaincodeHeader.Sequence.Value,
	}
}

// MsgUpdateClient defines a message to update an IBC client
type MsgUpdateClient struct {
	ClientID string         `json:"client_id" yaml:"client_id"`
	Header   Header         `json:"header" yaml:"header"`
	Signer   sdk.AccAddress `json:"address" yaml:"address"`
}

// dummy implementation of proto.Message
func (msg MsgUpdateClient) Reset()         {}
func (msg MsgUpdateClient) String() string { return TODO }
func (msg MsgUpdateClient) ProtoMessage()  {}

// NewMsgUpdateClient creates a new MsgUpdateClient instance
func NewMsgUpdateClient(id string, header Header, signer sdk.AccAddress) MsgUpdateClient {
	return MsgUpdateClient{
		ClientID: id,
		Header:   header,
		Signer:   signer,
	}
}

// Route implements sdk.Msg
func (msg MsgUpdateClient) Route() string {
	return host.RouterKey
}

// Type implements sdk.Msg
func (msg MsgUpdateClient) Type() string {
	// return clientexported.TypeMsgUpdateClient
	return "update_client"
}

// ValidateBasic implements sdk.Msg
func (msg MsgUpdateClient) ValidateBasic() error {
	if msg.Signer.Empty() {
		return sdkerrors.ErrInvalidAddress
	}
	return host.ClientIdentifierValidator(msg.ClientID)
}

// GetSignBytes implements sdk.Msg
func (msg MsgUpdateClient) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners implements sdk.Msg
func (msg MsgUpdateClient) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Signer}
}

// GetClientID implements clientexported.MsgUpdateClient
func (msg MsgUpdateClient) GetClientID() string {
	return msg.ClientID
}

// GetHeader implements clientexported.MsgUpdateClient
func (msg MsgUpdateClient) GetHeader() clientexported.Header {
	return msg.Header
}
