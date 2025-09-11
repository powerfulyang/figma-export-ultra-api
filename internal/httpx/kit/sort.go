package kit

import (
	"strings"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/user"

	"github.com/samber/lo"
)

type userSortApplier struct {
	Asc  func(*ent.UserQuery) *ent.UserQuery
	Desc func(*ent.UserQuery) *ent.UserQuery
}

// UserSortWhitelist defines allowed sort fields and their query modifiers for users
var UserSortWhitelist = map[string]userSortApplier{
	"created_at":   {Asc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Asc(user.FieldCreatedAt)) }, Desc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Desc(user.FieldCreatedAt)) }},
	"updated_at":   {Asc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Asc(user.FieldUpdatedAt)) }, Desc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Desc(user.FieldUpdatedAt)) }},
	"display_name": {Asc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Asc(user.FieldDisplayName)) }, Desc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Desc(user.FieldDisplayName)) }},
	"id":           {Asc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Asc(user.FieldID)) }, Desc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(ent.Desc(user.FieldID)) }},
}

func parseSortSpec(spec string) (field string, asc bool, err error) {
	if spec == "" {
		return "", true, nil
	}
	parts := strings.Split(spec, ":")
	field = strings.TrimSpace(parts[0])
	dir := lo.TernaryF(len(parts) > 1,
		func() string { return strings.ToLower(strings.TrimSpace(parts[1])) },
		func() string { return "asc" },
	)
	switch dir {
	case "asc":
		asc = true
	case "desc":
		asc = false
	default:
		return "", true, BadRequest("invalid sort direction", dir)
	}
	return field, asc, nil
}

// ApplyUserSort applies a validated sort spec to an ent UserQuery
func ApplyUserSort(q *ent.UserQuery, s string) (*ent.UserQuery, error) {
	field, asc, err := parseSortSpec(s)
	if err != nil {
		return nil, err
	}
	if field == "" {
		return q, nil
	}
	ap, ok := UserSortWhitelist[field]
	if !ok {
		return nil, BadRequest("invalid sort field", field)
	}
	if asc {
		return ap.Asc(q), nil
	}
	return ap.Desc(q), nil
}
