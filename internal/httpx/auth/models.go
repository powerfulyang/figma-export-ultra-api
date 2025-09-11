package auth

// TokenResponse represents an access token response
// swagger:model TokenResponse
type TokenResponse struct {
	AccessToken string `json:"access_token" example:"<JWT>"`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int    `json:"expires_in" example:"900"`
	DeviceID    string `json:"device_id,omitempty" example:"web-uuid-123"`
	AnonID      string `json:"anon_id,omitempty" example:"8a0d1b7c-..."`
}

// AnonymousInitRequest represents the request body for anonymous init
// swagger:model AnonymousInitRequest
type AnonymousInitRequest struct {
	DeviceID string         `json:"device_id" example:"web-uuid-123"`
	FPHash   *string        `json:"fp_hash,omitempty" example:"sha256:abcdef..."`
	Meta     map[string]any `json:"meta,omitempty"`
}

// LoginRequest represents the password login request body
// swagger:model LoginRequest
type LoginRequest struct {
	Identifier string `json:"identifier" example:"alice@example.com"`
	Password   string `json:"password" example:"Secretp@ssw0rd"`
	DeviceID   string `json:"device_id,omitempty" example:"web-uuid-123"`
}

// RegisterRequest represents the registration request body
// swagger:model RegisterRequest
type RegisterRequest struct {
	Identifier  string `json:"identifier" example:"alice@example.com"`
	Password    string `json:"password" example:"Secretp@ssw0rd"`
	DisplayName string `json:"display_name" example:"Alice"`
	DeviceID    string `json:"device_id,omitempty" example:"web-uuid-123"`
}

// FpSyncRequest represents the fingerprint/device sync request body
// swagger:model FpSyncRequest
type FpSyncRequest struct {
	DeviceID string         `json:"device_id" example:"web-uuid-123"`
	FPHash   *string        `json:"fp_hash,omitempty" example:"sha256:abcdef..."`
	UAHash   *string        `json:"ua_hash,omitempty" example:"sha256:ua..."`
	IPHash   *string        `json:"ip_hash,omitempty" example:"sha256:ip..."`
	Meta     map[string]any `json:"meta,omitempty"`
}
