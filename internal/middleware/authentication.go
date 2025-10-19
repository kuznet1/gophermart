package middleware

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/kuznet1/gophermart/internal/config"
	"net/http"
)

var AuthCookieName = "token"

type key int

const UserIDKey key = iota

type Auth struct {
	cfg config.Config
}

func NewAuth(cfg config.Config) *Auth {
	return &Auth{cfg: cfg}
}

type claims struct {
	jwt.RegisteredClaims
	UserID int
}

func (auth *Auth) Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(AuthCookieName)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := auth.parseToken(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (auth *Auth) parseToken(tokenString string) (*claims, error) {
	claims := &claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Method.Alg())
			}
			return []byte(auth.cfg.SecretKey), nil
		})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (auth *Auth) CreateToken(userID int) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims{UserID: userID}).SignedString([]byte(auth.cfg.SecretKey))
}

func (auth *Auth) GetUserID(ctx context.Context) (int, error) {
	val := ctx.Value(UserIDKey)
	id, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("unable to get id")
	}
	return id, nil
}
