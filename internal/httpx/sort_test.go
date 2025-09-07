package httpx

import (
	"testing"

	"fiber-ent-apollo-pg/ent"
)

func TestParseSortSpec(t *testing.T) {
	f, asc, err := parseSortSpec("name:asc")
	if err != nil || f != "name" || !asc {
		t.Fatalf("want name asc, got f=%s asc=%v err=%v", f, asc, err)
	}
	f, asc, err = parseSortSpec("id:desc")
	if err != nil || f != "id" || asc {
		t.Fatalf("want id desc, got f=%s asc=%v err=%v", f, asc, err)
	}
	f, asc, err = parseSortSpec("id")
	if err != nil || f != "id" || !asc {
		t.Fatalf("want id asc default, got f=%s asc=%v err=%v", f, asc, err)
	}
	if _, _, err = parseSortSpec("name:sideways"); err == nil {
		t.Fatalf("expected error for invalid direction")
	}
}

func TestApplyUserSort_ValidateField(t *testing.T) {
	c := ent.NewClient()
	q := c.User.Query()
	if _, err := applyUserSort(q, "username:asc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := applyUserSort(q, "unknown:asc"); err == nil {
		t.Fatalf("expected error for unknown field")
	}
}
