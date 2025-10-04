package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// In a real application, this should be loaded from config and be much more complex.
var jwtSecret = []byte("my-super-secret-key")

// hashPassword uses bcrypt to generate a hash from a plaintext password.
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// checkPasswordHash compares a plaintext password with a bcrypt hash.
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateJWT creates a new JWT token for a given user ID.
func generateJWT(userID string) (string, error) {
	// Create a new token object, specifying signing method and the claims
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 72).Unix(), // Token expires in 72 hours
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// validateJWT parses and validates a JWT token string.
// It returns the user ID (subject) from the token if it's valid.
func validateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		sub, err := claims.GetSubject()
		if err != nil {
			return "", errors.New("invalid token subject")
		}
		return sub, nil
	}

	return "", errors.New("invalid token")
}

// generateSecureToken creates a random, URL-safe string of a given length.
func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashToken creates a SHA-256 hash of a token string.
func hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}
