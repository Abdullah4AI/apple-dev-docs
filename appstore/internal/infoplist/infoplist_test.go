package infoplist

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestCheckDeclaredSizeAtLimit(t *testing.T) {
	if err := CheckDeclaredSize(MaxBytes); err != nil {
		t.Fatalf("CheckDeclaredSize(MaxBytes) error: %v", err)
	}
}

func TestCheckDeclaredSizeOneByteOverLimit(t *testing.T) {
	err := CheckDeclaredSize(MaxBytes + 1)
	if err == nil {
		t.Fatal("expected declared-size rejection, got nil")
	}
	want := fmt.Sprintf("declared uncompressed size %d bytes exceeds the %d byte Info.plist limit", MaxBytes+1, MaxBytes)
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestReadBoundedAtLimit(t *testing.T) {
	data, err := ReadBounded(bytes.NewReader(bytes.Repeat([]byte("a"), MaxBytes)))
	if err != nil {
		t.Fatalf("ReadBounded() error: %v", err)
	}
	if len(data) != MaxBytes {
		t.Fatalf("expected %d bytes, got %d", MaxBytes, len(data))
	}
}

func TestReadBoundedOneByteOverLimit(t *testing.T) {
	_, err := ReadBounded(bytes.NewReader(bytes.Repeat([]byte("a"), MaxBytes+1)))
	if err == nil {
		t.Fatal("expected streamed-byte rejection, got nil")
	}
	want := fmt.Sprintf("expanded contents exceed the %d byte Info.plist limit", MaxBytes)
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

// TestReadBoundedStopsShortOfEndlessStream proves the streamed bound, not the
// declared one, is what protects the reader: an endless source is refused after
// MaxBytes+1 bytes instead of being expanded until memory runs out.
func TestReadBoundedStopsShortOfEndlessStream(t *testing.T) {
	source := &countingReader{}

	_, err := ReadBounded(source)
	if err == nil {
		t.Fatal("expected streamed-byte rejection, got nil")
	}
	if !strings.Contains(err.Error(), "Info.plist limit") {
		t.Fatalf("expected Info.plist limit error, got %v", err)
	}
	if source.read > MaxBytes+1 {
		t.Fatalf("expected at most %d bytes read, got %d", MaxBytes+1, source.read)
	}
}

func TestValidateStructureAcceptsDeepButReasonableXML(t *testing.T) {
	data := nestedXMLPlist(MaxDepth)
	if err := ValidateStructure(data); err != nil {
		t.Fatalf("ValidateStructure() rejected plist at depth limit: %v", err)
	}
}

func TestValidateStructureRejectsXMLDepthAmplification(t *testing.T) {
	data := nestedXMLPlist(MaxDepth + 1)
	err := ValidateStructure(data)
	if err == nil {
		t.Fatal("expected excessive plist depth rejection, got nil")
	}
	if !strings.Contains(err.Error(), "nesting depth") {
		t.Fatalf("expected nesting-depth error, got %v", err)
	}
}

func TestValidateStructureRejectsXMLObjectAmplification(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0"?><plist><array>`)
	for range MaxObjects + 1 {
		builder.WriteString(`<true/>`)
	}
	builder.WriteString(`</array></plist>`)

	err := ValidateStructure([]byte(builder.String()))
	if err == nil {
		t.Fatal("expected excessive plist object-count rejection, got nil")
	}
	if !strings.Contains(err.Error(), "object count") {
		t.Fatalf("expected object-count error, got %v", err)
	}
}

func TestValidateStructureRejectsBinaryObjectAmplificationFromTrailer(t *testing.T) {
	data := make([]byte, 40)
	copy(data, "bplist00")
	trailer := data[len(data)-32:]
	trailer[6] = 1
	trailer[7] = 1
	binary.BigEndian.PutUint64(trailer[8:16], MaxObjects+1)
	binary.BigEndian.PutUint64(trailer[16:24], 0)
	binary.BigEndian.PutUint64(trailer[24:32], 9)

	err := ValidateStructure(data)
	if err == nil {
		t.Fatal("expected excessive binary plist object-count rejection, got nil")
	}
	if !strings.Contains(err.Error(), "object count") {
		t.Fatalf("expected object-count error, got %v", err)
	}
}

func TestValidateStructureRejectsBinarySharedObjectAmplification(t *testing.T) {
	data := binaryPlistWithSharedArrayLevels(16)

	err := ValidateStructure(data)
	if err == nil {
		t.Fatal("expected shared binary object-count amplification rejection, got nil")
	}
	if !strings.Contains(err.Error(), "object count") {
		t.Fatalf("expected object-count error, got %v", err)
	}
}

func TestValidateStructureAcceptsReasonableBinarySharedObjects(t *testing.T) {
	data := binaryPlistWithSharedArrayLevels(10)

	if err := ValidateStructure(data); err != nil {
		t.Fatalf("ValidateStructure() rejected reasonable shared binary plist: %v", err)
	}
}

func TestValidateStructureRejectsBinaryDepthBeforeDescending(t *testing.T) {
	data := binaryPlistWithMalformedObjectBeyondDepthLimit()

	err := ValidateStructure(data)
	if err == nil {
		t.Fatal("expected excessive binary plist depth rejection, got nil")
	}
	if !strings.Contains(err.Error(), "nesting depth") {
		t.Fatalf("expected nesting-depth error before inspecting the deeper object, got %v", err)
	}
}

func TestValidateStructureRejectsBinaryCachedObjectPastDepthLimit(t *testing.T) {
	data := binaryPlistWithCachedObjectBeyondDepthLimit()

	err := ValidateStructure(data)
	if err == nil {
		t.Fatal("expected excessive binary plist depth rejection, got nil")
	}
	if !strings.Contains(err.Error(), "nesting depth") {
		t.Fatalf("expected nesting-depth error before inspecting the deeper sibling, got %v", err)
	}
}

func binaryPlistWithMalformedObjectBeyondDepthLimit() []byte {
	const objectCount = MaxDepth + 1
	data := append([]byte(nil), "bplist00"...)
	offsets := make([]uint16, 0, objectCount)
	for object := 0; object < objectCount-1; object++ {
		offsets = append(offsets, uint16(len(data)))
		data = append(data, 0xA1, byte(object+1))
	}
	offsets = append(offsets, uint16(len(data)))
	data = append(data, 0xAF, 0x00)

	offsetTable := uint64(len(data))
	for _, offset := range offsets {
		data = binary.BigEndian.AppendUint16(data, offset)
	}
	trailer := make([]byte, 32)
	trailer[6] = 2
	trailer[7] = 1
	binary.BigEndian.PutUint64(trailer[8:16], objectCount)
	binary.BigEndian.PutUint64(trailer[16:24], 0)
	binary.BigEndian.PutUint64(trailer[24:32], offsetTable)
	return append(data, trailer...)
}

func binaryPlistWithCachedObjectBeyondDepthLimit() []byte {
	const (
		chainObjects = MaxDepth - 2
		objectCount  = chainObjects + 4
		cachedObject = 1
		cachedLeaf   = 2
		firstChain   = 3
		malformed    = objectCount - 1
	)
	data := append([]byte(nil), "bplist00"...)
	offsets := make([]uint16, 0, objectCount)
	appendObject := func(object ...byte) {
		offsets = append(offsets, uint16(len(data)))
		data = append(data, object...)
	}

	appendObject(0xA2, cachedObject, firstChain)
	appendObject(0xA1, cachedLeaf)
	appendObject(0x51, 'x')
	for object := 0; object < chainObjects-1; object++ {
		appendObject(0xA1, byte(firstChain+object+1))
	}
	appendObject(0xA2, cachedObject, malformed)
	appendObject(0xAF, 0x00)

	offsetTable := uint64(len(data))
	for _, offset := range offsets {
		data = binary.BigEndian.AppendUint16(data, offset)
	}
	trailer := make([]byte, 32)
	trailer[6] = 2
	trailer[7] = 1
	binary.BigEndian.PutUint64(trailer[8:16], objectCount)
	binary.BigEndian.PutUint64(trailer[16:24], 0)
	binary.BigEndian.PutUint64(trailer[24:32], offsetTable)
	return append(data, trailer...)
}

func binaryPlistWithSharedArrayLevels(levels int) []byte {
	data := append([]byte(nil), "bplist00"...)
	offsets := make([]byte, 0, levels+3)
	appendObject := func(object ...byte) {
		offsets = append(offsets, byte(len(data)))
		data = append(data, object...)
	}

	appendObject(0x54, 'B', 'o', 'm', 'b')
	appendObject(0x51, 'x')
	previous := byte(1)
	for range levels {
		appendObject(0xA2, previous, previous)
		previous++
	}
	topObject := byte(len(offsets))
	appendObject(0xD1, 0, previous)

	offsetTable := uint64(len(data))
	data = append(data, offsets...)
	trailer := make([]byte, 32)
	trailer[6] = 1
	trailer[7] = 1
	binary.BigEndian.PutUint64(trailer[8:16], uint64(len(offsets)))
	binary.BigEndian.PutUint64(trailer[16:24], uint64(topObject))
	binary.BigEndian.PutUint64(trailer[24:32], offsetTable)
	return append(data, trailer...)
}

func nestedXMLPlist(depth int) []byte {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0"?><plist>`)
	for range depth - 1 {
		builder.WriteString(`<array>`)
	}
	builder.WriteString(`<string>value</string>`)
	for range depth - 1 {
		builder.WriteString(`</array>`)
	}
	builder.WriteString(`</plist>`)
	return []byte(builder.String())
}

type countingReader struct {
	read int
}

func (r *countingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	r.read += len(p)
	return len(p), nil
}

var _ io.Reader = (*countingReader)(nil)
