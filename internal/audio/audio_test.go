package audio

import (
	"testing"

	"github.com/ebitengine/oto/v3"
)

// TestOtoImport verifies that oto/v3 imports correctly
func TestOtoImport(t *testing.T) {
	// Just verify the import works
	_ = oto.NewContext
	t.Log("oto/v3 imported successfully")
}
