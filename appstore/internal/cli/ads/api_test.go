package ads

import (
	"net/http"
	"strings"
	"testing"
)

func TestRawV5RequestRequiresConfirmByMethodAndCanonicalPath(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		confirm bool
		wantErr string
	}{
		{name: "known GET", method: http.MethodGet, path: "v5/campaigns", confirm: false},
		{name: "known find POST", method: http.MethodPost, path: "v5/campaigns/find", confirm: false},
		{name: "known report POST absolute URL", method: http.MethodPost, path: "https://api.searchads.apple.com/api/v5/reports/campaigns/12/adgroups?audit=true", confirm: false},
		{name: "known geo resolve POST", method: http.MethodPost, path: "v5/search/geo", confirm: false},
		{name: "benign creative create POST", method: http.MethodPost, path: "v5/creatives", confirm: false},
		{name: "read-like custom report create POST", method: http.MethodPost, path: "v5/custom-reports", confirm: false},
		{name: "budget create POST", method: http.MethodPost, path: "v5/budgetorders", confirm: true},
		{name: "campaign update PUT", method: http.MethodPut, path: "v5/campaigns/123", confirm: true},
		{name: "keyword create POST canonical IDs", method: http.MethodPost, path: "v5/campaigns/1/adgroups/2/targetingkeywords/bulk", confirm: true},
		{name: "bulk delete POST", method: http.MethodPost, path: "v5/campaigns/1/negativekeywords/delete/bulk", confirm: true},
		{name: "unknown POST fails closed", method: http.MethodPost, path: "v5/future-resource/query", confirm: true},
		{name: "unknown PUT fails closed", method: http.MethodPut, path: "v5/future-resource/1", confirm: true},
		{name: "unknown GET stays read-only", method: http.MethodGet, path: "v5/future-resource/1", confirm: false},
		{name: "DELETE always confirms", method: http.MethodDelete, path: "v5/future-resource/1", confirm: true},
		{name: "method mismatch does not borrow GET contract", method: http.MethodPost, path: "v5/campaigns/123", confirm: true},
		{name: "reject non Apple URL", method: http.MethodPost, path: "https://example.com/api/v5/campaigns", wantErr: "Apple Ads v5 URL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			confirm, err := rawV5RequestRequiresConfirm(test.method, test.path)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("rawV5RequestRequiresConfirm() error: %v", err)
			}
			if confirm != test.confirm {
				t.Fatalf("confirm = %t, want %t", confirm, test.confirm)
			}
		})
	}
}
