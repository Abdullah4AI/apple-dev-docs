package profiles

import "testing"

func TestIsValidProfileUUID(t *testing.T) {
	t.Parallel()

	valid := []string{
		"01234567-89ab-cdef-0123-456789abcdef",
		"01234567-89AB-CDEF-0123-456789ABCDEF", // uppercase
	}
	for _, v := range valid {
		v := v
		t.Run("valid/"+v, func(t *testing.T) {
			t.Parallel()
			if !isValidProfileUUID(v) {
				t.Fatalf("expected valid UUID, got invalid: %q", v)
			}
		})
	}

	invalid := []string{
		"",
		"not-a-uuid",
		"01234567-89ab-cdef-0123-456789abcde",   // too short
		"01234567-89ab-cdef-0123-456789abcdef0", // too long
		"/tmp/evil",
		"../evil",
		"..\\evil",
		"foo/bar",
		"foo\\bar",
		"C:\\evil",
		"C:/evil",
	}
	for _, v := range invalid {
		v := v
		t.Run("invalid/"+v, func(t *testing.T) {
			t.Parallel()
			if isValidProfileUUID(v) {
				t.Fatalf("expected invalid UUID, got valid: %q", v)
			}
		})
	}
}
