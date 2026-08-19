package asc

import (
	"encoding/json"
	"testing"
)

func TestBetaTesterUsagesResponsePaginationAccessors(t *testing.T) {
	payload := json.RawMessage(`{"data":[{"id":"a"},{"id":"b"}],"links":{"next":"https://api.appstoreconnect.apple.com/next"}}`)
	resp := &BetaTesterUsagesResponse{Data: payload}

	links := resp.GetLinks()
	if links == nil || links.Next != "https://api.appstoreconnect.apple.com/next" {
		t.Fatalf("GetLinks() = %+v, want next URL", links)
	}
	if n, ok := PageDataLen(resp); !ok || n != 2 {
		t.Fatalf("PageDataLen = %d ok=%t, want 2 true", n, ok)
	}

	var nilResp *BetaTesterUsagesResponse
	if nilResp.GetLinks() != nil || nilResp.GetData() != nil {
		t.Fatal("nil receiver should return nil accessors")
	}

	empty := &BetaTesterUsagesResponse{}
	if empty.GetLinks() != nil {
		t.Fatal("empty payload should return nil links")
	}
}

func TestBetaBuildUsagesResponsePaginationAccessors(t *testing.T) {
	payload := json.RawMessage(`{"data":[{"id":"a"}],"links":{"next":""}}`)
	resp := &BetaBuildUsagesResponse{Data: payload}
	links := resp.GetLinks()
	if links == nil || links.Next != "" {
		t.Fatalf("GetLinks() = %+v, want empty next", links)
	}
}
