package phone

import "strings"

// Normalize converts Indonesian phone numbers to +62 format.
// Handles: 08xxx → +628xxx, 628xxx → +628xxx, +628xxx → unchanged,
// wa-xxx, =xxx prefixes stripped first.
func Normalize(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "=")
	p = strings.TrimPrefix(p, "whatsapp:")
	p = strings.TrimPrefix(p, "wa")

	if strings.HasPrefix(p, "+") {
		return p
	}
	if strings.HasPrefix(p, "62") {
		return "+" + p
	}
	if strings.HasPrefix(p, "0") {
		return "+62" + p[1:]
	}
	return p
}
