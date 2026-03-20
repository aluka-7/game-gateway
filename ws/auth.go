package ws

import (
	"errors"
	"github.com/aluka-7/cache"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("kX9Gxcd1-@0eV-*1")

type User struct {
	Id int64 `json:"id"`
}

type UserClaims struct {
	User User
	jwt.RegisteredClaims
}

func Intercept(ce cache.Provider, authHeader string) *UserClaims {
	claims := func(authHeader string) *UserClaims {
		if authHeader == "" {
			return nil
		}
		// Bearer <token>
		var tokenString string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			return nil
		}
		// 解析 Token
		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return nil
		}
		// 提取 claims
		claims, ok := token.Claims.(*UserClaims)
		if !ok {
			return nil
		}

		if claims.User.Id == 0 {
			return nil
		}

		// todo 校验 jwt-id 跟 redis 保存的key是否对得上
		//if claims.ID != ce.String(context.Background(), dto.GetUserTokenKey(claims.User.Id)) {
		//	return nil
		//}
		return claims
	}(authHeader)

	if claims == nil {
		return nil
	}

	if claims.User.Id == 0 {
		return nil
	}

	return claims
}
