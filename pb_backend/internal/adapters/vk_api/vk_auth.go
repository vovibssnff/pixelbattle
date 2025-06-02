package vk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"
)

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
	logrus.Info("Access token: ", accessResp.Response.AccessToken, " User ID: ", accessResp.Response.UserID)
	logrus.Info(accessResp.Response)

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
