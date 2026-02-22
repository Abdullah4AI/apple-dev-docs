package testflight

import (
	"encoding/json"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestParseDeviceFamilyOsVersionFilters(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{name: "single version", input: "IPHONE=26", wantLen: 1},
		{name: "version range", input: "IPAD=17..18", wantLen: 1},
		{name: "multiple families", input: "IPHONE=26,IPAD=17..18", wantLen: 2},
		{name: "missing separator", input: "IPHONE26", wantErr: true},
		{name: "missing version", input: "IPHONE=", wantErr: true},
		{name: "unknown family", input: "ANDROID=26", wantErr: true},
		{name: "bad range", input: "IPHONE=17..", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseDeviceFamilyOsVersionFilters(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != test.wantLen {
				t.Fatalf("expected %d filters, got %d", test.wantLen, len(got))
			}
		})
	}
}

func TestNormalizeRecruitmentHelpers(t *testing.T) {
	if _, err := normalizeBetaRecruitmentCriterionOptionsFields("deviceFamilyOsVersions"); err != nil {
		t.Fatalf("unexpected fields error: %v", err)
	}
	if _, err := normalizeBetaRecruitmentCriterionOptionsFields("bad"); err == nil {
		t.Fatal("expected field validation error")
	}

	if got, err := normalizeBetaRecruitmentDeviceFamily(string(asc.DeviceFamilyIPhone)); err != nil || got != asc.DeviceFamilyIPhone {
		t.Fatalf("expected %q, got %q err=%v", asc.DeviceFamilyIPhone, got, err)
	}
	if _, err := normalizeBetaRecruitmentDeviceFamily("BAD"); err == nil {
		t.Fatal("expected device family validation error")
	}

	if got, err := normalizeBetaTesterUsagePeriod("P30D"); err != nil || got != "P30D" {
		t.Fatalf("expected P30D, got %q err=%v", got, err)
	}
	if _, err := normalizeBetaTesterUsagePeriod("P10D"); err == nil {
		t.Fatal("expected period validation error")
	}
}

func TestParseBetaTesterUsagesPage(t *testing.T) {
	if _, err := parseBetaTesterUsagesPage(nil); err == nil {
		t.Fatal("expected error for empty payload")
	}
	if _, err := parseBetaTesterUsagesPage(json.RawMessage("{bad-json}")); err == nil {
		t.Fatal("expected error for invalid payload")
	}

	payload := json.RawMessage(`{
		"data":[{"metric":"x"}],
		"links":{"next":"https://example.com/next"},
		"meta":{"paging":"ok"}
	}`)
	page, err := parseBetaTesterUsagesPage(payload)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(page.Data) != 1 || page.Links.Next == "" || len(page.Meta) == 0 {
		t.Fatalf("unexpected parsed page: %+v", page)
	}
}

func TestRelationshipTypeListSorted(t *testing.T) {
	got := relationshipTypeList(map[string]relationshipKind{
		"builds": relationshipList,
		"app":    relationshipSingle,
		"groups": relationshipList,
	})
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0] != "app" || got[1] != "builds" || got[2] != "groups" {
		t.Fatalf("expected sorted relationship names, got %v", got)
	}
}
