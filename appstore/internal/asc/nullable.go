package asc

import "encoding/json"

// NullableString represents a string that may be explicitly null in JSON.
// Use a nil pointer to omit the field entirely.
type NullableString struct {
	Value *string
}

func (n NullableString) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

func (n *NullableString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.Value = &value
	return nil
}
