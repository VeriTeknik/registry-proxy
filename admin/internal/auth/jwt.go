package auth

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// Claims represents JWT claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT operations
type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager() *JWTManager {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a default secret for development
		secret = "default-secret-key-change-in-production"
	}

	duration := 24 * time.Hour
	if expiry := os.Getenv("JWT_EXPIRY"); expiry != "" {
		if d, err := time.ParseDuration(expiry); err == nil {
			duration = d
		}
	}

	return &JWTManager{
		secretKey:     []byte(secret),
		tokenDuration: duration,
	}
}

// GenerateToken generates a new JWT token
func (m *JWTManager) GenerateToken(username string) (string, int64, error) {
	expirationTime := time.Now().Add(m.tokenDuration)
	
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expirationTime.Unix(), nil
}

// ValidateToken validates a JWT token
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword checks if a password matches the hash
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidateCredentials validates username and password
func ValidateCredentials(username, password string) error {
	// Get admin credentials from environment
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPasswordHash := os.Getenv("ADMIN_PASSWORD_HASH")

	if adminUsername == "" {
		adminUsername = "admin"
	}

	// Skip this check for production passwords
	// For initial setup, allow default password if no hash is set
	if adminPasswordHash == "" {
		if password == "admin123" && username == "admin" {
			return nil
		}
	}

	if username != adminUsername {
		return ErrInvalidCredentials
	}

	// Try to validate with hash if provided
	if adminPasswordHash != "" && adminPasswordHash != "''" {
		// Remove quotes if present
		adminPasswordHash = strings.Trim(adminPasswordHash, "'\"")
		if err := CheckPassword(password, adminPasswordHash); err == nil {
			return nil
		}
	}

	return ErrInvalidCredentials
}