package types

import "fmt"

func VerifyChaincodeHeaderPath(seq int64) string {
	return fmt.Sprintf("/verify/header/%d", seq)
}
