// Package infoplist holds the shared ceiling the CLI applies when it expands an
// app bundle's Info.plist out of an IPA archive.
//
// An IPA is an untrusted ZIP archive: the CLI may be handed one by a CI job, a
// release pipeline, or a contributor. A ZIP member advertises its uncompressed
// size in metadata that the archive itself controls, and a highly compressible
// member expands to far more than it costs to store. Reading the selected
// Info.plist with an unbounded io.ReadAll therefore lets a small artifact drive
// an arbitrarily large allocation. Every IPA metadata reader shares the policy
// below so both the advertised size and the bytes actually streamed are
// rejected once they pass the limit.
package infoplist

import (
	"fmt"
	"io"
)

// MaxBytes is the largest uncompressed size accepted for the top-level
// Payload/*.app/Info.plist member of an IPA.
//
// A real top-level app Info.plist is a few kilobytes: a handful of bundle
// identifiers, version strings, supported platforms, icon names, and URL
// schemes. Even the unusually large ones — long ATS exception lists, wide
// device-capability matrices, or heavily localized declarations — stay in the
// low hundreds of kilobytes. 4 MiB leaves more than an order of magnitude of
// headroom above any plist Xcode plausibly emits while capping how much a
// crafted archive can force the CLI to allocate. There is deliberately no flag
// or environment override: raising the ceiling is the same as removing it.
const MaxBytes = 4 << 20

// CheckDeclaredSize rejects an Info.plist whose ZIP metadata already advertises
// more than MaxBytes, so an oversized member is refused before it is opened or
// decompressed.
func CheckDeclaredSize(uncompressedSize uint64) error {
	if uncompressedSize > MaxBytes {
		return fmt.Errorf("declared uncompressed size %d bytes exceeds the %d byte Info.plist limit", uncompressedSize, MaxBytes)
	}
	return nil
}

// ReadBounded expands at most MaxBytes from reader and fails if more bytes are
// available, so forged or absent ZIP size metadata cannot bypass
// CheckDeclaredSize.
func ReadBounded(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("expanded contents exceed the %d byte Info.plist limit", MaxBytes)
	}
	return data, nil
}
