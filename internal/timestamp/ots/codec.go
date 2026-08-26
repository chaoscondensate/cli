package ots

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	MaxReceiptBytes           = 4 << 20
	MaxProofBranches          = 256
	MaxProofDepth             = 256
	MaxOperationArgumentBytes = 1 << 20
)

var (
	detachedMagic = []byte{0x00, 0x4f, 0x70, 0x65, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x73, 0x00, 0x00, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x00, 0xbf, 0x89, 0xe2, 0xe8, 0x84, 0xe8, 0x92, 0x94}
	pendingMagic  = []byte{0x83, 0xdf, 0xe3, 0x0d, 0x2e, 0xf9, 0x0c, 0x8e}
	bitcoinMagic  = []byte{0x05, 0x88, 0x96, 0x0d, 0x73, 0xd7, 0x19, 0x01}
)

func ParseReceipt(data []byte) (*Receipt, error) {
	if len(data) == 0 || len(data) > MaxReceiptBytes {
		return nil, errors.New("OpenTimestamps receipt is empty or too large")
	}
	reader := newProofReader(data)
	magic, err := reader.bytes(len(detachedMagic))
	if err != nil || !bytes.Equal(magic, detachedMagic) {
		return nil, errors.New("file is not a detached OpenTimestamps proof")
	}
	version, err := reader.varUint()
	if err != nil || version != 1 {
		return nil, errors.New("unsupported OpenTimestamps detached proof version")
	}
	hashTag, err := reader.byte()
	if err != nil || hashTag != OperationSHA256 {
		return nil, errors.New("only detached SHA-256 OpenTimestamps proofs are supported")
	}
	digest, err := reader.bytes(32)
	if err != nil {
		return nil, errors.New("detached OpenTimestamps digest is truncated")
	}
	sequences, err := parseTimestampTree(reader)
	if err != nil {
		return nil, err
	}
	if !reader.eof() {
		return nil, errors.New("trailing OpenTimestamps proof data")
	}
	result := &Receipt{Sequences: sequences}
	copy(result.Digest[:], digest)
	result.Normalize()
	return result, nil
}

func ParseCalendarResponse(data []byte) ([]Sequence, error) {
	if len(data) == 0 || len(data) > MaxReceiptBytes {
		return nil, errors.New("calendar response is empty or too large")
	}
	reader := newProofReader(data)
	sequences, err := parseTimestampTree(reader)
	if err != nil {
		return nil, err
	}
	if !reader.eof() {
		return nil, errors.New("trailing calendar proof data")
	}
	return sequences, nil
}

func (receipt *Receipt) Serialize() ([]byte, error) {
	if receipt == nil || len(receipt.Sequences) == 0 || len(receipt.Sequences) > MaxProofBranches {
		return nil, errors.New("receipt has an invalid number of proof branches")
	}
	copyReceipt := &Receipt{Digest: receipt.Digest, Sequences: cloneSequences(receipt.Sequences)}
	copyReceipt.Normalize()
	for _, sequence := range copyReceipt.Sequences {
		if err := validateSequence(sequence); err != nil {
			return nil, err
		}
	}
	result := make([]byte, 0, 1024)
	result = append(result, detachedMagic...)
	result = appendVarUint(result, 1)
	result = append(result, OperationSHA256)
	result = append(result, receipt.Digest[:]...)
	tree, err := serializeSequences(copyReceipt.Sequences)
	if err != nil {
		return nil, err
	}
	result = append(result, tree...)
	if len(result) > MaxReceiptBytes {
		return nil, errors.New("serialized OpenTimestamps receipt is too large")
	}
	return result, nil
}

func SerializeCalendarResponse(sequences []Sequence) ([]byte, error) {
	return serializeSequences(cloneSequences(sequences))
}

