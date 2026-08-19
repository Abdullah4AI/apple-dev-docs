package asc

import "encoding/json"

// BetaBuildUsagesResponse wraps raw beta build usage metrics JSON.
type BetaBuildUsagesResponse struct {
	Data json.RawMessage `json:"-"`
}

// MarshalJSON preserves raw API JSON for beta build usage metrics.
func (r BetaBuildUsagesResponse) MarshalJSON() ([]byte, error) {
	if len(r.Data) == 0 {
		return []byte("null"), nil
	}
	return r.Data, nil
}

// BetaTesterUsagesResponse wraps raw beta tester usage metrics JSON.
type BetaTesterUsagesResponse struct {
	Data json.RawMessage `json:"-"`
}

// MarshalJSON preserves raw API JSON for beta tester usage metrics.
func (r BetaTesterUsagesResponse) MarshalJSON() ([]byte, error) {
	if len(r.Data) == 0 {
		return []byte("null"), nil
	}
	return r.Data, nil
}

// GetLinks parses the wrapped payload's links so single-page prints can
// participate in pagination warnings; printed output stays the raw payload.
func (r *BetaTesterUsagesResponse) GetLinks() *Links {
	return rawEnvelopeLinks(r.rawPayload())
}

// GetData exposes the wrapped payload's data array for page-size reporting.
func (r *BetaTesterUsagesResponse) GetData() any {
	return rawEnvelopeData(r.rawPayload())
}

func (r *BetaTesterUsagesResponse) rawPayload() json.RawMessage {
	if r == nil {
		return nil
	}
	return r.Data
}

// GetLinks parses the wrapped payload's links so single-page prints can
// participate in pagination warnings; printed output stays the raw payload.
func (r *BetaBuildUsagesResponse) GetLinks() *Links {
	return rawEnvelopeLinks(r.rawPayload())
}

// GetData exposes the wrapped payload's data array for page-size reporting.
func (r *BetaBuildUsagesResponse) GetData() any {
	return rawEnvelopeData(r.rawPayload())
}

func (r *BetaBuildUsagesResponse) rawPayload() json.RawMessage {
	if r == nil {
		return nil
	}
	return r.Data
}

func rawEnvelopeLinks(payload json.RawMessage) *Links {
	if len(payload) == 0 {
		return nil
	}
	var envelope struct {
		Links Links `json:"links"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil
	}
	links := envelope.Links
	return &links
}

func rawEnvelopeData(payload json.RawMessage) any {
	if len(payload) == 0 {
		return nil
	}
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil
	}
	return envelope.Data
}
