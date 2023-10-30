package middleware

import (
	"fmt"
	"golang-api/api/responses"
	"net/http"
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		// Membersihkan token dari string "Bearer "
		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
		if tokenString == "" {
			responses.ErrorResponse(w, "Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		// Mendapatkan kunci rahasia dari environment variable
		secretKeyString := os.Getenv("SECRET_KEY")
		secretKey := []byte(secretKeyString)

		// Parse token dengan kunci rahasia
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Periksa metode tanda tangan token
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				errorMessage := fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"])
				responses.ErrorResponse(w, errorMessage, http.StatusUnauthorized)
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			errorMessage := fmt.Sprintf("Unauthorized: %v", err)
			responses.ErrorResponse(w, errorMessage, http.StatusUnauthorized)
			return
		}

		// Token valid, lanjutkan ke handler berikutnya
		next.ServeHTTP(w, r)
	})
}
