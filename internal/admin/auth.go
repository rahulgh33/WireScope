package admin

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents successful login
type LoginResponse struct {
	User      *UserResponse `json:"user"`
	CSRFToken string        `json:"csrf_token"`
}

// UserResponse represents user information
type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// RegisterAuthRoutes registers authentication routes
func (s *Service) RegisterAuthRoutes(router *mux.Router) {
	auth := router.PathPrefix("/api/v1/auth").Subrouter()
	auth.HandleFunc("/login", s.handleLogin).Methods("POST")
	auth.HandleFunc("/logout", s.handleLogout).Methods("POST")
	auth.HandleFunc("/me", s.handleMe).Methods("GET")
	auth.HandleFunc("/refresh", s.handleRefresh).Methods("POST")
	auth.HandleFunc("/ws-token", s.handleWSToken).Methods("GET")
}

// handleLogin processes user login
func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate credentials using UserStore
	authUser, err := s.userStore.ValidateCredentials(req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	user := &UserResponse{
		ID:       authUser.ID,
		Username: authUser.Username,
		Role:     authUser.Role,
	}

	// Generate CSRF token
	csrfToken, err := generateCSRFToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate CSRF token")
		return
	}

	// Set session cookie (httpOnly for security)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    csrfToken,
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   s.config.TLS.Enabled, // Enable secure flag when TLS is enabled
		SameSite: http.SameSiteLaxMode,
	})

	// Set refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    csrfToken + "_refresh",
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   s.config.TLS.Enabled,
		SameSite: http.SameSiteStrictMode,
	})

	// Return response with user info and CSRF token
	response := LoginResponse{
		User:      user,
		CSRFToken: csrfToken,
	}

	respondJSON(w, http.StatusOK, response)
}

// handleLogout clears session
func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleMe returns current user info
func (s *Service) handleMe(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, validate session token and get user from database
	// For demo purposes, return a mock user
	cookie, err := r.Cookie("session_token")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	if cookie.Value == "" {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	user := &UserResponse{
		ID:       "admin-001",
		Username: "admin",
		Role:     "admin",
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

// handleRefresh refreshes the session token
func (s *Service) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// Check refresh token cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "No refresh token")
		return
	}

	if cookie.Value == "" {
		respondError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// Generate new CSRF token
	csrfToken, err := generateCSRFToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate CSRF token")
		return
	}

	// Rotate session token
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    csrfToken,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   s.config.TLS.Enabled,
		SameSite: http.SameSiteLaxMode,
	})

	respondJSON(w, http.StatusOK, map[string]string{"csrf_token": csrfToken})
}

// handleWSToken generates a token for WebSocket authentication
func (s *Service) handleWSToken(w http.ResponseWriter, r *http.Request) {
	// Verify user is authenticated
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Generate WebSocket token
	token, err := generateCSRFToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"expires_in": 3600, // 1 hour
	})
}

// generateCSRFToken creates a random CSRF token
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
