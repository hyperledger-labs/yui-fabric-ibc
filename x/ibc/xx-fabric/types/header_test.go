package types

import "testing"

func TestMSPPolicies_ValidateBasic(t *testing.T) {
	type fields struct {
		Policies []MSPPolicy
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"invalid1", fields{Policies: []MSPPolicy{{ID: "MSP1", Policy: []byte("policy"), Proof: nil}}}, true},
		{"invalid2", fields{Policies: []MSPPolicy{{ID: "MSP1", Policy: nil, Proof: &MessageProof{}}}}, true},
		{"ids are sorted", fields{Policies: []MSPPolicy{
			{ID: "MSP1", Policy: []byte("policy1"), Proof: &MessageProof{}},
			{ID: "MSP2", Policy: []byte("policy2"), Proof: &MessageProof{}},
		}}, false},
		{"ids are unsorted", fields{Policies: []MSPPolicy{
			{ID: "MSP2", Policy: []byte("policy2"), Proof: &MessageProof{}},
			{ID: "MSP1", Policy: []byte("policy1"), Proof: &MessageProof{}},
		}}, true},
		{"duplicated id", fields{Policies: []MSPPolicy{
			{ID: "MSP1", Policy: []byte("policy1"), Proof: &MessageProof{}},
			{ID: "MSP2", Policy: []byte("policy2a"), Proof: &MessageProof{}},
			{ID: "MSP2", Policy: []byte("policy2b"), Proof: &MessageProof{}},
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
