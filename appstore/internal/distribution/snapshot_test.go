package distribution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
)

func TestCopyWithContextStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	source := &cancelingReader{
		reader: bytes.NewReader(bytes.Repeat([]byte("snapshot"), 16<<10)),
		cancel: cancel,
	}
	var destination bytes.Buffer

	written, err := copyWithContext(ctx, &destination, source, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copyWithContext() error = %v, want context.Canceled", err)
	}
	if written != 0 || destination.Len() != 0 {
		t.Fatalf("copyWithContext() wrote %d bytes after cancellation during read", written)
	}
	if source.reads != 1 {
		t.Fatalf("copyWithContext() reads = %d, want 1", source.reads)
	}
}

func TestInspectionAndPreparationPropagateCancellation(t *testing.T) {
	path := writeIPA(t, map[string][]byte{"Payload/Demo.app/Info.plist": infoPlist(t, "com.example.demo")})
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := InspectIPAContext(ctx, file, info.Size(), InspectOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("InspectIPAContext() error = %v, want context.Canceled", err)
	}
	if _, err := PrepareIPAContext(ctx, file, info.Size(), PrepareOptions{Root: t.TempDir()}); !errors.Is(err, context.Canceled) {
		t.Fatalf("PrepareIPAContext() error = %v, want context.Canceled", err)
	}
}

type cancelingReader struct {
	reader *bytes.Reader
	cancel context.CancelFunc
	reads  int
}

func (r *cancelingReader) Read(buffer []byte) (int, error) {
	r.reads++
	count, err := r.reader.Read(buffer)
	r.cancel()
	return count, err
}
