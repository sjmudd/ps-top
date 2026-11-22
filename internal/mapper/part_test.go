package mapper

import (
	"testing"

	"github.com/sjmudd/ps-top/internal/snapshot"
)

func TestPart(t *testing.T) {
	k := snapshot.JoinKey("app", "10.0.0.5", "shop.v2", "21") // a part may contain "."
	for i, want := range []string{"app", "10.0.0.5", "shop.v2", "21", ""} {
		if got := Part(i)(k); got != want {
			t.Errorf("Part(%d) = %q, want %q", i, got, want)
		}
	}
}
