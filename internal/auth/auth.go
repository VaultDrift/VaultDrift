package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/util"
)

// Common errors.
var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserDisabled       = errors.New("account is disabled")
	ErrUserLocked         = errors.New("account is locked")
	ErrInvalidTOTP        = errors.New("invalid TOTP code")
	ErrInvalidSession     = errors.New("invalid session")
	ErrSessionExpired     = errors.New("session expired")
)

// BruteForceConfig configures brute-force protection.
type BruteForceConfig struct {
	MaxAttempts      int
	LockoutDuration  time.Duration
	ProgressiveDelay bool
}

// DefaultBruteForceConfig returns default brute-force settings.
func DefaultBruteForceConfig() *BruteForceConfig {
	return &BruteForceConfig{
		MaxAttempts:      5,
		LockoutDuration:  15 * time.Minute,
		ProgressiveDelay: true,
	}
}

// FailedAttempt tracks failed login attempts.
type FailedAttempt struct {
	Count       int
	LastFailed  time.Time
	LockedUntil *time.Time
}

// Service handles authentication operations.
type Service struct {
	db             *db.Manager
	jwtSigner      *JWTSigner
	totp           *TOTP
	bruteConfig    *BruteForceConfig
	failedAttempts map[string]*FailedAttempt
	attemptsMu     sync.RWMutex
	sessionTTL     time.Duration
}

// ServiceOption configures the auth service.
type ServiceOption func(*Service)

// WithBruteForceConfig sets brute-force protection config.
func WithBruteForceConfig(config *BruteForceConfig) ServiceOption {
	return func(s *Service) {
		s.bruteConfig = config
	}
}

// WithSessionTTL sets the session TTL.
func WithSessionTTL(ttl time.Duration) ServiceOption {
	return func(s *Service) {
		s.sessionTTL = ttl
	}
}

