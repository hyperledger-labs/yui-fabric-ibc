package fabric

import (
	"reflect"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := updateMSPConfigs(tt.args.clientState, tt.args.mspConfigs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("updateMSPConfigs() = %v, want %v", got, tt.want)
			}
		})
	}
}
