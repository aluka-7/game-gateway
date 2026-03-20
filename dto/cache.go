package dto

import "fmt"

const (
	UserTokenKey = "system:user:token:%d"
)

func GetUserTokenKey(userId int64) string {
	return fmt.Sprintf(UserTokenKey, userId)
}