func serializeSequences(sequences []Sequence) ([]byte, error) {
	if len(sequences) == 0 || len(sequences) > MaxProofBranches {
		return nil, errors.New("invalid number of proof branches")
	}
	for _, sequence := range sequences {
		if err := validateSequence(sequence); err != nil {
			return nil, err
		}
	}
	// The timestamp format uses 0xff as a continuation checkpoint. This
	// deterministic prefix serializer is compatible with the reference client.
	result := make([]byte, 0, 512)
	var writeNode func(prefix Sequence, group []Sequence) error
	writeNode = func(prefix Sequence, group []Sequence) error {
		if len(group) == 1 {
			for _, step := range group[0][len(prefix):] {
				var err error
				result, err = appendStep(result, step)
				if err != nil {
					return err
				}
			}
			return nil
		}
		common := len(prefix)
		for {
			if common >= len(group[0]) {
				break
			}
			key := sequenceKey(group[0][common : common+1])
			same := true
			for _, sequence := range group[1:] {
				if common >= len(sequence) || !bytes.Equal(key, sequenceKey(sequence[common:common+1])) {
					same = false
					break
				}
			}
			if !same {
				break
			}
			common++
		}
		for _, step := range group[0][len(prefix):common] {
			var err error
			result, err = appendStep(result, step)
			if err != nil {
				return err
			}
		}
		groups := make([][]Sequence, 0)
		for _, sequence := range group {
			placed := false
			for index := range groups {
				if bytes.Equal(sequenceKey(groups[index][0][common:common+1]), sequenceKey(sequence[common:common+1])) {
					groups[index] = append(groups[index], sequence)
					placed = true
					break
				}
			}
			if !placed {
				groups = append(groups, []Sequence{sequence})
			}
		}
		for index, subgroup := range groups {
			if index < len(groups)-1 {
				result = append(result, 0xff)
			}
			if err := writeNode(group[0][:common], subgroup); err != nil {
				return err
			}
		}
		return nil
	}
	if err := writeNode(nil, sequences); err != nil {
		return nil, err
	}
	return result, nil
}

func appendStep(destination []byte, step Step) ([]byte, error) {
	if step.Operation != nil {
		destination = append(destination, step.Operation.Tag)
		if step.Operation.Tag == OperationAppend || step.Operation.Tag == OperationPrepend {
			destination = appendVarBytes(destination, step.Operation.Argument)
		}
		return destination, nil
	}
	if step.Attestation == nil {
		return nil, errors.New("empty OpenTimestamps proof step")
	}
	destination = append(destination, 0x00)
	body := make([]byte, 0, 64)
	switch step.Attestation.Kind {
	case AttestationPending:
		destination = append(destination, pendingMagic...)
		body = appendVarBytes(body, []byte(step.Attestation.Calendar))
	case AttestationBitcoin:
		destination = append(destination, bitcoinMagic...)
		body = appendVarUint(body, step.Attestation.Height)
	default:
		return nil, fmt.Errorf("unsupported OpenTimestamps attestation %q", step.Attestation.Kind)
	}
	destination = appendVarBytes(destination, body)
	return destination, nil
}

func parseTimestampTree(reader *proofReader) ([]Sequence, error) {
	sequences := []Sequence{{}}
	current := 0
	checkpoints := make([]Sequence, 0, 8)
	for !reader.eof() {
		if len(sequences) > MaxProofBranches {
			return nil, errors.New("OpenTimestamps proof has too many branches")
		}
		tag, err := reader.byte()
		if err != nil {
			return nil, err
		}
		switch tag {
		case 0xff:
			checkpoints = append(checkpoints, cloneSequence(sequences[current]))
		case 0x00:
			attestation, err := parseAttestation(reader)
			if err != nil {
				return nil, err
			}
			sequences[current] = append(sequences[current], Step{Attestation: &attestation})
			if len(checkpoints) == 0 {
				if !reader.eof() {
					return nil, errors.New("proof contains data after its final attestation")
				}
				break
			}
			checkpoint := checkpoints[len(checkpoints)-1]
			checkpoints = checkpoints[:len(checkpoints)-1]
			sequences = append(sequences, checkpoint)
			current++
		default:
			operation, err := parseOperation(reader, tag)
			if err != nil {
				return nil, err
			}
			sequences[current] = append(sequences[current], Step{Operation: &operation})
			if len(sequences[current]) > MaxProofDepth {
				return nil, errors.New("OpenTimestamps proof branch is too deep")
			}
		}
	}
	if len(checkpoints) != 0 {
		return nil, errors.New("OpenTimestamps proof has an unfinished branch")
	}
	for _, sequence := range sequences {
		if err := validateSequence(sequence); err != nil {
			return nil, err
		}
	}
	return sequences, nil
}

