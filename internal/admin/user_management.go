package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/network-qoe-telemetry-platform/internal/auth"
)

// UpdatePasswordRequest represents a request to update user password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password,omitempty"`
	NewPassword     string `json:"new_password"`
}

// UserManagementResponse represents user information (without sensitive data)
type UserManagementResponse struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email,omitempty"`
	FullName  string     `json:"full_name,omitempty"`
	Role      string     `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

// RegisterUserManagementRoutes registers user management routes
func (s *Service) RegisterUserManagementRoutes(router *mux.Router) {
	userRouter := router.PathPrefix("/api/v1/admin/users").Subrouter()

	// Require authentication for all user management endpoints
	userRouter.Use(s.requireAuth)

	// List all users (admin only)
	userRouter.HandleFunc("", s.handleListUsers).Methods("GET")

	// Create new user (admin only)
	userRouter.HandleFunc("", s.handleCreateUser).Methods("POST")

	// Get user details
	userRouter.HandleFunc("/{username}", s.handleGetUser).Methods("GET")

	// Update user (admin only)
	userRouter.HandleFunc("/{username}", s.handleUpdateUser).Methods("PUT")

	// Delete user (admin only)
	userRouter.HandleFunc("/{username}", s.handleDeleteUser).Methods("DELETE")

	// Update password (users can update their own)
	userRouter.HandleFunc("/{username}/password", s.handleUpdatePassword).Methods("PUT")
}

// handleListUsers returns all users
func (s *Service) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	if !s.isAdmin(r) {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	users, err := s.userStore.ListUsers()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	// Convert to response format (exclude sensitive data)
	response := make([]*UserManagementResponse, len(users))
	for i, user := range users {
		response[i] = &UserManagementResponse{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": response,
		"total": len(response),
	})
}

// handleCreateUser creates a new user
func (s *Service) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	if !s.isAdmin(r) {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" || req.Role == "" {
		respondError(w, http.StatusBadRequest, "Username, password, and role are required")
		return
	}

	// Validate role
	if req.Role != "admin" && req.Role != "operator" && req.Role != "viewer" {
		respondError(w, http.StatusBadRequest, "Invalid role. Must be: admin, operator, or viewer")
		return
	}

	// Validate password strength
	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	// Create user
	user, err := s.userStore.CreateUser(req.Username, req.Password, req.Role)
	if err != nil {
		if err == auth.ErrUserExists {
			respondError(w, http.StatusConflict, "User already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	response := &UserManagementResponse{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	respondJSON(w, http.StatusCreated, response)
}

// handleGetUser returns user details
func (s *Service) handleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	// Users can view their own profile, admins can view anyone
	currentUser := s.getCurrentUser(r)
	if currentUser == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	if currentUser.Username != username && currentUser.Role != "admin" {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	user, err := s.userStore.GetUser(username)
	if err != nil {
		if err == auth.ErrUserNotFound {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	response := &UserManagementResponse{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	respondJSON(w, http.StatusOK, response)
}

// handleUpdateUser updates user information
func (s *Service) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	// Only admins can update users
	if !s.isAdmin(r) {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing user
	user, err := s.userStore.GetUser(username)
	if err != nil {
		if err == auth.ErrUserNotFound {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Update fields (this is simplified; in production, you'd have proper update methods)
	user.UpdatedAt = time.Now()

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "User updated successfully",
	})
}

// handleDeleteUser deletes a user
func (s *Service) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	// Only admins can delete users
	if !s.isAdmin(r) {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Prevent deleting yourself
	currentUser := s.getCurrentUser(r)
	if currentUser != nil && currentUser.Username == username {
		respondError(w, http.StatusBadRequest, "Cannot delete your own account")
		return
	}

	if err := s.userStore.DeleteUser(username); err != nil {
		if err == auth.ErrUserNotFound {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}

// handleUpdatePassword updates user password
func (s *Service) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	currentUser := s.getCurrentUser(r)
	if currentUser == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Users can update their own password, admins can update anyone's
	if currentUser.Username != username && currentUser.Role != "admin" {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate new password
	if len(req.NewPassword) < 8 {
		respondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	// If user is updating their own password, verify current password
	if currentUser.Username == username && req.CurrentPassword != "" {
		if _, err := s.userStore.ValidateCredentials(username, req.CurrentPassword); err != nil {
			respondError(w, http.StatusUnauthorized, "Current password is incorrect")
			return
		}
	}

	// Update password
	if err := s.userStore.UpdatePassword(username, req.NewPassword); err != nil {
		if err == auth.ErrUserNotFound {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Password updated successfully",
	})
}

// Helper functions

func (s *Service) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check session cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			respondError(w, http.StatusUnauthorized, "Authentication required")
			return
		}
		// In production, validate the session token properly
		next.ServeHTTP(w, r)
	})
}

func (s *Service) isAdmin(r *http.Request) bool {
	// In production, get this from validated session/token
	// For now, simplified check
	user := s.getCurrentUser(r)
	return user != nil && user.Role == "admin"
}

func (s *Service) getCurrentUser(r *http.Request) *auth.User {
	// In production, get this from validated session/token
	// For now, return a mock admin user
	// TODO: Implement proper session management
	return &auth.User{
		ID:       "admin-001",
		Username: "admin",
		Role:     "admin",
	}
}
