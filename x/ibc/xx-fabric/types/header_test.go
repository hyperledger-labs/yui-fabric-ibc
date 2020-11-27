package types

import (
	"testing"
)

func TestMSPHeaders_ValidateBasic(t *testing.T) {
	type fields struct {
		Headers []MSPHeader
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"no proof", fields{[]MSPHeader{
			{Type: MSPHeaderTypeCreate, MSPID: "MSP1", Config: []byte("config"), Policy: []byte("policy"), Proof: nil},
		}}, true},
		{"no config for create", fields{[]MSPHeader{
			{Type: MSPHeaderTypeCreate, MSPID: "MSP1", Config: nil, Policy: []byte("policy"), Proof: &MessageProof{}},
		}}, true},
		{"no policy for create", fields{[]MSPHeader{
			{Type: MSPHeaderTypeCreate, MSPID: "MSP1", Config: []byte("config"), Policy: nil, Proof: &MessageProof{}},
		}}, true},
		{"no config for config update", fields{[]MSPHeader{
			{Type: MSPHeaderTypeUpdateConfig, MSPID: "MSP1", Config: nil, Policy: nil, Proof: &MessageProof{}},
		}}, true},
		{"no policy for policy update", fields{[]MSPHeader{
			{Type: MSPHeaderTypeUpdatePolicy, MSPID: "MSP1", Config: nil, Policy: nil, Proof: &MessageProof{}},
		}}, true},
		{"valid for delete", fields{[]MSPHeader{
			{Type: MSPHeaderTypeFreeze, MSPID: "MSP1", Config: nil, Policy: nil, Proof: &MessageProof{}},
		}}, false},
		{"MSPIDs are sorted", fields{[]MSPHeader{
			{Type: MSPHeaderTypeCreate, MSPID: "MSP1", Config: []byte("config1"), Policy: []byte("policy1"), Proof: &MessageProof{}},
			{Type: MSPHeaderTypeUpdateConfig, MSPID: "MSP2", Config: []byte("config2_update"), Policy: nil, Proof: &MessageProof{}},
		}}, false},
		{"MSPIDs must be sorted", fields{[]MSPHeader{
			{Type: MSPHeaderTypeUpdateConfig, MSPID: "MSP2", Config: []byte("config2_update"), Policy: nil, Proof: &MessageProof{}},
			{Type: MSPHeaderTypeCreate, MSPID: "MSP1", Config: []byte("config1"), Policy: []byte("policy1"), Proof: &MessageProof{}},
		}}, true},
		{"invalid for duplicate MSPID", fields{[]MSPHeader{
			{Type: MSPHeaderTypeCreate, MSPID: "MSP1", Config: []byte("config1"), Policy: []byte("policy1"), Proof: &MessageProof{}},
			{Type: MSPHeaderTypeCreate, MSPID: "MSP2", Config: []byte("config2"), Policy: []byte("policy2a"), Proof: &MessageProof{}},
			{Type: MSPHeaderTypeUpdatePolicy, MSPID: "MSP2", Config: nil, Policy: []byte("policy2b"), Proof: &MessageProof{}},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mps := MSPHeaders{
				Headers: tt.fields.Headers,
			}
			if err := mps.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("MSPPolicies.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
