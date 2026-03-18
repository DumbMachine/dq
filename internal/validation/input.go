package validation

import (
	"fmt"
	"strings"
	"unicode"
)

const maxQueryLength = 100000

// ValidateSQL performs basic input validation on SQL queries.
func ValidateSQL(sql string) error {
	if strings.TrimSpace(sql) == "" {
		return fmt.Errorf("empty SQL query")
	}
	if len(sql) > maxQueryLength {
		return fmt.Errorf("query exceeds maximum length of %d characters", maxQueryLength)
	}
	if containsControlChars(sql) {
		return fmt.Errorf("query contains invalid control characters")
	}
	return nil
}

// ValidateName validates connection/table names.
func ValidateName(name, label string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if len(name) > 128 {
		return fmt.Errorf("%s exceeds maximum length of 128 characters", label)
	}
	if containsControlChars(name) {
		return fmt.Errorf("%s contains invalid control characters", label)
	}
	return nil
}

func containsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
}
