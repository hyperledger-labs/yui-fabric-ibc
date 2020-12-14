package fabric

import (
	"bytes"
	"testing"

	"github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric/types"
)

func Test_updateMSPInfos(t *testing.T) {
	type args struct {
		clientState ClientState
		mspHeaders  types.MSPHeaders
	}
	tests := []struct {
		name string
		args args
		want ClientState
	}{
		// TODO: Add test cases.
		{"new MSPInfos",
			args{
				clientState: ClientState{LastMSPInfos: types.MSPInfos{Infos: []types.MSPInfo{}}},
				mspHeaders: types.MSPHeaders{Headers: []types.MSPHeader{
					{Type: types.MSPHeaderTypeCreate, MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1"), Proof: &types.MessageProof{}},
					{Type: types.MSPHeaderTypeCreate, MSPID: "MSPID2", Config: []byte("config2"), Policy: []byte("policy2"), Proof: &types.MessageProof{}},
				}},
			},
			ClientState{
				LastMSPInfos: types.MSPInfos{Infos: []types.MSPInfo{
					{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1"), Freezed: false},
					{MSPID: "MSPID2", Config: []byte("config2"), Policy: []byte("policy2"), Freezed: false},
				}},
			},
		},
		{"all MSPHeaderType",
			args{
				clientState: ClientState{LastMSPInfos: types.MSPInfos{Infos: []types.MSPInfo{
					{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1"), Freezed: false},
					{MSPID: "MSPID3", Config: []byte("config3"), Policy: []byte("policy3"), Freezed: false},
					{MSPID: "MSPID4", Config: []byte("config4"), Policy: []byte("policy4"), Freezed: false},
				}}},
				mspHeaders: types.MSPHeaders{Headers: []types.MSPHeader{
					{Type: types.MSPHeaderTypeUpdatePolicy, MSPID: "MSPID1", Config: nil, Policy: []byte("policy1_update"), Proof: &types.MessageProof{}},
					{Type: types.MSPHeaderTypeCreate, MSPID: "MSPID2", Config: []byte("config2"), Policy: []byte("policy2"), Proof: &types.MessageProof{}},
					{Type: types.MSPHeaderTypeFreeze, MSPID: "MSPID3", Config: nil, Policy: nil, Proof: &types.MessageProof{}},
					{Type: types.MSPHeaderTypeUpdateConfig, MSPID: "MSPID4", Config: []byte("config4_update"), Policy: nil, Proof: &types.MessageProof{}},
				}},
			},
			ClientState{
				LastMSPInfos: types.MSPInfos{Infos: []types.MSPInfo{
					{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1_update"), Freezed: false}, // updated
					{MSPID: "MSPID2", Config: []byte("config2"), Policy: []byte("policy2"), Freezed: false},        // created
					{MSPID: "MSPID3", Config: []byte("config3"), Policy: []byte("policy3"), Freezed: true},         // freezed
					{MSPID: "MSPID4", Config: []byte("config4_update"), Policy: []byte("policy4"), Freezed: false}, // updated
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateMSPInfos(tt.args.clientState, tt.args.mspHeaders)
			if len(got.LastMSPInfos.Infos) != len(tt.want.LastMSPInfos.Infos) {
				t.Errorf("updateMSPInfos().LastMSPInfos.Infos len(got)= %v, len(want) %v", len(got.LastMSPInfos.Infos), len(tt.want.LastMSPInfos.Infos))
			}
			for i, info := range tt.want.LastMSPInfos.Infos {
				gInfo := got.LastMSPInfos.Infos[i]
				if gInfo.MSPID != info.MSPID || !bytes.Equal(gInfo.Config, info.Config) || !bytes.Equal(gInfo.Policy, info.Policy) || gInfo.Freezed != info.Freezed {
					t.Errorf("difference info index = %v, got = %v, want = %v", i, gInfo, info)
				}
			}
		})
	}
}
