package types

import (
	"bytes"
	"testing"
)

func Test_updateMSPInfos(t *testing.T) {
	type args struct {
		clientState ClientState
		mspHeaders  MSPHeaders
	}
	tests := []struct {
		name string
		args args
		want ClientState
	}{
		// TODO: Add test cases.
		{"new MSPInfos",
			args{
				clientState: ClientState{LastMspInfos: MSPInfos{Infos: []MSPInfo{}}},
				mspHeaders: MSPHeaders{Headers: []MSPHeader{
					{Type: MSPHeaderTypeCreate, MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1"), Proof: &MessageProof{}},
					{Type: MSPHeaderTypeCreate, MSPID: "MSPID2", Config: []byte("config2"), Policy: []byte("policy2"), Proof: &MessageProof{}},
				}},
			},
			ClientState{
				LastMspInfos: MSPInfos{Infos: []MSPInfo{
					{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1"), Freezed: false},
					{MSPID: "MSPID2", Config: []byte("config2"), Policy: []byte("policy2"), Freezed: false},
				}},
			},
		},
		{"all MSPHeaderType",
			args{
				clientState: ClientState{LastMspInfos: MSPInfos{Infos: []MSPInfo{
					{MSPID: "MSPID1", Config: []byte("config1"), Policy: []byte("policy1"), Freezed: false},
					{MSPID: "MSPID3", Config: []byte("config3"), Policy: []byte("policy3"), Freezed: false},
					{MSPID: "MSPID4", Config: []byte("config4"), Policy: []byte("policy4"), Freezed: false},
				}}},
				mspHeaders: MSPHeaders{Headers: []MSPHeader{
					{Type: MSPHeaderTypeUpdatePolicy, MSPID: "MSPID1", Config: nil, Policy: []byte("policy1_update"), Proof: &MessageProof{}},
					{Type: MSPHeaderTypeCreate, MSPID: "MSPID2", Config: []byte("config2"), Policy: []byte("policy2"), Proof: &MessageProof{}},
					{Type: MSPHeaderTypeFreeze, MSPID: "MSPID3", Config: nil, Policy: nil, Proof: &MessageProof{}},
					{Type: MSPHeaderTypeUpdateConfig, MSPID: "MSPID4", Config: []byte("config4_update"), Policy: nil, Proof: &MessageProof{}},
				}},
			},
			ClientState{
				LastMspInfos: MSPInfos{Infos: []MSPInfo{
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
			if len(got.LastMspInfos.Infos) != len(tt.want.LastMspInfos.Infos) {
				t.Errorf("updateMSPInfos().LastMspInfos.Infos len(got)= %v, len(want) %v", len(got.LastMspInfos.Infos), len(tt.want.LastMspInfos.Infos))
			}
			for i, info := range tt.want.LastMspInfos.Infos {
				gInfo := got.LastMspInfos.Infos[i]
				if gInfo.MSPID != info.MSPID || !bytes.Equal(gInfo.Config, info.Config) || !bytes.Equal(gInfo.Policy, info.Policy) || gInfo.Freezed != info.Freezed {
					t.Errorf("difference info index = %v, got = %v, want = %v", i, gInfo, info)
				}
			}
		})
	}
}
