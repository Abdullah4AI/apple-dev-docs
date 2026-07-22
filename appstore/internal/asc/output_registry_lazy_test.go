package asc

import (
	"reflect"
	"sync"
	"testing"
)

// registryEntriesAtProcessInit records how many renderers were registered by
// the time package initialization finished. Test-file init functions run
// after every non-test init in the package, so this observes the state a
// freshly started process would have before any registry lookup.
var registryEntriesAtProcessInit = -1

//nolint:gochecknoinits // required to observe package-init-time registry state
func init() {
	registryEntriesAtProcessInit = len(outputRegistry) + len(directRenderRegistry)
}

func TestOutputRegistryLazyPopulation(t *testing.T) {
	t.Run("no renderers registered during package init", func(t *testing.T) {
		if registryEntriesAtProcessInit != 0 {
			t.Fatalf("expected 0 renderers registered during package init, got %d; registration must stay lazy",
				registryEntriesAtProcessInit)
		}
	})

	t.Run("registry is fully populated after first use", func(t *testing.T) {
		var gotHeaders []string
		err := renderByRegistry(&LinkagesResponse{Data: []ResourceData{{Type: "apps", ID: "123"}}},
			func(headers []string, rows [][]string) {
				gotHeaders = headers
			})
		if err != nil {
			t.Fatalf("renderByRegistry returned error: %v", err)
		}
		if len(gotHeaders) == 0 {
			t.Fatal("expected registry-backed rendering for LinkagesResponse, got no headers")
		}

		// Mirrors the "expected minimum total registrations" sanity check:
		// one lookup must populate the complete registry, not a subset.
		const minExpected = 460
		total := len(outputRegistry) + len(directRenderRegistry)
		if total < minExpected {
			t.Fatalf("expected at least %d registered types after first use, got %d (rows: %d, direct: %d)",
				minExpected, total, len(outputRegistry), len(directRenderRegistry))
		}
	})

	t.Run("concurrent lookups are safe", func(t *testing.T) {
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := renderByRegistry(&LinkagesResponse{}, func([]string, [][]string) {})
				if err != nil {
					t.Errorf("renderByRegistry returned error: %v", err)
				}
			}()
		}
		wg.Wait()
	})
}

// BenchmarkRegistryFirstUse measures the one-time registration cost that was
// previously paid in package init by every process, including commands that
// never render registry output such as `asc --version`.
func BenchmarkRegistryFirstUse(b *testing.B) {
	origOutput, origDirect := outputRegistry, directRenderRegistry
	defer func() {
		outputRegistry, directRenderRegistry = origOutput, origDirect
	}()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputRegistry = make(map[reflect.Type]rowsFunc)
		directRenderRegistry = make(map[reflect.Type]directRenderFunc)
		registerAllOutputRenderers()
		registerValidationRenderers()
	}
}
