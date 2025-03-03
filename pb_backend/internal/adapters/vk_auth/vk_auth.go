package vk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"
)

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

// ToVKResponse converts URL query parameters to a VKResponse.
func (s *VKAuthProvider) toVkResponse(query url.Values) *VKResponse {
	decoded, err := url.QueryUnescape(query.Get("payload"))
	if err != nil {
		logrus.Error(err)
	}
	var vkResponse VKResponse
	if err := json.Unmarshal([]byte(decoded), &vkResponse); err != nil {
		logrus.Error(err)
	}
	return &vkResponse
}

// SilentToAccess exchanges a silent token for an access token.
func (s *VKAuthProvider) silentToAccess(accessReq AccessReq) string {
	response, err := http.PostForm("https://api.vk.com/method/auth.exchangeSilentAuthToken", url.Values{
		"v":            {accessReq.V},
		"token":        {accessReq.SilentToken},
		"access_token": {accessReq.AccessToken},
		"uuid":         {accessReq.UUID},
	})

	if err != nil {
		logrus.Error(err)
		return ""
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	if err != nil {
		logrus.Error(err)
		return ""
	}

	var accessResp AccessResp
	err = json.Unmarshal([]byte(string(body)), &accessResp)
	if err != nil {
		logrus.Error(err)
		return ""
	}

	return accessResp.Response.AccessToken
}

// IsBanned checks if a user is banned or deleted.
func (s *VKAuthProvider) isBanned(userID int) bool {
	checkReq := s.newCheckReq(userID)
	response, err := http.PostForm("https://api.vk.com/method/users.get", url.Values{
		"user_ids":     {checkReq.UserIds},
		"access_token": {checkReq.AccessToken},
		"v":            {checkReq.V},
	})

	if err != nil {
		logrus.Error(err)
		return true
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	if err != nil {
		logrus.Error(err)
		return true
	}

	var usr VKCheckUser
	err = json.Unmarshal([]byte(string(body)), &usr)
	if err != nil {
		logrus.Error(err)
		return true
	}
	if usr.Response[0].Deactivated == "banned" || usr.Response[0].Deactivated == "deleted" {
		logrus.Info("Login request from vk banned usr: ", usr.Response[0].ID)
		return true
	}
	return false
}

// newCheckReq creates a new CheckReq for checking user status.
func (s *VKAuthProvider) newCheckReq(userID int) *CheckReq {
	return &CheckReq{
		UserIds:     strconv.Itoa(userID),
		AccessToken: s.ServiceToken,
		V:           s.ApiVer,
	}
}

// newAccessReq creates a new AccessReq for exchanging a silent token.
func (s *VKAuthProvider) newAccessReq(silentToken, uuid string) *AccessReq {
	return &AccessReq{
		V:           s.ApiVer,
		SilentToken: silentToken,
		AccessToken: s.ServiceToken,
		UUID:        uuid,
	}
}

func (s *VKAuthProvider) ValidVkUser(usr *VKUser, accessToken string) bool {
	if usr.FirstName == "" || usr.LastName == "" || usr.ID == 0 ||
		accessToken == "" || s.isBanned(usr.ID) {
		return false
	}
	return true
}

// returns VK access token
func (s *VKAuthProvider) Register(r *http.Request) (*VKUser, string) {
	vkResp := s.toVkResponse(r.URL.Query())
	accessReq := s.newAccessReq(vkResp.Token, vkResp.UUID)

	return &vkResp.User, s.silentToAccess(*accessReq)
}
