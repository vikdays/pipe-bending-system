package main

import (
	"sync"

	"changeme/internal/apperror"
	"changeme/internal/auth"
)

// DesktopAuthService is the only auth object exposed to the WebView.
// The refresh credential lives in native memory only; restarting requires a password.
type DesktopAuthService struct {
	mu      sync.Mutex
	manager *auth.Manager
	refresh string
}

func newDesktopAuthService(manager *auth.Manager) *DesktopAuthService {
	return &DesktopAuthService{manager: manager}
}

func (s *DesktopAuthService) NeedsSetup() (bool, error) { return publicResult(s.manager.NeedsSetup()) }

func (s *DesktopAuthService) Bootstrap(username, password string) (auth.User, error) {
	return publicResult(s.manager.Bootstrap(username, password))
}

func (s *DesktopAuthService) Login(username, password string) (auth.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pair, err := s.manager.Login(username, password)
	if err != nil {
		return auth.Session{}, apperror.Public(err)
	}
	if s.refresh != "" {
		if err = s.manager.Logout(s.refresh); err != nil {
			_ = s.manager.Logout(pair.RefreshToken)
			return auth.Session{}, apperror.Public(err)
		}
	}
	s.refresh = pair.RefreshToken
	return pair.Session, nil
}

func (s *DesktopAuthService) Refresh() (auth.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.refresh == "" {
		return auth.Session{}, auth.ErrUnauthorized
	}
	pair, err := s.manager.Refresh(s.refresh)
	if err != nil {
		// Storage failures also fail closed. Never retain a possibly consumed refresh token.
		_ = s.manager.Logout(s.refresh)
		s.refresh = ""
		return auth.Session{}, apperror.Public(err)
	}
	s.refresh = pair.RefreshToken
	return pair.Session, nil
}

func (s *DesktopAuthService) Logout() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential := s.refresh
	s.refresh = ""
	return apperror.Public(s.manager.Logout(credential))
}

func (s *DesktopAuthService) CurrentUser(access string) (auth.User, error) {
	return publicResult(s.manager.CurrentUser(access))
}
func (s *DesktopAuthService) ListUsers(access string) ([]auth.User, error) {
	return publicResult(s.manager.ListUsers(access))
}
func (s *DesktopAuthService) CreateUser(access, username, password, role string) (auth.User, error) {
	return publicResult(s.manager.CreateUser(access, username, password, role))
}
func (s *DesktopAuthService) SetUserEnabled(access, userID string, enabled bool) error {
	return apperror.Public(s.manager.SetUserEnabled(access, userID, enabled))
}

func (s *DesktopAuthService) ChangePassword(access, currentPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.manager.ChangePassword(access, currentPassword, newPassword); err != nil {
		return apperror.Public(err)
	}
	s.refresh = ""
	return nil
}

func publicResult[T any](value T, err error) (T, error) {
	if err != nil {
		var zero T
		return zero, apperror.Public(err)
	}
	return value, nil
}