// NewService creates a new authentication service.
func NewService(db *db.Manager, jwtSecret []byte, opts ...ServiceOption) *Service {
	s := &Service{
		db:             db,
		jwtSigner:      NewJWTSigner(jwtSecret),
		totp:           NewTOTP(),
		bruteConfig:    DefaultBruteForceConfig(),
		failedAttempts: make(map[string]*FailedAttempt),
		sessionTTL:     7 * 24 * time.Hour,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	Tokens       *TokenPair
	SessionID    string
	RequiresTOTP bool
	TOTPSession  string // Temporary session ID for TOTP verification
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, username, password, deviceName, deviceType, ipAddress, userAgent string) (*LoginResult, error) {
	// Check brute-force protection
	delay := s.checkBruteForce(username)
	if delay > 0 {
		time.Sleep(delay) // Progressive delay
	}

	// Get user by username
	user, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		s.recordFailedAttempt(username)
		return nil, ErrInvalidCredentials
	}

	// Check account status
	switch user.Status {
	case "disabled":
		return nil, ErrUserDisabled
	case "locked":
		return nil, ErrUserLocked
	}

	// Verify password
	valid, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !valid {
		s.recordFailedAttempt(username)
		return nil, ErrInvalidCredentials
	}

	// Clear failed attempts on successful password
	s.clearFailedAttempts(username)

	// Check if TOTP is enabled
	if user.TOTPEnabled && user.TOTPSecret != nil {
		// Return partial result requiring TOTP
		return &LoginResult{
			RequiresTOTP: true,
			TOTPSession:  s.createTOTPSession(user.ID),
		}, nil
	}

	// Create session
	sessionID := util.GenerateUUID()
	session := &db.Session{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: util.GenerateUUID(),
		DeviceName:   deviceName,
		DeviceType:   deviceType,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(s.sessionTTL),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.db.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	now := time.Now().UTC()
	if err := s.db.UpdateUser(ctx, user.ID, map[string]any{
		"last_login_at": now.Format(time.RFC3339),
	}); err != nil {
		// Non-fatal, log but continue
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Generate tokens
	roles := []string{user.Role}
	tokens, err := s.jwtSigner.GenerateTokenPair(user.ID, user.Username, roles, deviceName, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &LoginResult{
		Tokens:    tokens,
		SessionID: sessionID,
	}, nil
}

// LoginWithTOTP completes login with TOTP code.
func (s *Service) LoginWithTOTP(ctx context.Context, tempSessionID, totpCode, deviceName, deviceType, ipAddress, userAgent string) (*LoginResult, error) {
	// Verify TOTP session and get user
	userID, valid := s.verifyTOTPSession(tempSessionID)
	if !valid {
		return nil, ErrInvalidSession
	}

	// Get user
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidSession
	}

	// Validate TOTP code
	if !s.totp.ValidateCode(*user.TOTPSecret, totpCode) {
		return nil, ErrInvalidTOTP
	}

	// Clear TOTP session
	s.clearTOTPSession(tempSessionID)

	// Create session
	sessionID := util.GenerateUUID()
	session := &db.Session{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: util.GenerateUUID(),
		DeviceName:   deviceName,
		DeviceType:   deviceType,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(s.sessionTTL),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.db.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	now := time.Now().UTC()
	if err := s.db.UpdateUser(ctx, user.ID, map[string]any{
		"last_login_at": now.Format(time.RFC3339),
	}); err != nil {
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Generate tokens
	roles := []string{user.Role}
	tokens, err := s.jwtSigner.GenerateTokenPair(user.ID, user.Username, roles, deviceName, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &LoginResult{
		Tokens:    tokens,
		SessionID: sessionID,
	}, nil
}

// Refresh rotates tokens and generates a new pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwtSigner.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check session validity
	session, err := s.db.GetSessionByRefreshToken(ctx, claims.SessionID)
	if err != nil {
		return nil, ErrInvalidSession
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	// Get user for roles
	user, err := s.db.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidSession
	}

	// Delete old session
	if err := s.db.DeleteSession(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("failed to invalidate old session: %w", err)
	}

	// Create new session
	newSessionID := util.GenerateUUID()
	newSession := &db.Session{
		ID:           newSessionID,
		UserID:       user.ID,
		RefreshToken: util.GenerateUUID(),
		DeviceName:   session.DeviceName,
		DeviceType:   session.DeviceType,
		IPAddress:    session.IPAddress,
		UserAgent:    session.UserAgent,
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(s.sessionTTL),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.db.CreateSession(ctx, newSession); err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	// Generate new token pair
	roles := []string{user.Role}
	tokens, err := s.jwtSigner.GenerateTokenPair(user.ID, user.Username, roles, session.DeviceName, newSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

// Logout invalidates a session.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.db.DeleteSession(ctx, sessionID)
}

// LogoutAll invalidates all sessions for a user.
func (s *Service) LogoutAll(ctx context.Context, userID string) error {
	return s.db.DeleteSessionsByUser(ctx, userID)
}

// ValidateAccessToken validates an access token and returns claims.
func (s *Service) ValidateAccessToken(token string) (*AccessClaims, error) {
	return s.jwtSigner.ValidateAccessToken(token)
}

// checkBruteForce checks if progressive delay should be applied.
func (s *Service) checkBruteForce(username string) time.Duration {
	s.attemptsMu.RLock()
	attempt, exists := s.failedAttempts[username]
	s.attemptsMu.RUnlock()

	if !exists || attempt.Count == 0 {
		return 0
	}

	// Check if account is locked
	if attempt.LockedUntil != nil && time.Now().UTC().Before(*attempt.LockedUntil) {
		return s.bruteConfig.LockoutDuration
	}

	// Progressive delay: 1s, 2s, 4s, 8s...
	if s.bruteConfig.ProgressiveDelay {
		delay := time.Duration(1<<attempt.Count) * time.Second
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		return delay
	}

	return 0
}

// recordFailedAttempt records a failed login attempt.
func (s *Service) recordFailedAttempt(username string) {
	s.attemptsMu.Lock()
	defer s.attemptsMu.Unlock()

	attempt, exists := s.failedAttempts[username]
	if !exists {
		attempt = &FailedAttempt{}
		s.failedAttempts[username] = attempt
	}

	attempt.Count++
	attempt.LastFailed = time.Now().UTC()

	// Lock account after max attempts
	if attempt.Count >= s.bruteConfig.MaxAttempts {
		lockUntil := time.Now().UTC().Add(s.bruteConfig.LockoutDuration)
		attempt.LockedUntil = &lockUntil
	}
}

// clearFailedAttempts clears failed attempts for a username.
func (s *Service) clearFailedAttempts(username string) {
	s.attemptsMu.Lock()
	delete(s.failedAttempts, username)
	s.attemptsMu.Unlock()
}

// TOTP session management (in-memory, short-lived)

type totpSession struct {
	UserID    string
	CreatedAt time.Time
}

var (
	totpSessions   = make(map[string]*totpSession)
	totpSessionsMu sync.RWMutex
)

// createTOTPSession creates a temporary session for TOTP verification.
func (s *Service) createTOTPSession(userID string) string {
	sessionID := util.GenerateUUID()
	totpSessionsMu.Lock()
	totpSessions[sessionID] = &totpSession{
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
	}
	totpSessionsMu.Unlock()

	// Auto-expire after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		totpSessionsMu.Lock()
		delete(totpSessions, sessionID)
		totpSessionsMu.Unlock()
	}()

	return sessionID
}

// verifyTOTPSession verifies a temporary TOTP session.
func (s *Service) verifyTOTPSession(sessionID string) (string, bool) {
	totpSessionsMu.RLock()
	session, exists := totpSessions[sessionID]
	totpSessionsMu.RUnlock()

	if !exists {
		return "", false
	}

	// Check if expired (5 minutes)
	if time.Since(session.CreatedAt) > 5*time.Minute {
		totpSessionsMu.Lock()
		delete(totpSessions, sessionID)
		totpSessionsMu.Unlock()
		return "", false
	}

	return session.UserID, true
}

// clearTOTPSession removes a temporary TOTP session.
func (s *Service) clearTOTPSession(sessionID string) {
	totpSessionsMu.Lock()
	delete(totpSessions, sessionID)
	totpSessionsMu.Unlock()
}
