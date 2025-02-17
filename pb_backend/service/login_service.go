package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"
)

type VKUserData struct {
	UserID       int    `json:"userID"`
	DeviceID     string `json:"deviceID"`
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
	IDToken      string `json:"idToken"`
}

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

type VKUser struct {
	ID         int    `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Avatar     string `json:"avatar"`
	AvatarBase string `json:"avatar_base"`
	Phone      string `json:"phone"`
}

type User struct {
	ID              int    `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	CanAccessClosed bool   `json:"can_access_closed"`
	IsClosed        bool   `json:"is_closed"`
}

type VKUsrArray struct {
	Response []User `json:"response"`
}

type AccessReq struct {
	V           string `json:"v"`
	SilentToken string `json:"token"`
	AccessToken string `json:"access_token"`
	UUID        string `json:"uuid"`
}

type NameReq struct {
	UserId      string `json:"user_ids"`
	AccessToken string `json:"access_token"`
	V           string `json:"v"`
}

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

func NewNameReq(userId int, accessToken string, v string) *NameReq {
	return &NameReq{
		UserId:      strconv.Itoa(userId),
		AccessToken: accessToken,
		V:           v,
	}
}

func NewAccessReq(v string, silent_token string, service_token string, uuid string) *AccessReq {
	return &AccessReq{
		V:           v,
		SilentToken: silent_token,
		AccessToken: service_token,
		UUID:        uuid,
	}
}

func ToVKResponse(query url.Values) *VKResponse {
	decoded, err := url.QueryUnescape(query.Get("payload"))
	if err != nil {
		logrus.Error(err)
	}
	logrus.Info(decoded)
	var vkResponse VKResponse
	if err := json.Unmarshal([]byte(decoded), &vkResponse); err != nil {
		logrus.Error(err)
	}
	return &vkResponse
}

func SilentToAccess(access_req AccessReq) string {
	response, err := http.PostForm("https://api.vk.com/method/auth.exchangeSilentAuthToken", url.Values{
		"v":            {access_req.V},
		"token":        {access_req.SilentToken},
		"access_token": {access_req.AccessToken},
		"uuid":         {access_req.UUID},
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
	logrus.Info(string(body))

	var accessResp AccessResp
	err = json.Unmarshal([]byte(string(body)), &accessResp)
	if err != nil {
		logrus.Error(err)
		return ""
	}

	return accessResp.Response.AccessToken
}

func GetUsrInfo(nameReq NameReq) (User, error) {
	var usr User
	response, err := http.PostForm("https://api.vk.com/method/users.get", url.Values{
		"user_ids":     {nameReq.UserId},
		"access_token": {nameReq.AccessToken},
		"v":            {nameReq.V},
	})
	if err != nil {
		return usr, err
	}
	logrus.Info(response)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return usr, err
	}
	var usrs VKUsrArray
	err = json.Unmarshal([]byte(string(body)), &usrs)
	if err != nil {
		return usr, err
	}
	usr = usrs.Response[0]
	return usr, nil
}

// func IsBanned(checkReq CheckReq) bool {
// 	response, err := http.PostForm("https://api.vk.com/method/users.get", url.Values{
// 		"user_ids":     {checkReq.UserIds},
// 		"access_token": {checkReq.AccessToken},
// 		"v":            {checkReq.V},
// 	})

// 	if err != nil {
// 		logrus.Error(err)
// 		return true
// 	}

// defer response.Body.Close()
// body, err := io.ReadAll(response.Body)

// 	if err != nil {
// 		logrus.Error(err)
// 		return true
// 	}

// 	var usr VKCheckUser
// 	err = json.Unmarshal([]byte(string(body)), &usr)
// 	if err != nil {
// 		logrus.Error(err)
// 		return true
// 	}
// 	logrus.Info(usr)
// 	if usr.Response[0].Deactivated == "banned" || usr.Response[0].Deactivated == "deleted" {
// 		logrus.Info("Login request from vk banned usr: ", usr.Response[0].ID)
// 		return true
// 	}
// 	return false
// }
