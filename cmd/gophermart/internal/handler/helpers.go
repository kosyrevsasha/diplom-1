package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"diplom-1/cmd/gophermart/internal/config"
	"diplom-1/cmd/gophermart/internal/repository"
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v4"
	"io"
	"net/http"
	"time"
)

func ReadRequestData[T any](r *http.Request, data *T) error {

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil || len(body) == 0 {
		return err
	}
	umarshErr := json.Unmarshal(body, &data)
	if umarshErr != nil {
		return umarshErr
	}

	return nil
}

func HashPassword(pass string) string {
	h := hmac.New(sha256.New, []byte(config.Secretkey))
	h.Write([]byte(pass))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func BuildJWTString(id int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, repository.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.TokenExp)),
		},
		UserID: id,
	})

	tokenString, err := token.SignedString([]byte(config.Secretkey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func MakeErrResponse(w *http.ResponseWriter, code int, message string) {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	(*w).WriteHeader(code)
	if message != "" {
		(*w).Write([]byte(message))
	}
}

func getUserId(r *http.Request) int {
	return r.Context().Value("userId").(int)
}

// TODO: валидировать номер ?
func validateNumber(number int) error {
	return nil
}
