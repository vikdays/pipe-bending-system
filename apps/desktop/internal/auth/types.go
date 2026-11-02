package auth

import "changeme/internal/apperror"

var (
	ErrCredentials  = apperror.New(apperror.InvalidCredentials, "Не удалось войти. Проверьте логин и пароль. Если было много неудачных попыток, повторите вход через 5 минут.")
	ErrUnauthorized = apperror.New(apperror.Unauthenticated, "Сессия завершена. Войдите снова.")
	ErrForbidden    = apperror.New(apperror.Forbidden, "Недостаточно прав для этого действия.")
	ErrRateLimited  = apperror.New(apperror.RateLimited, "Слишком много неудачных попыток. Повторите действие через 5 минут.")
	ErrConfigured   = apperror.New(apperror.Conflict, "Учётная запись руководителя уже создана. Войдите в систему.")
	ErrStorage      = apperror.New(apperror.InternalError, "Не удалось выполнить операцию с локальной базой данных.")
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Enabled  bool   `json:"enabled"`
}

// Session is safe to send to the WebView. Refresh tokens never cross that boundary.
type Session struct {
	AccessToken     string `json:"accessToken"`
	AccessExpiresAt int64  `json:"accessExpiresAt"`
	User            User   `json:"user"`
}

// TokenPair is internal Go data, not a registered Wails return type.
type TokenPair struct {
	Session      Session
	RefreshToken string
}
