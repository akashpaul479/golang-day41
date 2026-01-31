// package provide implements JWT-based authentication with access and refresh tokens.
// It provides handlers for login , refresh , logout and middleware for protecting routes.

package project

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT payload.
// It includes the user's email , token_type (access/refresh),
// and standard registered clalims like expiration and issue time
type Claims struct {
	Email     string
	TokenType string
	jwt.RegisteredClaims
}

// Credentials represents the login request payload.
// It contains the user's email and password.
type Credentials struct {
	Email    string
	Password string
}

// secret key is the key to sign JWT token
var SecretKey []byte

// access and refresh Token TTL(time to live)
const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 24 * 7 * time.Hour
)

// Generate access token creates a signed JWT access token for the given email.
func GenerateAccessToken(email string) (string, error) {
	claims := &Claims{
		Email:     email,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

// Generate Refresh token creates a signed JWT refresh token for the given email
func GenerateRefreshToken(email string) (string, error) {
	claims := &Claims{
		Email:     email,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey)
}

// Set Access cookies sets the access token in an HTTP-only cookie.
func SetAccessCookies(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		Expires:  time.Now().Add(AccessTokenTTL),
		SameSite: http.SameSiteStrictMode,
	})
}

// Set refresh cookies sets the refresh token in an HTTP-only cookie.
func SetRefreshCookies(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		Expires:  time.Now().Add(RefreshTokenTTL),
		SameSite: http.SameSiteStrictMode,
	})
}

// Clear Access cookies removes the access token by setting its expiration in the past.
func ClearAccessCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
	})
}

// validation parses and validates a JWT string.
// It returns the claims if the token is valid, otherwise an error.
func Validation(tokenstr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenstr, claims, func(t *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("Token not valid")
	}
	return claims, nil
}

// Login handler handles user login requests.
// It validates credentials , generate access and refresh token , sets them in cookies and returns a success message.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}
	if creds.Email != "akashpaul@gmail.com" || creds.Password != "akash@479" {
		http.Error(w, "invalid credentials", http.StatusInternalServerError)
		return
	}

	accessToken, _ := GenerateAccessToken(creds.Email)
	refreshToken, _ := GenerateRefreshToken(creds.Email)

	SetAccessCookies(w, accessToken)
	SetRefreshCookies(w, refreshToken)

	json.NewEncoder(w).Encode(map[string]string{"message": "login succesful!"})
}

// Refresh Handler handles requests to refresh the access token.
func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "refresh token missing", http.StatusUnauthorized)
		return
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if claims.TokenType != "refresh" {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	NewAccessToken, _ := GenerateAccessToken(claims.Email)

	SetAccessCookies(w, NewAccessToken)

	json.NewEncoder(w).Encode(map[string]string{"message": "new access token generated using refresh token", "access_token": NewAccessToken})
}

// JWTMiddleware validates the access token from cookies
func JwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")
		if err != nil {
			http.Error(w, "access token missing", http.StatusUnauthorized)
			return
		}
		claims, err := Validation(cookie.Value)
		if err != nil {
			http.Error(w, "invalid or expired jwt cookie", http.StatusUnauthorized)
			return
		}
		if claims.TokenType != "access" {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		r.Header.Set("X-User-Email", claims.Email)
		next.ServeHTTP(w, r)
	})
}

// Logout Handler handles user's logout requests.
// It clears the access token cookie and returns a success message
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	ClearAccessCookies(w)

	json.NewEncoder(w).Encode(map[string]string{"message": "Logout succesful!"})
}
