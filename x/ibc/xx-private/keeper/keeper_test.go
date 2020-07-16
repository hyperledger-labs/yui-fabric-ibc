package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShieldedData(t *testing.T) {
	var cases = []struct {
		key  string
		data []byte
	}{
		{key: "key", data: []byte("value")},
		{key: "key", data: []byte("prefix/value")},
		{key: "key", data: []byte("")},
	}

	for i, cs := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			assert := assert.New(t)

			sdata := MakeShieldedData(cs.key, cs.data)
			key, h, err := ParseShieldedData(sdata)
			assert.NoError(err)
			assert.Equal(cs.key, key)
			assert.Equal(hash(cs.data), h)
		})
	}
}
