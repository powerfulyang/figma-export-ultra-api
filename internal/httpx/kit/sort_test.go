package kit

import (
	"testing"

	"fiber-ent-apollo-pg/ent"
)

func TestApplyUserSort_ValidateField(t *testing.T) {
	c := ent.NewClient()
	q := c.User.Query()
	if _, err := ApplyUserSort(q, "display_name:asc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := ApplyUserSort(q, "unknown:asc"); err == nil {
		t.Fatalf("expected error for unknown field")
	}
}
