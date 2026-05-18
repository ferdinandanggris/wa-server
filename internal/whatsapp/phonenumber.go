package whatsapp

import (
	"strings"
)

func NormalizePhoneNumber(phone string) string {
	if phone == "" {
		return ""
	}

	digits := ""
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			digits += string(c)
		}
	}

	if len(digits) < 10 {
		return phone
	}

	if strings.HasPrefix(digits, "0") {
		return "62" + digits[1:]
	}

	if !strings.HasPrefix(digits, "62") {
		return "62" + digits
	}

	return digits
}
