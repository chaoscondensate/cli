package ots

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
)

const (
	OperationAppend  byte = 0xf0
	OperationPrepend byte = 0xf1
	OperationSHA256  byte = 0x08
)

type Operation struct {
	Tag      byte
	Argument []byte
}

type AttestationKind string

const (
	AttestationPending AttestationKind = "pending"
	AttestationBitcoin AttestationKind = "bitcoin"
)

type Attestation struct {
	Kind     AttestationKind
	Calendar string
	Height   uint64
}

type Step struct {
	Operation   *Operation
	Attestation *Attestation
}

type Sequence []Step

type Receipt struct {
	Digest    [32]byte
	Sequences []Sequence
}

type EvaluatedAttestation struct {
	Message     []byte
	Attestation Attestation
	Sequence    Sequence
}

func NewPendingReceipt(targetDigest [32]byte, nonce [16]byte, branches []Sequence) (*Receipt, error) {
	if len(branches) == 0 {
		return nil, errors.New("at least one calendar branch is required")
	}
	prefix := Sequence{
		{Operation: &Operation{Tag: OperationAppend, Argument: append([]byte(nil), nonce[:]...)}},
		{Operation: &Operation{Tag: OperationSHA256}},
	}
	sequences := make([]Sequence, 0, len(branches))
	for _, branch := range branches {
		if err := validateSequence(branch); err != nil {
			return nil, fmt.Errorf("calendar branch: %w", err)
		}
		sequence := cloneSequence(prefix)
		sequence = append(sequence, cloneSequence(branch)...)
		sequences = append(sequences, sequence)
	}
	result := &Receipt{Digest: targetDigest, Sequences: sequences}
	result.Normalize()
	return result, nil
}

func Blind(targetDigest [32]byte, nonce [16]byte) [32]byte {
	buffer := make([]byte, 0, 48)
	buffer = append(buffer, targetDigest[:]...)
	buffer = append(buffer, nonce[:]...)
	return sha256.Sum256(buffer)
}

func (receipt *Receipt) Evaluate() ([]EvaluatedAttestation, error) {
	if receipt == nil || len(receipt.Sequences) == 0 {
		return nil, errors.New("receipt has no proof branches")
	}
	result := make([]EvaluatedAttestation, 0, len(receipt.Sequences))
	for _, sequence := range receipt.Sequences {
		if err := validateSequence(sequence); err != nil {
			return nil, err
		}
		message := append([]byte(nil), receipt.Digest[:]...)
		for _, step := range sequence[:len(sequence)-1] {
			var err error
			message, err = applyOperation(message, *step.Operation)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, EvaluatedAttestation{Message: message, Attestation: *sequence[len(sequence)-1].Attestation, Sequence: cloneSequence(sequence)})
	}
	return result, nil
}

func (receipt *Receipt) VerifyBinding(target []byte) error {
	digest := sha256.Sum256(target)
	if receipt == nil || digest != receipt.Digest {
		return errors.New("receipt digest does not match target bytes")
	}
	_, err := receipt.Evaluate()
	return err
}

func (receipt *Receipt) Normalize() {
	if receipt == nil {
		return
	}
	unique := make(map[string]Sequence, len(receipt.Sequences))
	for _, sequence := range receipt.Sequences {
		unique[string(sequenceKey(sequence))] = cloneSequence(sequence)
	}
	receipt.Sequences = receipt.Sequences[:0]
	for _, sequence := range unique {
		receipt.Sequences = append(receipt.Sequences, sequence)
	}
	sort.Slice(receipt.Sequences, func(i, j int) bool {
		return bytes.Compare(sequenceKey(receipt.Sequences[i]), sequenceKey(receipt.Sequences[j])) < 0
	})
}

func Merge(base *Receipt, additions ...*Receipt) (*Receipt, error) {
	if base == nil {
		return nil, errors.New("base receipt is nil")
	}
	merged := &Receipt{Digest: base.Digest, Sequences: cloneSequences(base.Sequences)}
	for _, addition := range additions {
		if addition == nil || addition.Digest != base.Digest {
			return nil, errors.New("cannot merge receipts for different digests")
		}
		merged.Sequences = append(merged.Sequences, cloneSequences(addition.Sequences)...)
	}
	merged.Normalize()
	return merged, nil
}

