package vk

type VKAuthProvider struct {
	ServiceToken string
	ApiVer       string
}

func NewVKAuthProvider(serviceToken, apiVer string) *VKAuthProvider {
	return &VKAuthProvider{
		ServiceToken: serviceToken,
		ApiVer:       apiVer,
	}
}

// VKResponse represents the response from VK API.
type VKResponse struct {
	Type      string `json:"type"`
	Auth      int    `json:"auth"`
	User      VKUser `json:"user"`
	Token     string `json:"token"`
	TTL       int    `json:"ttl"`
	UUID      string `json:"uuid"`
	Hash      string `json:"hash"`
	LoadUsers bool   `json:"loadExternalUsers"`
}

// VKUser represents a VK user.
type VKUser struct {
	ID         int    `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Avatar     string `json:"avatar"`
	AvatarBase string `json:"avatar_base"`
	Phone      string `json:"phone"`
}

// User represents a user in the system.
type User struct {
	ID              int    `json:"id"`
	Deactivated     string `json:"deactivated"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	CanAccessClosed bool   `json:"can_access_closed"`
	IsClosed        bool   `json:"is_closed"`
}

// VKCheckUser represents the response from VK API for user checks.
type VKCheckUser struct {
	Response []User `json:"response"`
}

// AccessReq represents a request for exchanging a silent token for an access token.
type AccessReq struct {
	V           string `json:"v"`
	SilentToken string `json:"token"`
	AccessToken string `json:"access_token"`
	UUID        string `json:"uuid"`
}

// CheckReq represents a request for checking user status.
type CheckReq struct {
	UserIds     string `json:"user_ids"`
	AccessToken string `json:"access_token"`
	V           string `json:"v"`
}

// AccessResp represents the response from VK API for access token exchange.
type AccessResp struct {
	Response struct {
		AccessToken              string `json:"access_token"`
		AccessTokenID            string `json:"access_token_id"`
		UserID                   int    `json:"user_id"`
		AdditionalSignupRequired bool   `json:"additional_signup_required"`
		IsPartial                bool   `json:"is_partial"`
		IsService                bool   `json:"is_service"`
		Source                   int    `json:"source"`
		SourceDescription        string `json:"source_description"`
		ExpiresIn                int    `json:"expires_in"`
	} `json:"response"`
}
