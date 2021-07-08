package types

import (
	"fmt"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/hyperledger/fabric/protoutil"
)

var _ exported.Proof = CommitmentProof{}

// VerifyMembership implements ProofI.
func (CommitmentProof) VerifyMembership([]*ics23.ProofSpec, exported.Root, exported.Path, []byte) error {
	return nil
}

// VerifyNonMembership implements ProofI.
func (CommitmentProof) VerifyNonMembership([]*ics23.ProofSpec, exported.Root, exported.Path) error {
	return nil
}

// IsEmpty returns trie if the signature is emtpy.
func (proof CommitmentProof) Empty() bool {
	return len(proof.Signatures) == 0
}

// ValidateBasic checks if the proof is empty.
func (proof CommitmentProof) ValidateBasic() error {
	a := len(proof.Identities)
	b := len(proof.Signatures)
	if a != b {
		return fmt.Errorf("mismatch length: %v != %v", a, b)
	}
	return nil
}

// ToSignedData returns SignedData slice built with CommitmentProof
func (proof CommitmentProof) ToSignedData() []*protoutil.SignedData {
	var sigSet []*protoutil.SignedData
	for i := 0; i < len(proof.Signatures); i++ {
		msg := make([]byte, len(proof.Proposal)+len(proof.Identities[i]))
		copy(msg[:len(proof.Proposal)], proof.Proposal)
		copy(msg[len(proof.Proposal):], proof.Identities[i])

		sigSet = append(
			sigSet,
			&protoutil.SignedData{
				Data:      msg,
				Identity:  proof.Identities[i],
				Signature: proof.Signatures[i],
			},
		)
	}
	return sigSet
}

var _ exported.Prefix = (*Prefix)(nil)

// NewPrefix constructs new Prefix instance
func NewPrefix(value []byte) Prefix {
	return Prefix{
		Value: value,
	}
}

// Bytes returns the key prefix bytes
func (fp Prefix) Bytes() []byte {
	return fp.Value
}

// Empty returns true if the prefix is empty
func (fp Prefix) Empty() bool {
	return len(fp.Bytes()) == 0
}
