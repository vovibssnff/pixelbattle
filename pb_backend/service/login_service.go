package service

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
)

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

type AccessReq struct {
	V           string `json:"v"`
	SilentToken string `json:"token"`
	AccessToken string `json:"access_token"`
	UUID        string `json:"uuid"`
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

type FacultyRequest struct {
	Faculty string `json:"faculty"`
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
	logrus.Info(response.Body)
	body, err := ioutil.ReadAll(response.Body)

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

func ToFaculty(r *http.Request) *FacultyRequest {
	var reqBody FacultyRequest
    err := json.NewDecoder(r.Body).Decode(&reqBody)
    if err != nil {
        logrus.Error(err)
    }
	return &reqBody
}
