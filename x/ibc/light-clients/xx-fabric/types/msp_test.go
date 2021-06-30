package types

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	fabrictests "github.com/hyperledger-labs/yui-fabric-ibc/x/ibc/light-clients/xx-fabric/tests"
	"github.com/stretchr/testify/require"
)

func TestMSPInfos_GetMSPPBConfigs(t *testing.T) {
	conf := configForTest()
	org1, err := fabrictests.GetMSPFixture(conf.MSPsDir, "Org1MSP")
	require.NoError(t, err)
	org1MspConf, err := proto.Marshal(org1.MSPConf)
	require.NoError(t, err)
	org2, err := fabrictests.GetMSPFixture(conf.MSPsDir, "Org2MSP")
	require.NoError(t, err)

	type fields struct {
		Infos []MSPInfo
	}
	tests := []struct {
		name    string
		fields  fields
		want    []MSPPBConfig
		wantErr bool
	}{
		{"no msp",
			fields{Infos: []MSPInfo{}},
			[]MSPPBConfig{},
			false},
		{"valid case",
			fields{Infos: []MSPInfo{
				{MSPID: org1.MSPID, Config: org1MspConf, Policy: makePolicy([]string{org1.MSPID}), Freezed: false},
			}},
			[]MSPPBConfig{*org1.MSPConf},
			false},
		{"skipping freezed MSPInfo",
			fields{Infos: []MSPInfo{
				{MSPID: org1.MSPID, Config: org1MspConf, Policy: makePolicy([]string{org1.MSPID}), Freezed: false},
				{MSPID: org2.MSPID, Config: []byte("config2"), Policy: makePolicy([]string{org1.MSPID}), Freezed: true},
			}},
			[]MSPPBConfig{*org1.MSPConf},
			false},
		{"error with nil config",
			fields{Infos: []MSPInfo{
				{MSPID: org1.MSPID, Config: org1MspConf, Policy: makePolicy([]string{org1.MSPID})},
				{MSPID: org2.MSPID, Config: nil, Policy: makePolicy([]string{org1.MSPID})},
			}},
			[]MSPPBConfig{},
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mi := MSPInfos{
				Infos: tt.fields.Infos,
			}
			got, err := mi.GetMSPPBConfigs()
			if (err != nil) != tt.wantErr {
				t.Errorf("MSPInfos.GetMSPPBConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("MSPInfos.GetMSPPBConfigs() len(got) = %v, len(want) %v", len(got), len(tt.want))
				return
			}
			for i, want := range tt.want {
				if !bytes.Equal(want.Config, got[i].Config) {
					t.Errorf("MSPInfos.GetMSPPBConfigs() different config for index %v", i)
				}
			}
		})
	}
}

func TestMSPInfos_HasMSPID(t *testing.T) {
	mi := MSPInfos{Infos: []MSPInfo{
		{MSPID: "MSPID1", Config: nil, Policy: nil, Freezed: false},
		{MSPID: "MSPID2", Config: nil, Policy: nil, Freezed: false},
		{MSPID: "MSPID3", Config: nil, Policy: nil, Freezed: true},
	}}
	type args struct {
		mspID string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"true for valid case", args{"MSPID1"}, true},
		{"true for freezed", args{"MSPID3"}, true},
		{"false for case sensitive", args{"mspid2"}, false},
		{"false for not found", args{"MSPID4"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mi.HasMSPID(tt.args.mspID); got != tt.want {
				t.Errorf("MSPInfos.HasMSPID() = %v, want %v", got, tt.want)
			}
		})
	}
}
