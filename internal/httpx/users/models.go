package users

// UserCreateRequest represents user creation body (demo)
// swagger:model UserCreateRequest
type UserCreateRequest struct {
	DisplayName string `json:"display_name" example:"Alice"`
}
