package idtest

import "testing"

func TestSanitizeLogValueRedactsInput(t *testing.T) {
	t.Parallel()

	if got := sanitizeLogValue("customer@example.com\r\nforged"); got != `[REDACTED]` {
		t.Fatalf("sanitizeLogValue() = %q", got)
	}
}
