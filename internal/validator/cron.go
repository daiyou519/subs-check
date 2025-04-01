package validator

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidCronFormat = errors.New("invalid cron expression format")
	ErrInvalidCronValue  = errors.New("invalid cron expression value")
)

// ValidateCron validates if the cron expression is valid
// Supported format: * * * * * (minute hour day month week)
// Supported special characters: * / , -
func ValidateCron(cron string) error {
	cron = strings.TrimSpace(cron)

	parts := strings.Fields(cron)
	if len(parts) != 5 {
		return ErrInvalidCronFormat
	}

	validations := []struct {
		field    string
		min, max int
		pattern  string
	}{
		{parts[0], 0, 59, `^(\*|[0-9\-\*\/,]+)$`},
		{parts[1], 0, 23, `^(\*|[0-9\-\*\/,]+)$`},
		{parts[2], 1, 31, `^(\*|[0-9\-\*\/,]+)$`},
		{parts[3], 1, 12, `^(\*|[0-9\-\*\/,]+)$`},
		{parts[4], 0, 6, `^(\*|[0-9\-\*\/,]+)$`},
	}

	for _, v := range validations {
		matched, err := regexp.MatchString(v.pattern, v.field)
		if err != nil || !matched {
			return ErrInvalidCronFormat
		}

		if v.field == "*" {
			continue
		}

		values := strings.Split(v.field, ",")
		for _, value := range values {
			if strings.Contains(value, "-") {
				rangeParts := strings.Split(value, "-")
				if len(rangeParts) != 2 {
					return ErrInvalidCronValue
				}
				continue
			}

			if strings.Contains(value, "/") {
				stepParts := strings.Split(value, "/")
				if len(stepParts) != 2 {
					return ErrInvalidCronValue
				}
				continue
			}

			if value != "*" {
			}
		}
	}

	return nil
}
