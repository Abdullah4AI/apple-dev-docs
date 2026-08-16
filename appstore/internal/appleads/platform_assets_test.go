package appleads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestUploadPlatformAssetStreamsExactMultipartRequest(t *testing.T) {
	contents := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 'a', 's', 's', 'e', 't'}
	path := t.TempDir() + "/photo.jpeg"
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := os.Rename(path, path+".opened"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement must not be uploaded"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/assets/upload" {
			t.Fatalf("request = %s %s", req.Method, req.URL)
		}
		if got := req.Header.Get("X-AP-Context"); got != "adAccountId=ACCOUNT;" {
			t.Fatalf("X-AP-Context = %q", got)
		}
		if got := req.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q", got)
		}
		if len(req.TransferEncoding) != 0 {
			t.Fatalf("Transfer-Encoding = %v, want none", req.TransferEncoding)
		}
		if req.ContentLength <= int64(len(contents)) {
			t.Fatalf("ContentLength = %d", req.ContentLength)
		}
		if got := req.Header.Get("Content-Length"); got != strconv.FormatInt(req.ContentLength, 10) {
			t.Fatalf("Content-Length header = %q, want %d", got, req.ContentLength)
		}

		mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
			t.Fatalf("Content-Type = %q error=%v", req.Header.Get("Content-Type"), err)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if int64(len(body)) != req.ContentLength {
			t.Fatalf("body length = %d, want %d", len(body), req.ContentLength)
		}

		reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		type part struct {
			name, filename, contentType string
			body                        []byte
		}
		var parts []part
		for {
			item, err := reader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("next part: %v", err)
			}
			data, err := io.ReadAll(item)
			if err != nil {
				t.Fatalf("read part: %v", err)
			}
			parts = append(parts, part{item.FormName(), item.FileName(), item.Header.Get("Content-Type"), data})
		}
		if len(parts) != 3 {
			t.Fatalf("multipart parts = %+v", parts)
		}
		if parts[0].name != "file" || parts[0].filename != "photo.jpeg" || parts[0].contentType != "image/jpeg" || !bytes.Equal(parts[0].body, contents) {
			t.Fatalf("file part = %+v", parts[0])
		}
		if parts[1].name != "promotedObjectId" || string(parts[1].body) != "BRAND" {
			t.Fatalf("brand part = %+v", parts[1])
		}
		if parts[2].name != "promotedObjectType" || string(parts[2].body) != "BUSINESS_BRAND" {
			t.Fatalf("type part = %+v", parts[2])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"asset"}}`))
	}))
	defer server.Close()

	client, err := NewClient(
		Credentials{AccessToken: "ACCESS", AdAccountID: "ACCOUNT"},
		WithPlatformBaseURL(server.URL+"/v1/"),
	)
	if err != nil {
		t.Fatal(err)
	}

	response, err := client.UploadPlatformAsset(context.Background(), file, int64(len(contents)), "photo.jpeg", "image/jpeg", "BRAND")
	if err != nil {
		t.Fatalf("UploadPlatformAsset() error: %v", err)
	}
	if !strings.Contains(string(response), `"id":"asset"`) {
		t.Fatalf("response = %s", response)
	}
}

func TestUploadPlatformAssetReturnsAPIErrorsAndCancellation(t *testing.T) {
	newFile := func(t *testing.T) *os.File {
		t.Helper()
		path := t.TempDir() + "/asset.png"
		if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = file.Close() })
		return file
	}

	t.Run("API error", func(t *testing.T) {
		client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "ACCOUNT"}, WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(400, `{"error":{"message":"invalid asset"}}`), nil
		})}))
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.UploadPlatformAsset(context.Background(), newFile(t), 3, "asset.png", "image/png", "BRAND")
		if err == nil || !strings.Contains(err.Error(), "400") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "ACCOUNT"}, WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		})}))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = client.UploadPlatformAsset(ctx, newFile(t), 3, "asset.png", "image/png", "BRAND")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	})
}
