package json

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type UserCredentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func NewUser() *UserCredentials {
	return &UserCredentials{}
}

func UnpackingJSON(c *gin.Context) (*UserCredentials, error) {
	user := NewUser()
	if err := json.NewDecoder(c.Request.Body).Decode(user); err != nil {
		return nil, err
	}
	return user, nil
}
