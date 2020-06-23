package types

import (
	"fmt"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
)

const (
	CommitmentTypeFabric exported.Type = 100 // dummy
)

var _ exported.Proof = Proof{}

// GetCommitmentType implements ProofI.
func (Proof) GetCommitmentType() exported.Type {
	return CommitmentTypeFabric
}

// VerifyMembership implements ProofI.
func (Proof) VerifyMembership([]*ics23.ProofSpec, exported.Root, exported.Path, []byte) error {
	return nil
}

// VerifyNonMembership implements ProofI.
func (Proof) VerifyNonMembership([]*ics23.ProofSpec, exported.Root, exported.Path) error {
	return nil
}

// IsEmpty returns trie if the signature is emtpy.
func (proof Proof) Empty() bool {
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

var _ exported.Prefix = (*Prefix)(nil)

// NewPrefix constructs new Prefix instance
func NewPrefix(value []byte) Prefix {
	return Prefix{
		Value: value,
	}
}

// GetCommitmentType implements Prefix interface
func (Prefix) GetCommitmentType() exported.Type {
	return CommitmentTypeFabric
}

// Bytes returns the key prefix bytes
func (fp Prefix) Bytes() []byte {
	return fp.Value
}

// Empty returns true if the prefix is empty
func (fp Prefix) Empty() bool {
	return len(fp.Bytes()) == 0
}
