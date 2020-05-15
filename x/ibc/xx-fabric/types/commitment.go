package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
)

const (
	TypeFabric exported.Type = 100 // dummy
)

var _ exported.Proof = Proof{}

type Proof struct {
	Proposal      []byte
	NSIndex       uint32
	WriteSetIndex uint32
	Identities    [][]byte
	Signatures    [][]byte // signatures of endorsers. This order must be equals consensState.endorsers order.
}

// GetCommitmentType implements ProofI.
func (Proof) GetCommitmentType() exported.Type {
	return TypeFabric
}

// VerifyMembership implements ProofI.
func (Proof) VerifyMembership(exported.Root, exported.Path, []byte) error {
	return nil
}

// VerifyNonMembership implements ProofI.
func (Proof) VerifyNonMembership(exported.Root, exported.Path) error {
	return nil
}

// IsEmpty returns trie if the signature is emtpy.
func (proof Proof) IsEmpty() bool {
	return len(proof.Signatures) == 0
}

// ValidateBasic checks if the proof is empty.
func (proof Proof) ValidateBasic() error {
	a := len(proof.Identities)
	b := len(proof.Signatures)
	if a != b {
		return fmt.Errorf("mismatch length: %v != %v", a, b)
	}
	return nil
}
