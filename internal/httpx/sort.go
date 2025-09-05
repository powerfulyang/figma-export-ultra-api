package httpx

import (
	"strings"

	entlib "entgo.io/ent"
	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/ent/post"
	"fiber-ent-apollo-pg/ent/user"
)

// Sort whitelist mapping for entities. Extend these maps to add new sortable fields safely.
type userSortApplier struct {
	Asc  func(*ent.UserQuery) *ent.UserQuery
	Desc func(*ent.UserQuery) *ent.UserQuery
}

type postSortApplier struct {
	Asc  func(*ent.PostQuery) *ent.PostQuery
	Desc func(*ent.PostQuery) *ent.PostQuery
}

var UserSortWhitelist = map[string]userSortApplier{
	"created_at": {Asc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(entlib.Asc(user.FieldCreatedAt)) }, Desc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(entlib.Desc(user.FieldCreatedAt)) }},
	"name":       {Asc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(entlib.Asc(user.FieldName)) }, Desc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(entlib.Desc(user.FieldName)) }},
	"id":         {Asc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(entlib.Asc(user.FieldID)) }, Desc: func(q *ent.UserQuery) *ent.UserQuery { return q.Order(entlib.Desc(user.FieldID)) }},
}

var PostSortWhitelist = map[string]postSortApplier{
	"created_at": {Asc: func(q *ent.PostQuery) *ent.PostQuery { return q.Order(entlib.Asc(post.FieldCreatedAt)) }, Desc: func(q *ent.PostQuery) *ent.PostQuery { return q.Order(entlib.Desc(post.FieldCreatedAt)) }},
	"id":         {Asc: func(q *ent.PostQuery) *ent.PostQuery { return q.Order(entlib.Asc(post.FieldID)) }, Desc: func(q *ent.PostQuery) *ent.PostQuery { return q.Order(entlib.Desc(post.FieldID)) }},
}

func parseSortSpec(spec string) (field string, asc bool, err error) {
	if spec == "" {
		return "", true, nil
	}
	parts := strings.Split(spec, ":")
	field = strings.TrimSpace(parts[0])
	dir := "asc"
	if len(parts) > 1 {
		dir = strings.ToLower(strings.TrimSpace(parts[1]))
	}
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

func applyUserSort(q *ent.UserQuery, s string) (*ent.UserQuery, error) {
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

func applyPostSort(q *ent.PostQuery, s string) (*ent.PostQuery, error) {
	field, asc, err := parseSortSpec(s)
	if err != nil {
		return nil, err
	}
	if field == "" {
		return q, nil
	}
	ap, ok := PostSortWhitelist[field]
	if !ok {
		return nil, BadRequest("invalid sort field", field)
	}
	if asc {
		return ap.Asc(q), nil
	}
	return ap.Desc(q), nil
}
