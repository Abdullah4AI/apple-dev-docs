package validation

import (
	"fmt"
	"unicode/utf8"
)

const keywordLengthUnit = "characters"

// KeywordFieldLength returns the App Store Connect keyword field length in characters.
func KeywordFieldLength(value string) int {
	return utf8.RuneCountInString(value)
}

// KeywordFieldLengthIssue returns an over-limit issue for keywords when present.
func KeywordFieldLengthIssue(value string) *MetadataLengthIssue {
	length := KeywordFieldLength(value)
	if length <= LimitKeywords {
		return nil
	}
	return &MetadataLengthIssue{
		Field:  "keywords",
		Length: length,
		Limit:  LimitKeywords,
		Unit:   keywordLengthUnit,
	}
}

// ValidateKeywordField returns an error when the keyword field exceeds ASC's character limit.
func ValidateKeywordField(value string) error {
	issue := KeywordFieldLengthIssue(value)
	if issue == nil {
		return nil
	}
	return fmt.Errorf("keywords exceed %d %s", issue.Limit, issue.Unit)
}
