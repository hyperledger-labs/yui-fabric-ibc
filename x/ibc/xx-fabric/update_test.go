package fabric

import (
	"bytes"
	"testing"

	"github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
)

func Test_updateMSPConfigs(t *testing.T) {
	type args struct {
		clientState ClientState
		mspConfigs  types.MSPConfigs
	}
	tests := []struct {
		name string
		args args
		want ClientState
	}{
		// TODO: Add test cases.
		{"valid case",
			args{
				clientState: ClientState{
					LastMSPInfos: types.MSPInfos{
						Infos: []types.MSPInfo{
							{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1")},
							{MSPID: "MSPID2", Config: nil, Policy: []byte{}},
							{MSPID: "MSPID3", Config: nil, Policy: []byte("policy3")},
							{MSPID: "MSPID4", Config: []byte("config4"), Policy: []byte{}},
						},
					},
				},
				mspConfigs: types.MSPConfigs{
					Configs: []types.MSPConfig{
						{Type: types.TypeUpdate, MSPID: "MSPID1", Config: []byte("config1_update"), Proof: &types.MessageProof{}},
						{Type: types.TypeCreate, MSPID: "MSPID3", Config: []byte("config3"), Proof: &types.MessageProof{}},
					},
				},
			},
			ClientState{
				LastMSPInfos: types.MSPInfos{
					Infos: []types.MSPInfo{
						{MSPID: "MSPID1", Config: []byte("config1_update"), Policy: []byte("policy1")}, // updated
						{MSPID: "MSPID2", Config: nil, Policy: []byte{}},
						{MSPID: "MSPID3", Config: []byte("config3"), Policy: []byte("policy3")}, // created
						{MSPID: "MSPID4", Config: []byte("config4"), Policy: []byte{}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateMSPConfigs(tt.args.clientState, tt.args.mspConfigs)
			if len(got.LastMSPInfos.Infos) != len(tt.want.LastMSPInfos.Infos) {
				t.Errorf("updateMSPConfigs().LastMSPInfos.Infos len(got)= %v, len(want) %v", len(got.LastMSPInfos.Infos), len(tt.want.LastMSPInfos.Infos))
			}
			for i, info := range tt.want.LastMSPInfos.Infos {
				gInfo := got.LastMSPInfos.Infos[i]
				if gInfo.MSPID != info.MSPID || !bytes.Equal(gInfo.Config, info.Config) || !bytes.Equal(gInfo.Policy, info.Policy) {
					t.Errorf("difference info index = %v", i)
				}
			}
		})
	}
}

func Test_updateMSPPolicies(t *testing.T) {
	type args struct {
		clientState ClientState
		mspPolicies types.MSPPolicies
	}
	tests := []struct {
		name string
		args args
		want ClientState
	}{
		// TODO: Add test cases.
		{"valid case",
			args{
				clientState: ClientState{
					LastMSPInfos: types.MSPInfos{
						Infos: []types.MSPInfo{
							{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1")},
							{MSPID: "MSPID3", Config: nil, Policy: []byte("policy3")},
						},
					},
				},
				mspPolicies: types.MSPPolicies{
					Policies: []types.MSPPolicy{
						{Type: types.TypeUpdate, MSPID: "MSPID1", Policy: []byte("policy1_update"), Proof: &types.MessageProof{}},
						{Type: types.TypeCreate, MSPID: "MSPID2", Policy: []byte("policy2"), Proof: &types.MessageProof{}},
						{Type: types.TypeCreate, MSPID: "MSPID4", Policy: []byte("policy4"), Proof: &types.MessageProof{}},
					},
				},
			},
			ClientState{
				LastMSPInfos: types.MSPInfos{
					Infos: []types.MSPInfo{
						{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1_update")}, // updated
						{MSPID: "MSPID2", Config: nil, Policy: []byte("policy2")},                      // created
						{MSPID: "MSPID3", Config: nil, Policy: []byte("policy3")},
						{MSPID: "MSPID4", Config: nil, Policy: []byte("policy4")}, // created
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateMSPPolicies(tt.args.clientState, tt.args.mspPolicies)
			if len(got.LastMSPInfos.Infos) != len(tt.want.LastMSPInfos.Infos) {
				t.Errorf("updateMSPPolicies().LastMSPInfos.Infos len(got)= %v, len(want) %v", len(got.LastMSPInfos.Infos), len(tt.want.LastMSPInfos.Infos))
			}
			for i, info := range tt.want.LastMSPInfos.Infos {
				gInfo := got.LastMSPInfos.Infos[i]
				if gInfo.MSPID != info.MSPID || !bytes.Equal(gInfo.Config, info.Config) || !bytes.Equal(gInfo.Policy, info.Policy) {
					t.Errorf("difference info index = %v", i)
				}
			}
		})
	}
}
