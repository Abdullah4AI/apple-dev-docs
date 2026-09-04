package signing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type signingRunJSONFrame struct {
	object    bool
	expectKey bool
	keys      map[string]struct{}
}

// rejectDuplicateSigningRunJSONKeys rejects any object containing two field
// names that encoding/json would treat as the same tagged field.
func rejectDuplicateSigningRunJSONKeys(data []byte) error {
	return validateSigningRunJSONKeys(data, nil)
}

// validateSigningRunJSONKeys walks a JSON document and rejects duplicate
// field names within an object. Keys are canonicalized with the same simple
// case folding encoding/json uses to match tagged struct fields, so a
// case-variant alias counts as the duplicate it effectively is. When a
// non-nil allowed set is supplied, every field name must also match one of
// its members exactly, which closes the standalone case-variant hole left by
// the decoder's case-insensitive field matching.
func validateSigningRunJSONKeys(data []byte, allowed map[string]struct{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var stack []signingRunJSONFrame
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{':
				stack = append(stack, signingRunJSONFrame{object: true, expectKey: true, keys: make(map[string]struct{})})
			case '[':
				stack = append(stack, signingRunJSONFrame{})
			case '}', ']':
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				markSigningRunJSONValueConsumed(stack)
			}
		case string:
			if len(stack) > 0 && stack[len(stack)-1].object && stack[len(stack)-1].expectKey {
				current := &stack[len(stack)-1]
				folded := foldSigningRunJSONKey(value)
				if _, exists := current.keys[folded]; exists {
					return fmt.Errorf("duplicate JSON field %q", value)
				}
				current.keys[folded] = struct{}{}
				if allowed != nil {
					if _, exact := allowed[value]; !exact {
						return fmt.Errorf("unknown JSON field %q", value)
					}
				}
				current.expectKey = false
			} else {
				markSigningRunJSONValueConsumed(stack)
			}
		default:
			markSigningRunJSONValueConsumed(stack)
		}
	}
}

// foldSigningRunJSONKey maps every rune to the minimum member of its simple
// case-folding orbit, so two keys fold to the same string exactly when
// strings.EqualFold reports them equal.
func foldSigningRunJSONKey(key string) string {
	return strings.Map(func(character rune) rune {
		minimum := character
		for folded := unicode.SimpleFold(character); folded != character; folded = unicode.SimpleFold(folded) {
			if folded < minimum {
				minimum = folded
			}
		}
		return minimum
	}, key)
}

func markSigningRunJSONValueConsumed(stack []signingRunJSONFrame) {
	if len(stack) > 0 && stack[len(stack)-1].object && !stack[len(stack)-1].expectKey {
		stack[len(stack)-1].expectKey = true
	}
}