func IsSemanticSuperset(candidate, original *Receipt) bool {
	if candidate == nil || original == nil || candidate.Digest != original.Digest {
		return false
	}
	set := make(map[string]struct{}, len(candidate.Sequences))
	for _, sequence := range candidate.Sequences {
		set[string(sequenceKey(sequence))] = struct{}{}
	}
	for _, sequence := range original.Sequences {
		if _, ok := set[string(sequenceKey(sequence))]; ok {
			continue
		}
		// A confirmed branch is a semantic extension of its pending prefix.
		if sequence[len(sequence)-1].Attestation.Kind != AttestationPending {
			return false
		}
		prefix := sequence[:len(sequence)-1]
		found := false
		for _, other := range candidate.Sequences {
			if len(other) > len(prefix) && sequencePrefixEqual(other, prefix) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func applyOperation(message []byte, operation Operation) ([]byte, error) {
	switch operation.Tag {
	case OperationAppend:
		result := make([]byte, 0, len(message)+len(operation.Argument))
		result = append(result, message...)
		result = append(result, operation.Argument...)
		return result, nil
	case OperationPrepend:
		result := make([]byte, 0, len(message)+len(operation.Argument))
		result = append(result, operation.Argument...)
		result = append(result, message...)
		return result, nil
	case OperationSHA256:
		digest := sha256.Sum256(message)
		return digest[:], nil
	default:
		return nil, fmt.Errorf("unsupported OpenTimestamps operation tag 0x%02x", operation.Tag)
	}
}

func validateSequence(sequence Sequence) error {
	if len(sequence) == 0 || len(sequence) > MaxProofDepth {
		return errors.New("proof branch is empty or too deep")
	}
	for index, step := range sequence {
		if index == len(sequence)-1 {
			if step.Attestation == nil || step.Operation != nil {
				return errors.New("proof branch must end with one attestation")
			}
			if step.Attestation.Kind == AttestationPending && step.Attestation.Calendar == "" {
				return errors.New("pending attestation has no calendar identity")
			}
			if step.Attestation.Kind == AttestationBitcoin && step.Attestation.Height == 0 {
				return errors.New("Bitcoin attestation has no height")
			}
			continue
		}
		if step.Operation == nil || step.Attestation != nil {
			return errors.New("attestation may appear only at the end of a proof branch")
		}
		if step.Operation.Tag != OperationAppend && step.Operation.Tag != OperationPrepend && step.Operation.Tag != OperationSHA256 {
			return fmt.Errorf("unsupported OpenTimestamps operation tag 0x%02x", step.Operation.Tag)
		}
		if (step.Operation.Tag == OperationAppend || step.Operation.Tag == OperationPrepend) && len(step.Operation.Argument) > MaxOperationArgumentBytes {
			return errors.New("OpenTimestamps operation argument is too large")
		}
	}
	return nil
}

func sequencePrefixEqual(sequence, prefix Sequence) bool {
	if len(sequence) < len(prefix) {
		return false
	}
	return bytes.Equal(sequenceKey(sequence[:len(prefix)]), sequenceKey(prefix))
}

func sequenceKey(sequence Sequence) []byte {
	result := make([]byte, 0, len(sequence)*4)
	for _, step := range sequence {
		if step.Operation != nil {
			result = append(result, 1, step.Operation.Tag)
			result = appendVarBytes(result, step.Operation.Argument)
			continue
		}
		result = append(result, 0)
		if step.Attestation != nil {
			result = append(result, []byte(step.Attestation.Kind)...)
			result = append(result, 0)
			result = appendVarBytes(result, []byte(step.Attestation.Calendar))
			result = appendVarUint(result, step.Attestation.Height)
		}
	}
	return result
}

func cloneSequence(sequence Sequence) Sequence {
	result := make(Sequence, len(sequence))
	for index, step := range sequence {
		if step.Operation != nil {
			operation := *step.Operation
			operation.Argument = append([]byte(nil), operation.Argument...)
			result[index].Operation = &operation
		}
		if step.Attestation != nil {
			attestation := *step.Attestation
			result[index].Attestation = &attestation
		}
	}
	return result
}

func cloneSequences(sequences []Sequence) []Sequence {
	result := make([]Sequence, len(sequences))
	for index := range sequences {
		result[index] = cloneSequence(sequences[index])
	}
	return result
}
