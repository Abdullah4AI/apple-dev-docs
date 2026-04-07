package validation

import (
	"strings"
	"testing"
)

func TestValidateKeywordField_AcceptsArabicWithinCharacterLimit(t *testing.T) {
	value := "تغريدات,ردود,اعجابات,فلترة,بحث,ارشفة,ازالة,سجل,ريتويت,لايكات,منشن,خصوصية,منشورات,قديمة,حساب"

	if got := KeywordFieldLength(value); got != 91 {
		t.Fatalf("expected keyword length 91, got %d", got)
	}
	if err := ValidateKeywordField(value); err != nil {
		t.Fatalf("expected Arabic keywords within 100 characters to be valid, got %v", err)
	}
}

func TestValidateKeywordField_RejectsCharactersOverLimit(t *testing.T) {
	value := strings.Repeat("語", 101)

	err := ValidateKeywordField(value)
	if err == nil {
		t.Fatal("expected keyword length error")
	}
	if err.Error() != "keywords exceed 100 characters" {
		t.Fatalf("unexpected error: %v", err)
	}
}
