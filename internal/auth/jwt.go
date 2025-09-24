package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token has expired")
)

type JWTService struct {
	secretKey     string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

func NewJWTService(secretKey string, accessTTL, refreshTTL time.Duration) *JWTService {
	return &JWTService{
		secretKey:  secretKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     "conx-cmdb",
	}
}

func (s *JWTService) GenerateAccessToken(userID, username string, roles []string) (string, error) {
	return s.generateToken(userID, username, roles, s.accessTTL)
}

func (s *JWTService) GenerateRefreshToken(userID, username string, roles []string) (string, error) {
	return s.generateToken(userID, username, roles, s.refreshTTL)
}

func (s *JWTService) generateToken(userID, username string, roles []string, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *JWTService) RefreshAccessToken(refreshTokenString string) (string, error) {
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// Generate new access token
	return s.GenerateAccessToken(claims.UserID, claims.Username, claims.Roles)
}

func (s *JWTService) ExtractUserID(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (s *JWTService) ExtractRoles(tokenString string) ([]string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return claims.Roles, nil
}

func (s *JWTService) IsTokenExpired(tokenString string) bool {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return errors.Is(err, ErrTokenExpired)
	}
	return false
}
