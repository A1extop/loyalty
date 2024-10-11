package json

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

type UserCredentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type History struct {
	HistoryID int       `json:"id"`
	OrderId   int       `json:"order_id "`
	UserId    int       `json:"user_id "`
	Status    string    `json:"status"`
	Accrual   int       `json:"accrual,omitempty"`
	Uploaded  time.Time `json:"uploaded_at "`
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

func PackingHistoryJSON(slHistory []History) ([]byte, error) {
	for i := range slHistory {
		slHistory[i].Uploaded = slHistory[i].Uploaded.UTC()
	}
	data, err := json.Marshal(slHistory)
	if err != nil {
		return nil, err
	}
	return data, nil
}
