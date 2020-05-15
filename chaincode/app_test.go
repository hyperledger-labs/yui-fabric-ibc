package chaincode

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/datachainlab/fabric-ibc/x/compat"
	fabric "github.com/datachainlab/fabric-ibc/x/ibc/xx-fabric"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	tmtime "github.com/tendermint/tendermint/types/time"
	dbm "github.com/tendermint/tm-db"
)

func TestApp(t *testing.T) {
	assert := assert.New(t)

	logger := log.NewTMLogger(os.Stdout)
	var dbProvider = tmDBProvider
	runner := NewAppRunner(logger, dbProvider)
	stub := compat.MakeFakeStub()
	msg := makeMsgCreateClient()
	assert.NoError(runner.RunMsg(stub, msg))
}

const (
	channelID = "dummyChannel"
	clientID  = "fabricclient"
)

var ccid = peer.ChaincodeID{
	Name:    "dummyCC",
	Version: "dummyVer",
}

func makeMsgCreateClient() string {
	cdc, _ := MakeCodecs()

	/// Build Msg
	ch := fabric.NewChaincodeHeader(1, tmtime.Now(), fabric.Proof{})
	var sigs [][]byte
	var pcBytes []byte
	ci := fabric.NewChaincodeInfo(channelID, ccid, pcBytes, sigs)

	h := fabric.NewHeader(ch, ci)
	signer := sdk.AccAddress("signer0")
	msg := fabric.NewMsgCreateClient(clientID, h, signer)
	if err := msg.ValidateBasic(); err != nil {
		panic(err)
	}
	/// END

	tx := auth.StdTx{
		Msgs: []sdk.Msg{msg},
	}

	bz, err := cdc.MarshalJSON(tx)
	if err != nil {
		panic(err)
	}
	return string(bz)
}

func tmDBProvider(_ shim.ChaincodeStubInterface) dbm.DB {
	return dbm.NewMemDB()
}
