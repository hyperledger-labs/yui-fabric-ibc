package commitment

import (
	"fmt"

	"github.com/cosmos/ibc-go/modules/core/exported"
)

func MakeCurrentSequenceKey(prefix exported.Prefix) string {
	return fmt.Sprintf("h/k:%v/current", string(prefix.Bytes()))
}

func MakeSequenceKey(prefix exported.Prefix, seq uint64) string {
	return fmt.Sprintf("h/k:%v/seq/%v", string(prefix.Bytes()), seq)
}

func MakeEntryKey(prefix exported.Prefix, key string) string {
	return fmt.Sprintf("e/k:%v/%v", string(prefix.Bytes()), key)
}
