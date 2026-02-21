package session

import (
	"net/http"
	"time"
)

/*
cookie.go manages session cookie lifecycle with secure defaults.
SetCookie() issues __Host- prefixed session cookies with normalized options enforcing HttpOnly.
ClearCookie() removes session cookies by setting MaxAge to -1.
CookieOptions.normalize() applies safe defaults without breaking caller configuration.
*/
const (
	CookieName = "__Host-session"
)

// CookieOptions defines how session cookies are issued.
type CookieOptions struct {
	Path     string
	HttpOnly bool
	Secure   bool
	SameSite http.SameSite
	Domain   string // should usually be empty for __Host- cookies
}

// normalize applies safe defaults without breaking callers
func (o CookieOptions) normalize() CookieOptions {
	if o.Path == "" {
		o.Path = "/" // required for __Host-
	}
	if !o.HttpOnly {
		o.HttpOnly = true // secure default
	}
	return o
}

// SetCookie issues the session cookie to the client.
func SetCookie(
	w http.ResponseWriter,
	sessionID string,
	expiresAt time.Time,
	opts CookieOptions,
) {
	opts = opts.normalize()

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sessionID,
		Path:     opts.Path,
		Domain:   opts.Domain,
		Expires:  expiresAt,
		HttpOnly: opts.HttpOnly,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

// ClearCookie removes the session cookie from the client.
func ClearCookie(
	w http.ResponseWriter,
	opts CookieOptions,
) {
	opts = opts.normalize()

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   -1,
		HttpOnly: opts.HttpOnly,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}
