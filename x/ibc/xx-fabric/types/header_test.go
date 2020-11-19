package types

import (
	"testing"
)

func TestHeader_TargetsSameMSPs(t *testing.T) {
	type fields struct {
		MSPConfigs  *MSPConfigs
		MSPPolicies *MSPPolicies
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"msps and types are same", fields{
			MSPConfigs: &MSPConfigs{
				Configs: []MSPConfig{
					{Type: TypeUpdate, MSPID: "MSPID1", Config: nil, Proof: nil},
					{Type: TypeUpdate, MSPID: "MSPID2", Config: nil, Proof: nil},
				},
			},
			MSPPolicies: &MSPPolicies{
				Policies: []MSPPolicy{
					{Type: TypeUpdate, MSPID: "MSPID1", Policy: nil, Proof: nil},
					{Type: TypeUpdate, MSPID: "MSPID2", Policy: nil, Proof: nil},
				},
			},
		}, true},
		{"msps are not same", fields{
			MSPConfigs: &MSPConfigs{
				Configs: []MSPConfig{
					{Type: TypeCreate, MSPID: "MSPID1", Config: nil, Proof: nil},
					{Type: TypeCreate, MSPID: "MSPID2", Config: nil, Proof: nil},
				},
			},
			MSPPolicies: &MSPPolicies{
				Policies: []MSPPolicy{
					{Type: TypeCreate, MSPID: "MSPID1", Policy: nil, Proof: nil},
					{Type: TypeCreate, MSPID: "MSPID3", Policy: nil, Proof: nil},
				},
			},
		}, false},
		{"order are not same", fields{
			MSPConfigs: &MSPConfigs{
				Configs: []MSPConfig{
					{Type: TypeCreate, MSPID: "MSPID1", Config: nil, Proof: nil},
					{Type: TypeCreate, MSPID: "MSPID2", Config: nil, Proof: nil},
				},
			},
			MSPPolicies: &MSPPolicies{
				Policies: []MSPPolicy{
					{Type: TypeCreate, MSPID: "MSPID2", Policy: nil, Proof: nil},
					{Type: TypeCreate, MSPID: "MSPID1", Policy: nil, Proof: nil},
				},
			},
		}, false},
		{"types are not same", fields{
			MSPConfigs: &MSPConfigs{
				Configs: []MSPConfig{
					{Type: TypeUpdate, MSPID: "MSPID1", Config: nil, Proof: nil},
				},
			},
			MSPPolicies: &MSPPolicies{
				Policies: []MSPPolicy{
					{Type: TypeCreate, MSPID: "MSPID1", Policy: nil, Proof: nil},
				},
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Header{
				ChaincodeHeader: nil,
				ChaincodeInfo:   nil,
				MSPConfigs:      tt.fields.MSPConfigs,
				MSPPolicies:     tt.fields.MSPPolicies,
			}
			if got := h.TargetsSameMSPs(); got != tt.want {
				t.Errorf("Header.TargetsSameMSPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMSPPolicies_ValidateBasic(t *testing.T) {
	type fields struct {
		Policies []MSPPolicy
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"invalid1", fields{Policies: []MSPPolicy{{MSPID: "MSP1", Policy: []byte("policy"), Proof: nil}}}, true},
		{"invalid2", fields{Policies: []MSPPolicy{{MSPID: "MSP1", Policy: nil, Proof: &MessageProof{}}}}, true},
		{"MSPIDs are sorted", fields{Policies: []MSPPolicy{
			{MSPID: "MSP1", Policy: []byte("policy1"), Proof: &MessageProof{}},
			{MSPID: "MSP2", Policy: []byte("policy2"), Proof: &MessageProof{}},
		}}, false},
		{"MSPIDs are unsorted", fields{Policies: []MSPPolicy{
			{MSPID: "MSP2", Policy: []byte("policy2"), Proof: &MessageProof{}},
			{MSPID: "MSP1", Policy: []byte("policy1"), Proof: &MessageProof{}},
		}}, true},
		{"MSPIDs are duplicated", fields{Policies: []MSPPolicy{
			{MSPID: "MSP1", Policy: []byte("policy1"), Proof: &MessageProof{}},
			{MSPID: "MSP2", Policy: []byte("policy2a"), Proof: &MessageProof{}},
			{MSPID: "MSP2", Policy: []byte("policy2b"), Proof: &MessageProof{}},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mps := MSPPolicies{
				Policies: tt.fields.Policies,
			}
			if err := mps.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("MSPPolicies.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMSPConfigs_ValidateBasic(t *testing.T) {
	type fields struct {
		Configs []MSPConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"invalid1", fields{Configs: []MSPConfig{{Type: TypeCreate, MSPID: "MSP1", Config: []byte("policy"), Proof: nil}}}, true},
		{"invalid2", fields{Configs: []MSPConfig{{Type: TypeCreate, MSPID: "MSP1", Config: nil, Proof: &MessageProof{}}}}, true},
		{"MSPIDs are sorted", fields{Configs: []MSPConfig{
			{Type: TypeCreate, MSPID: "MSPID1", Config: []byte("config"), Proof: &MessageProof{}},
			{Type: TypeCreate, MSPID: "MSPID2", Config: []byte("config"), Proof: &MessageProof{}},
			{Type: TypeCreate, MSPID: "MSPID3", Config: []byte("config"), Proof: &MessageProof{}},
		}}, false},
		{"MSPIDs are unsorted in ascending order", fields{Configs: []MSPConfig{
			{Type: TypeCreate, MSPID: "MSPID3", Config: []byte("config"), Proof: &MessageProof{}},
			{Type: TypeCreate, MSPID: "MSPID2", Config: []byte("config"), Proof: &MessageProof{}},
			{Type: TypeCreate, MSPID: "MSPID1", Config: []byte("config"), Proof: &MessageProof{}},
		}}, true},
		{"MSPIDs are duplicated", fields{Configs: []MSPConfig{
			{Type: TypeCreate, MSPID: "MSPID1", Config: []byte("config"), Proof: &MessageProof{}},
			{Type: TypeCreate, MSPID: "MSPID2", Config: []byte("config1"), Proof: &MessageProof{}},
			{Type: TypeCreate, MSPID: "MSPID2", Config: []byte("config2"), Proof: &MessageProof{}},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcs := MSPConfigs{
				Configs: tt.fields.Configs,
			}
			if err := mcs.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("MSPConfigs.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
