package deviceset

import "testing"

func TestDigestNormalizesFormattingAndDeduplicates(t *testing.T) {
	formatted := Digest([]string{"0000-1111:aaaa", "2222-bbbb", "00001111AAAA"})
	canonical := Digest([]string{"00001111AAAA", "2222BBBB"})
	if formatted != canonical {
		t.Fatalf("formatted digest=%#v, canonical=%#v", formatted, canonical)
	}
	if formatted.Count != 2 || formatted.SHA256 == "" {
		t.Fatalf("digest=%#v", formatted)
	}
}

func TestDigestDistinguishesDifferentSets(t *testing.T) {
	left := Digest([]string{"00001111AAAA", "2222BBBB"})
	right := Digest([]string{"00001111AAAA", "3333CCCC"})
	if left.SHA256 == right.SHA256 {
		t.Fatalf("different sets share digest %q", left.SHA256)
	}
}
