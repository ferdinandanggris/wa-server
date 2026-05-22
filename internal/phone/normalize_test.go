package phone

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"08123456789", "+628123456789"},
		{"628123456789", "+628123456789"},
		{"+628123456789", "+628123456789"},
		{"=628123456789", "+628123456789"},
		{"whatsapp:628123456789", "+628123456789"},
		{"wa6281234567890", "+6281234567890"},
		{"12345", "12345"},
		{"", ""},
	}
	for _, tt := range tests {
		got := Normalize(tt.in)
		if got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
