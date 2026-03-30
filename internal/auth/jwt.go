package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// JWT constants.
const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

var (
	// ErrInvalidToken is returned when the token format is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when the token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrInvalidSignature is returned when the signature is invalid.
	ErrInvalidSignature = errors.New("invalid signature")
)

// JWTHeader represents the JWT header.
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// AccessClaims represents the claims in an access token.
type AccessClaims struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Roles    []string  `json:"roles"`
	DeviceID string    `json:"device_id"`
	Exp      int64     `json:"exp"`
	Iat      int64     `json:"iat"`
	JTI      string    `json:"jti"`
}

// RefreshClaims represents the claims in a refresh token.
type RefreshClaims struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
	SessionID string `json:"session_id"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
	JTI      string `json:"jti"`
}

// TokenPair contains both access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// JWTSigner handles JWT signing and validation.
type JWTSigner struct {
	secret []byte
}

// NewJWTSigner creates a new JWT signer with the given secret.
func NewJWTSigner(secret []byte) *JWTSigner {
	return &JWTSigner{secret: secret}
}

// base64URLEncode encodes bytes to base64url without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode decodes base64url without padding.
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// GenerateAccessToken creates a new access token.
func (j *JWTSigner) GenerateAccessToken(claims AccessClaims) (string, error) {
	now := time.Now().UTC()
	claims.Iat = now.Unix()
	claims.Exp = now.Add(AccessTokenTTL).Unix()

	header := JWTHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	encodedHeader := base64URLEncode(headerJSON)
	encodedClaims := base64URLEncode(claimsJSON)

	payload := encodedHeader + "." + encodedClaims
	signature := j.sign(payload)

	return payload + "." + signature, nil
}

// GenerateRefreshToken creates a new refresh token.
func (j *JWTSigner) GenerateRefreshToken(claims RefreshClaims) (string, error) {
	now := time.Now().UTC()
	claims.Iat = now.Unix()
	claims.Exp = now.Add(RefreshTokenTTL).Unix()

	header := JWTHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	encodedHeader := base64URLEncode(headerJSON)
	encodedClaims := base64URLEncode(claimsJSON)

	payload := encodedHeader + "." + encodedClaims
	signature := j.sign(payload)

	return payload + "." + signature, nil
}

// sign creates an HMAC-SHA256 signature.
func (j *JWTSigner) sign(payload string) string {
	h := hmac.New(sha256.New, j.secret)
	h.Write([]byte(payload))
	return base64URLEncode(h.Sum(nil))
}

// verify checks if the signature is valid.
func (j *JWTSigner) verify(payload, signature string) bool {
	expected := j.sign(payload)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// ValidateAccessToken validates an access token and returns the claims.
func (j *JWTSigner) ValidateAccessToken(token string) (*AccessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	payload := parts[0] + "." + parts[1]
	if !j.verify(payload, parts[2]) {
		return nil, ErrInvalidSignature
	}

	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims AccessClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().UTC().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the claims.
func (j *JWTSigner) ValidateRefreshToken(token string) (*RefreshClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	payload := parts[0] + "." + parts[1]
	if !j.verify(payload, parts[2]) {
		return nil, ErrInvalidSignature
	}

	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims RefreshClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().UTC().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

// GenerateTokenPair creates a new pair of access and refresh tokens.
func (j *JWTSigner) GenerateTokenPair(userID, username string, roles []string, deviceID, sessionID string) (*TokenPair, error) {
	jti := generateJTI()

	accessClaims := AccessClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		DeviceID: deviceID,
		JTI:      jti,
	}

	refreshClaims := RefreshClaims{
		UserID:    userID,
		DeviceID:  deviceID,
		SessionID: sessionID,
		JTI:       jti + ":refresh",
	}

	accessToken, err := j.GenerateAccessToken(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := j.GenerateRefreshToken(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().UTC().Add(AccessTokenTTL),
	}, nil
}

// generateJTI generates a unique token identifier.
func generateJTI() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), randomInt())
}

// randomInt returns a random int for JTI uniqueness.
func randomInt() int {
	return int(time.Now().UnixNano() % 1000000)
}
