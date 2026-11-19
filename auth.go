package main

import (
    "crypto/rand"
    "encoding/hex"
    "net/http"
    "sync"
)

// what we remember about a logged-in user 
type Session struct {
    UserID   int
    Username string
    Role     string
}

func (s Session) IsAdmin() bool {
    return s.Role == "admin"
}

const cookieName = "session_id"

var (
    sessions   = map[string]Session{} // token -> session data
    sessionsMu sync.RWMutex           // maps are not safe for concurrent use
)

func newToken() string {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        panic(err)
    }
    return hex.EncodeToString(b)
}

// startSession creates a brand-new token on every login.
func startSession(w http.ResponseWriter, s Session) {
    token := newToken()

    sessionsMu.Lock()
    sessions[token] = s
    sessionsMu.Unlock()

    http.SetCookie(w, &http.Cookie{
        Name:     cookieName,
        Value:    token,
        Path:     "/",
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    })
}

func currentSession(r *http.Request) (Session, bool) {
    c, err := r.Cookie(cookieName)
    if err != nil {
        return Session{}, false
    }

    sessionsMu.RLock()
    defer sessionsMu.RUnlock()

    s, ok := sessions[c.Value]
    return s, ok
}

func endSession(w http.ResponseWriter, r *http.Request) {
    if c, err := r.Cookie(cookieName); err == nil {
        sessionsMu.Lock()
        delete(sessions, c.Value)
        sessionsMu.Unlock()
    }

    // Ask the browser to delete the cookie as well.
    http.SetCookie(w, &http.Cookie{
        Name:   cookieName,
        Value:  "",
        Path:   "/",
        MaxAge: -1,
    })
}

// ---------- Middleware (the "auth guard") ----------

type protectedHandler func(
    w http.ResponseWriter,
    r *http.Request,
    s Session,
)

// requireLogin: not logged in -> redirect to /login.
func requireLogin(next protectedHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        s, ok := currentSession(r)

        if !ok {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        next(w, r, s)
    }
}


func requireAdmin(next protectedHandler) http.HandlerFunc {
    return requireLogin(func(w http.ResponseWriter, r *http.Request, s Session) {
        if !s.IsAdmin() {
            http.Error(
                w,
                "403 - Access denied. This page is for admins only.",
                http.StatusForbidden,
            )
            return
        }
        next(w, r, s)
    })
}