func parseOperation(reader *proofReader, tag byte) (Operation, error) {
	operation := Operation{Tag: tag}
	switch tag {
	case OperationAppend, OperationPrepend:
		argument, err := reader.varBytes(MaxOperationArgumentBytes)
		if err != nil {
			return Operation{}, err
		}
		operation.Argument = append([]byte(nil), argument...)
	case OperationSHA256:
	default:
		return Operation{}, fmt.Errorf("unsupported OpenTimestamps operation tag 0x%02x", tag)
	}
	return operation, nil
}

func parseAttestation(reader *proofReader) (Attestation, error) {
	magic, err := reader.bytes(8)
	if err != nil {
		return Attestation{}, errors.New("truncated OpenTimestamps attestation tag")
	}
	body, err := reader.varBytes(MaxOperationArgumentBytes)
	if err != nil {
		return Attestation{}, err
	}
	bodyReader := newProofReader(body)
	switch {
	case bytes.Equal(magic, pendingMagic):
		calendar, err := bodyReader.varBytes(2048)
		if err != nil || !bodyReader.eof() || !utf8.Valid(calendar) || len(calendar) == 0 {
			return Attestation{}, errors.New("invalid pending OpenTimestamps attestation")
		}
		return Attestation{Kind: AttestationPending, Calendar: string(calendar)}, nil
	case bytes.Equal(magic, bitcoinMagic):
		height, err := bodyReader.varUint()
		if err != nil || !bodyReader.eof() || height == 0 {
			return Attestation{}, errors.New("invalid Bitcoin OpenTimestamps attestation")
		}
		return Attestation{Kind: AttestationBitcoin, Height: height}, nil
	default:
		return Attestation{}, fmt.Errorf("unsupported OpenTimestamps attestation tag %x", magic)
	}
}

type proofReader struct {
	data []byte
	pos  int
}

func newProofReader(data []byte) *proofReader { return &proofReader{data: data} }
func (reader *proofReader) eof() bool         { return reader.pos == len(reader.data) }

func (reader *proofReader) byte() (byte, error) {
	value, err := reader.bytes(1)
	if err != nil {
		return 0, err
	}
	return value[0], nil
}

func (reader *proofReader) bytes(count int) ([]byte, error) {
	if count < 0 || reader.pos > len(reader.data)-count {
		return nil, io.ErrUnexpectedEOF
	}
	value := reader.data[reader.pos : reader.pos+count]
	reader.pos += count
	return value, nil
}

func (reader *proofReader) varUint() (uint64, error) {
	var result uint64
	for shift := uint(0); shift < 64; shift += 7 {
		value, err := reader.byte()
		if err != nil {
			return 0, err
		}
		if shift == 63 && value > 1 {
			return 0, errors.New("OpenTimestamps varuint overflows")
		}
		result |= uint64(value&0x7f) << shift
		if value&0x80 == 0 {
			return result, nil
		}
	}
	return 0, errors.New("OpenTimestamps varuint is too long")
}

func (reader *proofReader) varBytes(limit int) ([]byte, error) {
	length, err := reader.varUint()
	if err != nil {
		return nil, err
	}
	if length > uint64(limit) {
		return nil, errors.New("OpenTimestamps field exceeds its byte limit")
	}
	return reader.bytes(int(length))
}

func appendVarUint(destination []byte, value uint64) []byte {
	for {
		part := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			part |= 0x80
		}
		destination = append(destination, part)
		if value == 0 {
			return destination
		}
	}
}

func appendVarBytes(destination, value []byte) []byte {
	destination = appendVarUint(destination, uint64(len(value)))
	return append(destination, value...)
}
