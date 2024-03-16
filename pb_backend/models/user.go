package models

import (
	"encoding/json"
)

type User struct {
	ID     		int  `json: "id"`
	FirstName   string 	`json: "name"`
	LastName 	string 	`json: "surname"`
	AccessToken string 	`json: "token"`
	Faculty		string 	`json: "faculty"`
}

func NewUser(id int, name string, surname string, token string) *User {
	return &User{
		ID: id,
		FirstName: name,
		LastName: surname,
		AccessToken: token,
		Faculty: "",
	} 
}

type UserSerializationService interface {
	SerializeUser() ([]byte, error)
	DeserializeUser(data []byte) error
}

func (usr *User) SerializeUser() ([]byte, error) {
	return json.Marshal(usr)
}

func (usr *User) DeserializeUser(data []byte) error {
	err := json.Unmarshal(data, usr)
	return err
}
