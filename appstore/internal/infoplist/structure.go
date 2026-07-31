package infoplist

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

// MaxDepth is the deepest container chain accepted in an app Info.plist.
// Xcode-generated metadata is normally fewer than ten levels deep. A limit of
// 128 keeps ample room for hand-authored URL schemes, scene manifests, and
// extension dictionaries while preventing recursive plist decoders from being
// driven into pathological stack growth.
const MaxDepth = 128

// MaxObjects bounds the number of scalar and container values a plist decoder
// may materialize. Sixty-five thousand values are already far beyond a real
// app manifest, but keep the 4 MiB byte allowance useful for legitimately wide
// arrays and dictionaries.
const MaxObjects = 65_536

// ValidateStructure applies format-aware depth and object-count limits before
// the third-party plist decoder materializes the document.
func ValidateStructure(data []byte) error {
	if len(data) > MaxBytes {
		return fmt.Errorf("contents exceed the %d byte Info.plist limit", MaxBytes)
	}
	if bytes.HasPrefix(data, []byte("bplist")) {
		return validateBinaryStructure(data)
	}
	trimmed := bytes.TrimSpace(data)
	trimmed = bytes.TrimPrefix(trimmed, []byte{0xEF, 0xBB, 0xBF})
	trimmed = bytes.TrimSpace(trimmed)
	if len(trimmed) > 0 && trimmed[0] == '<' {
		return validateXMLStructure(trimmed)
	}
	return validateOpenStepStructure(trimmed)
}

func validateXMLStructure(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	depth := 0
	objects := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect XML structure: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if !isPlistValueElement(value.Name.Local) {
				continue
			}
			objects++
			if objects > MaxObjects {
				return fmt.Errorf("info.plist object count exceeds %d", MaxObjects)
			}
			depth++
			if depth > MaxDepth {
				return fmt.Errorf("info.plist nesting depth exceeds %d", MaxDepth)
			}
		case xml.EndElement:
			if isPlistValueElement(value.Name.Local) {
				depth--
			}
		}
	}
}

func isPlistValueElement(name string) bool {
	switch name {
	case "dict", "array", "string", "integer", "real", "true", "false", "date", "data":
		return true
	default:
		return false
	}
}

func validateOpenStepStructure(data []byte) error {
	stack := make([]byte, 0, 16)
	objects := 1
	inString := false
	escaped := false
	lineComment := false
	blockComment := false
	inData := false

	for i := 0; i < len(data); i++ {
		current := data[i]
		var next byte
		if i+1 < len(data) {
			next = data[i+1]
		}
		switch {
		case lineComment:
			if current == '\n' || current == '\r' {
				lineComment = false
			}
			continue
		case blockComment:
			if current == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		case inString:
			if escaped {
				escaped = false
			} else if current == '\\' {
				escaped = true
			} else if current == '"' {
				inString = false
			}
			continue
		case inData:
			if current == '>' {
				inData = false
			}
			continue
		case current == '/' && next == '/':
			lineComment = true
			i++
			continue
		case current == '/' && next == '*':
			blockComment = true
			i++
			continue
		case current == '"':
			inString = true
			continue
		case current == '<':
			inData = true
			continue
		case current == '{' || current == '(':
			stack = append(stack, current)
			objects++
			if len(stack) > MaxDepth {
				return fmt.Errorf("info.plist nesting depth exceeds %d", MaxDepth)
			}
		case current == ',' || current == ';':
			objects++
		}
		if objects > MaxObjects {
			return fmt.Errorf("info.plist object count exceeds %d", MaxObjects)
		}
	}
	return nil
}

type binaryPlistTrailer struct {
	offsetSize  uint8
	refSize     uint8
	numObjects  uint64
	topObject   uint64
	offsetTable uint64
	trailerAt   uint64
}

type binaryObjectMetrics struct {
	expandedCount uint64
	depth         int
}

