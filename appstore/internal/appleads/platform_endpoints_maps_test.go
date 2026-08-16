package appleads

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestPlatformMapsEndpointSpecsMatchFixtureLane(t *testing.T) {
	file, err := os.Open("testdata/platform_v1_endpoints.tsv")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	laneCollections := map[string]bool{
		"assets-endpoints":         true,
		"brands-endpoints":         true,
		"creatives-endpoints":      true,
		"location-groups-overview": true,
		"locations-overview":       true,
	}
	type fixtureContract struct {
		method       string
		path         string
		bodyKind     string
		bodyType     string
		bodyRequired string
		response     string
		context      string
		destructive  string
	}
	want := map[string]fixtureContract{}
	for _, row := range rows[1:] {
		if laneCollections[row[0]] {
			want[row[12]] = fixtureContract{
				method:       row[3],
				path:         strings.TrimPrefix(row[4], "/"),
				bodyKind:     row[6],
				bodyType:     row[7],
				bodyRequired: row[8],
				response:     strings.TrimPrefix(row[9], "200 "),
				context:      row[10],
				destructive:  row[11],
			}
		}
	}
	if got := len(want); got != 21 {
		t.Fatalf("fixture lane count = %d, want 21", got)
	}

	specs := platformMapsEndpointSpecs()
	if got := len(specs); got != 21 {
		t.Fatalf("platformMapsEndpointSpecs() count = %d, want 21", got)
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		command := strings.Join(spec.CommandPath, " ")
		contract, ok := want[command]
		if !ok {
			t.Fatalf("unexpected command %q", command)
		}
		if spec.Method != contract.method || spec.Path != contract.path {
			t.Fatalf("%s method/path = %s %s, want %s %s", command, spec.Method, spec.Path, contract.method, contract.path)
		}
		if seen[command] {
			t.Fatalf("duplicate command %q", command)
		}
		seen[command] = true
		if spec.Version != APIVersionPlatformV1 || spec.Context != ContextAdAccount || contract.context != "ad-account" {
			t.Fatalf("%s version/context = %q/%v", command, spec.Version, spec.Context)
		}
		bodyKinds := map[string]BodyKind{
			"none":                BodyNone,
			"JSON object":         BodyObject,
			"multipart/form-data": BodyMultipart,
		}
		wantBodyKind, known := bodyKinds[contract.bodyKind]
		if !known {
			t.Fatalf("%s unexpected fixture body kind %q", command, contract.bodyKind)
		}
		if spec.BodyKind != wantBodyKind {
			t.Fatalf("%s body kind = %v, want %v", command, spec.BodyKind, wantBodyKind)
		}
		wantBodyType := contract.bodyType
		if wantBodyType == "none" {
			wantBodyType = ""
		}
		if spec.BodyType != wantBodyType {
			t.Fatalf("%s body type = %q, want %q", command, spec.BodyType, wantBodyType)
		}
		if spec.ResponseType != contract.response {
			t.Fatalf("%s response type = %q, want %q", command, spec.ResponseType, contract.response)
		}
		isQuery := spec.Method == "POST" && strings.HasSuffix(spec.Path, "/query")
		if contract.bodyRequired == "yes" && spec.BodyOptional {
			t.Fatalf("%s fixture requires a body", command)
		}
		if isQuery && contract.bodyRequired == "no" && !spec.BodyOptional {
			t.Fatalf("%s fixture permits an omitted query body", command)
		}
		if isQuery && !spec.RetrySafe {
			t.Fatalf("read-only query %s must be retry-safe", command)
		}
		if (contract.destructive == "yes") != spec.RequiresConfirm {
			t.Fatalf("%s confirmation = %t, fixture destructive = %q", command, spec.RequiresConfirm, contract.destructive)
		}
		delete(want, command)
	}
	if len(want) != 0 {
		t.Fatalf("missing fixture commands: %+v", want)
	}
}

func TestPlatformMapsRepresentativeRequests(t *testing.T) {
	var seen []string
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "ACCOUNT"}, WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body []byte
		if req.Body != nil {
			var readErr error
			body, readErr = io.ReadAll(req.Body)
			if readErr != nil {
				t.Fatalf("read request: %v", readErr)
			}
		}
		seen = append(seen, req.Method+" "+req.URL.Path+" "+req.Header.Get("X-AP-Context")+" "+string(body))
		return jsonResponse(200, `{"data":{}}`), nil
	})}))
	if err != nil {
		t.Fatal(err)
	}

	brandQuery, _ := PlatformEndpointByCommandPath("brands", "find")
	if _, err := client.Do(context.Background(), brandQuery, nil, url.Values{}, []byte(`{"pagination":{"offset":0}}`)); err != nil {
		t.Fatalf("brand query: %v", err)
	}
	creativeUpdate, _ := PlatformEndpointByCommandPath("creatives", "update")
	if _, err := client.Do(context.Background(), creativeUpdate, map[string]string{"id": "creative-123"}, nil, []byte(`{"name":"Updated"}`)); err != nil {
		t.Fatalf("creative update: %v", err)
	}
	locationView, _ := PlatformEndpointByCommandPath("locations", "view")
	if _, err := client.Do(context.Background(), locationView, map[string]string{"id": "location-123"}, nil, nil); err != nil {
		t.Fatalf("location view: %v", err)
	}

	want := []string{
		`POST /v1/business-brands/query adAccountId=ACCOUNT; {"pagination":{"offset":0}}`,
		`PUT /v1/creatives/creative-123 adAccountId=ACCOUNT; {"name":"Updated"}`,
		`GET /v1/locations/location-123 adAccountId=ACCOUNT; `,
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(seen, "\n"), strings.Join(want, "\n"))
	}
}

func TestPlatformMapsEndpointIdentifierTypesAndUploadInventory(t *testing.T) {
	for _, path := range [][]string{
		{"brands", "view"},
		{"business-categories", "view"},
		{"locations", "view"},
		{"location-groups", "view"},
		{"location-groups", "update"},
		{"location-groups", "delete"},
		{"creatives", "view"},
		{"creatives", "update"},
		{"creatives", "delete"},
		{"assets", "view"},
		{"assets", "delete"},
	} {
		spec, ok := PlatformEndpointByCommandPath(path...)
		if !ok || len(spec.PathParams) != 1 || spec.PathParams[0].Type != ParamString {
			t.Fatalf("%s identifier metadata = %+v", strings.Join(path, " "), spec.PathParams)
		}
	}
	upload, ok := PlatformEndpointByCommandPath("assets", "upload")
	if !ok || upload.Method != "POST" || upload.Path != "v1/assets/upload" || upload.BodyKind != BodyMultipart {
		t.Fatalf("assets upload inventory = %+v", upload)
	}
	assets, ok := PlatformEndpointByCommandPath("assets", "find")
	if !ok || !assets.BodyOptional || assets.BodyFileExample != "query.json" {
		t.Fatalf("assets find query metadata = %+v", assets)
	}
	for _, want := range []string{"default page", "non-deleted assets", "selected ad account", "promotedObjectId", "providerAssetId", "assetType (IMAGE)"} {
		if !strings.Contains(assets.BodyHint, want) {
			t.Fatalf("assets find body hint = %q, want %q", assets.BodyHint, want)
		}
	}
}
