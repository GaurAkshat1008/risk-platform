package keycloak

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpireIn int `json:"expires_in"`
	RefreshExpiresIn int `json:"refresh_expires_in"`
	TokenType string `json:"token_type"`
	Scope string `json:"scope"`
}

type UserRepresentation struct {
	ID string `json:"id"`
	Username string `json:"username"`
	Email string `json:"email"`
	FirstName string `json:"firstName"`
	LastName string `json:"lastName"`
	Enabled bool `json:"enabled"`
	Attributes map[string][]string `json:"attributes"`
}

type RoleRepresentation struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Composite bool `json:"composite"`
	ClientRole bool `json:"clientRole"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	ErrorDescription string `json:"error_description"`
}