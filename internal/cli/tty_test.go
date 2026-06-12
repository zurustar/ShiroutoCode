package cli

import (
	"os"
	"testing"
)

// enableCookedUTF8 on a non-terminal fd (a pipe) must fail safe: return a
// no-op restore func without panicking or changing anything.
func TestEnableCookedUTF8NonTTYNoop(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer pr.Close()
	defer pw.Close()

	restore := enableCookedUTF8(int(pr.Fd()))
	if restore == nil {
		t.Fatal("restore func must never be nil")
	}
	restore() // must not panic
}