func validateBinaryStructure(data []byte) error {
	if len(data) < 40 {
		return fmt.Errorf("inspect binary structure: document is too short")
	}
	trailerAt := uint64(len(data) - 32)
	trailerBytes := data[trailerAt:]
	trailer := binaryPlistTrailer{
		offsetSize:  trailerBytes[6],
		refSize:     trailerBytes[7],
		numObjects:  binary.BigEndian.Uint64(trailerBytes[8:16]),
		topObject:   binary.BigEndian.Uint64(trailerBytes[16:24]),
		offsetTable: binary.BigEndian.Uint64(trailerBytes[24:32]),
		trailerAt:   trailerAt,
	}
	if trailer.numObjects > MaxObjects {
		return fmt.Errorf("info.plist object count exceeds %d", MaxObjects)
	}
	if trailer.numObjects == 0 || trailer.topObject >= trailer.numObjects {
		return fmt.Errorf("inspect binary structure: invalid top object")
	}
	if trailer.offsetSize == 0 || trailer.offsetSize > 8 || trailer.refSize == 0 || trailer.refSize > 8 {
		return fmt.Errorf("inspect binary structure: invalid offset or object-reference size")
	}
	if trailer.offsetTable < 8 || trailer.offsetTable >= trailer.trailerAt {
		return fmt.Errorf("inspect binary structure: invalid offset table")
	}
	tableBytes := trailer.numObjects * uint64(trailer.offsetSize)
	if trailer.numObjects != 0 && tableBytes/trailer.numObjects != uint64(trailer.offsetSize) {
		return fmt.Errorf("inspect binary structure: offset table overflow")
	}
	if tableBytes > trailer.trailerAt-trailer.offsetTable {
		return fmt.Errorf("inspect binary structure: truncated offset table")
	}

	offsets := make([]uint64, trailer.numObjects)
	for object := uint64(0); object < trailer.numObjects; object++ {
		offsetAt := trailer.offsetTable + object*uint64(trailer.offsetSize)
		offset, ok := readBinaryUint(data, offsetAt, trailer.offsetSize)
		if !ok || offset < 8 || offset >= trailer.offsetTable {
			return fmt.Errorf("inspect binary structure: invalid object offset")
		}
		offsets[object] = offset
	}

	state := make([]uint8, trailer.numObjects)
	metrics := make([]binaryObjectMetrics, trailer.numObjects)
	var inspectObject func(uint64, int) (binaryObjectMetrics, error)
	inspectObject = func(object uint64, traversalDepth int) (binaryObjectMetrics, error) {
		if traversalDepth > MaxDepth {
			return binaryObjectMetrics{}, fmt.Errorf("info.plist nesting depth exceeds %d", MaxDepth)
		}
		switch state[object] {
		case 1:
			return binaryObjectMetrics{}, fmt.Errorf("inspect binary structure: self-referential container")
		case 2:
			if metrics[object].depth > MaxDepth-traversalDepth+1 {
				return binaryObjectMetrics{}, fmt.Errorf("info.plist nesting depth exceeds %d", MaxDepth)
			}
			return metrics[object], nil
		}
		state[object] = 1

		children, err := binaryContainerChildren(data, offsets[object], trailer)
		if err != nil {
			return binaryObjectMetrics{}, err
		}
		current := binaryObjectMetrics{expandedCount: 1, depth: 1}
		for _, child := range children {
			if child >= trailer.numObjects {
				return binaryObjectMetrics{}, fmt.Errorf("inspect binary structure: object reference is out of range")
			}
			childMetrics, err := inspectObject(child, traversalDepth+1)
			if err != nil {
				return binaryObjectMetrics{}, err
			}
			if childMetrics.expandedCount > uint64(MaxObjects)-current.expandedCount {
				return binaryObjectMetrics{}, fmt.Errorf("info.plist object count exceeds %d", MaxObjects)
			}
			current.expandedCount += childMetrics.expandedCount
			if childMetrics.depth+1 > current.depth {
				current.depth = childMetrics.depth + 1
			}
			if current.depth > MaxDepth {
				return binaryObjectMetrics{}, fmt.Errorf("info.plist nesting depth exceeds %d", MaxDepth)
			}
		}
		state[object] = 2
		metrics[object] = current
		return current, nil
	}
	_, err := inspectObject(trailer.topObject, 1)
	return err
}

func binaryContainerChildren(data []byte, offset uint64, trailer binaryPlistTrailer) ([]uint64, error) {
	if offset >= trailer.offsetTable {
		return nil, fmt.Errorf("inspect binary structure: object begins beyond object table")
	}
	tag := data[offset]
	kind := tag & 0xF0
	if kind != 0xA0 && kind != 0xD0 {
		return nil, nil
	}
	count, refsAt, err := binaryObjectCount(data, offset, trailer.offsetTable)
	if err != nil {
		return nil, err
	}
	if count > MaxObjects {
		return nil, fmt.Errorf("info.plist object count exceeds %d", MaxObjects)
	}
	if kind == 0xD0 {
		if count > MaxObjects/2 {
			return nil, fmt.Errorf("info.plist object count exceeds %d", MaxObjects)
		}
		count *= 2
	}
	refBytes := count * uint64(trailer.refSize)
	if count != 0 && refBytes/count != uint64(trailer.refSize) {
		return nil, fmt.Errorf("inspect binary structure: container size overflow")
	}
	if refsAt > trailer.offsetTable || refBytes > trailer.offsetTable-refsAt {
		return nil, fmt.Errorf("inspect binary structure: truncated container")
	}
	children := make([]uint64, count)
	for index := uint64(0); index < count; index++ {
		ref, ok := readBinaryUint(data, refsAt+index*uint64(trailer.refSize), trailer.refSize)
		if !ok {
			return nil, fmt.Errorf("inspect binary structure: truncated object reference")
		}
		children[index] = ref
	}
	return children, nil
}

func binaryObjectCount(data []byte, offset, tableAt uint64) (uint64, uint64, error) {
	tag := data[offset]
	if tag&0x0F != 0x0F {
		return uint64(tag & 0x0F), offset + 1, nil
	}
	integerAt := offset + 1
	if integerAt >= tableAt || data[integerAt]&0xF0 != 0x10 {
		return 0, 0, fmt.Errorf("inspect binary structure: invalid extended object count")
	}
	exponent := data[integerAt] & 0x0F
	if exponent > 3 {
		return 0, 0, fmt.Errorf("inspect binary structure: object count is too wide")
	}
	size := uint8(1 << exponent)
	count, ok := readBinaryUint(data, integerAt+1, size)
	if !ok || integerAt+1+uint64(size) > tableAt {
		return 0, 0, fmt.Errorf("inspect binary structure: truncated object count")
	}
	return count, integerAt + 1 + uint64(size), nil
}

func readBinaryUint(data []byte, offset uint64, size uint8) (uint64, bool) {
	if size == 0 || size > 8 || offset > uint64(len(data)) || uint64(size) > uint64(len(data))-offset {
		return 0, false
	}
	var value uint64
	for _, current := range data[offset : offset+uint64(size)] {
		value = value<<8 | uint64(current)
	}
	return value, true
}
