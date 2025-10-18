package jwt

import (
	"sso/internal/domain/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	jwtString := jwt.New(jwt.SigningMethodHS256)
	claims := jwtString.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["app_id"] = app.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(duration).Unix()
	token, err := jwtString.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}
	return token, nil
}
