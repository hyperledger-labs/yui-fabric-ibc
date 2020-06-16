package commitment

import (
	"fmt"

	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
)

func MakeCurrentSequenceKey(prefix commitmentexported.Prefix) string {
	return fmt.Sprintf("h/k:%v/current", string(prefix.Bytes()))
}

func MakeSequenceKey(prefix commitmentexported.Prefix, seq uint64) string {
	return fmt.Sprintf("h/k:%v/%v", string(prefix.Bytes()), seq)
}

func MakeSequenceCommitmentKey(seq uint64) string {
	return fmt.Sprintf("h/_/%v/commitment", seq)
}

func MakeEntryKey(prefix commitmentexported.Prefix, key string) string {
	return fmt.Sprintf("e/k:%v/%v", string(prefix.Bytes()), key)
}
