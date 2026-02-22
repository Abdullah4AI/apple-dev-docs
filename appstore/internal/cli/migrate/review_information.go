package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

type ReviewInformation struct {
	ContactFirstName    *string `json:"contactFirstName,omitempty"`
	ContactLastName     *string `json:"contactLastName,omitempty"`
	ContactPhone        *string `json:"contactPhone,omitempty"`
	ContactEmail        *string `json:"contactEmail,omitempty"`
	DemoAccountName     *string `json:"demoAccountName,omitempty"`
	DemoAccountPassword *string `json:"demoAccountPassword,omitempty"`
	DemoAccountRequired *bool   `json:"demoAccountRequired,omitempty"`
	Notes               *string `json:"notes,omitempty"`
}

func readFastlaneReviewInformation(metadataDir string) (*ReviewInformation, error) {
	reviewDir := filepath.Join(metadataDir, "review_information")
	if exists, err := dirExists(reviewDir); err != nil {
		return nil, err
	} else if !exists {
		return nil, nil
	}

	info := &ReviewInformation{}
	assigned := 0
	if value, ok, err := readOptionalFile(filepath.Join(reviewDir, "first_name.txt")); err != nil {
		return nil, err
	} else if ok {
		info.ContactFirstName = &value
		assigned++
	}
	if value, ok, err := readOptionalFile(filepath.Join(reviewDir, "last_name.txt")); err != nil {
		return nil, err
	} else if ok {
		info.ContactLastName = &value
		assigned++
	}
	if value, ok, err := readOptionalFile(filepath.Join(reviewDir, "phone_number.txt")); err != nil {
		return nil, err
	} else if ok {
		info.ContactPhone = &value
		assigned++
	}
	if value, ok, err := readOptionalFile(filepath.Join(reviewDir, "email_address.txt")); err != nil {
		return nil, err
	} else if ok {
		info.ContactEmail = &value
		assigned++
	}
	if value, ok, err := readOptionalFile(filepath.Join(reviewDir, "demo_user.txt")); err != nil {
		return nil, err
	} else if ok {
		info.DemoAccountName = &value
		assigned++
	}
	if value, ok, err := readOptionalFile(filepath.Join(reviewDir, "demo_password.txt")); err != nil {
		return nil, err
	} else if ok {
		info.DemoAccountPassword = &value
		assigned++
	}
	if value, ok, err := readOptionalFile(filepath.Join(reviewDir, "notes.txt")); err != nil {
		return nil, err
	} else if ok {
		info.Notes = &value
		assigned++
	}

	required, err := readOptionalReviewRequired(reviewDir)
	if err != nil {
		return nil, err
	}
	if required != nil {
		info.DemoAccountRequired = required
		assigned++
	}

	if assigned == 0 {
		return nil, nil
	}
	return info, nil
}

func buildReviewDetailCreateAttributes(info *ReviewInformation) *asc.AppStoreReviewDetailCreateAttributes {
	if info == nil {
		return nil
	}
	return &asc.AppStoreReviewDetailCreateAttributes{
		ContactFirstName:    info.ContactFirstName,
		ContactLastName:     info.ContactLastName,
		ContactPhone:        info.ContactPhone,
		ContactEmail:        info.ContactEmail,
		DemoAccountName:     info.DemoAccountName,
		DemoAccountPassword: info.DemoAccountPassword,
		DemoAccountRequired: info.DemoAccountRequired,
		Notes:               info.Notes,
	}
}

func buildReviewDetailUpdateAttributes(info *ReviewInformation) asc.AppStoreReviewDetailUpdateAttributes {
	if info == nil {
		return asc.AppStoreReviewDetailUpdateAttributes{}
	}
	return asc.AppStoreReviewDetailUpdateAttributes{
		ContactFirstName:    info.ContactFirstName,
		ContactLastName:     info.ContactLastName,
		ContactPhone:        info.ContactPhone,
		ContactEmail:        info.ContactEmail,
		DemoAccountName:     info.DemoAccountName,
		DemoAccountPassword: info.DemoAccountPassword,
		DemoAccountRequired: info.DemoAccountRequired,
		Notes:               info.Notes,
	}
}

func reviewInformationMatches(existing asc.AppStoreReviewDetailAttributes, info *ReviewInformation) bool {
	if info == nil {
		return true
	}
	if info.ContactFirstName != nil && existing.ContactFirstName != *info.ContactFirstName {
		return false
	}
	if info.ContactLastName != nil && existing.ContactLastName != *info.ContactLastName {
		return false
	}
	if info.ContactPhone != nil && existing.ContactPhone != *info.ContactPhone {
		return false
	}
	if info.ContactEmail != nil && existing.ContactEmail != *info.ContactEmail {
		return false
	}
	if info.DemoAccountName != nil && existing.DemoAccountName != *info.DemoAccountName {
		return false
	}
	if info.DemoAccountPassword != nil && existing.DemoAccountPassword != *info.DemoAccountPassword {
		return false
	}
	if info.DemoAccountRequired != nil && existing.DemoAccountRequired != *info.DemoAccountRequired {
		return false
	}
	if info.Notes != nil && existing.Notes != *info.Notes {
		return false
	}
	return true
}

func readOptionalReviewRequired(reviewDir string) (*bool, error) {
	primary := filepath.Join(reviewDir, "demo_account_required.txt")
	secondary := filepath.Join(reviewDir, "demo_required.txt")

	primaryValue, primaryExists, err := readOptionalFile(primary)
	if err != nil {
		return nil, err
	}
	secondaryValue, secondaryExists, err := readOptionalFile(secondary)
	if err != nil {
		return nil, err
	}

	if !primaryExists && !secondaryExists {
		return nil, nil
	}

	primaryParsed, primaryErr := parseReviewRequiredValue(primary, primaryValue, primaryExists)
	if primaryErr != nil {
		return nil, primaryErr
	}
	secondaryParsed, secondaryErr := parseReviewRequiredValue(secondary, secondaryValue, secondaryExists)
	if secondaryErr != nil {
		return nil, secondaryErr
	}
	if primaryParsed != nil && secondaryParsed != nil && *primaryParsed != *secondaryParsed {
		return nil, fmt.Errorf("review_information contains conflicting demo required values")
	}
	if primaryParsed != nil {
		return primaryParsed, nil
	}
	return secondaryParsed, nil
}

func parseReviewRequiredValue(path, value string, exists bool) (*bool, error) {
	if !exists {
		return nil, nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("review_information %s must be true or false", path)
	}
	switch strings.ToLower(trimmed) {
	case "true":
		v := true
		return &v, nil
	case "false":
		v := false
		return &v, nil
	default:
		return nil, fmt.Errorf("review_information %s must be true or false", path)
	}
}

func readOptionalFile(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(string(data)), true, nil
}

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}
