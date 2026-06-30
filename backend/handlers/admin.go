package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var reUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isValidUUID(s string) bool {
	return reUUID.MatchString(s)
}

// CreateUserRequest struct
type CreateUserRequest struct {
	FullName      string `json:"full_name"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	Role          string `json:"role"`
	EnvironmentID string `json:"environment_id"` // Optional: link to existing environment
	GroupID       string `json:"group_id"`        // Optional: link to existing group
	CompanyID     string `json:"company_id"`      // Optional: link to existing company
}

// CreateUserHandler creates a new user directly (Admin only)
func CreateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" || req.FullName == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Hash Password
		hash, err := HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}

		// Default Role
		if req.Role == "" {
			req.Role = "user"
		}

		// Insert User
		trialEnds := time.Now().Add(time.Hour * 24 * 14) // 14 days
		var userID string
		err = db.QueryRow(`
			INSERT INTO users (email, password_hash, full_name, trial_ends_at, is_verified, role)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, req.Email, hash, req.FullName, trialEnds, true, req.Role).Scan(&userID)

		if err != nil {
			log.Printf("Error creating user: %v", err)
			http.Error(w, "Error creating user (email might be taken)", http.StatusConflict)
			return
		}

		if req.EnvironmentID != "" {
			// Link to existing hierarchy
			_, err = db.Exec("INSERT INTO user_environments (user_id, environment_id, role) VALUES ($1, $2, 'user')", userID, req.EnvironmentID)
			if err != nil {
				log.Printf("Error linking user to environment: %v", err)
			}

			// If company_id provided, always set owner_id (admin explicitly chose the company)
			if req.CompanyID != "" {
				_, err = db.Exec("UPDATE companies SET owner_id = $1 WHERE id = $2", userID, req.CompanyID)
				if err != nil {
					log.Printf("Error setting company owner: %v", err)
				}
			}
		} else {
			// Auto-provision new hierarchy (original behavior)
			var envID string
			err = db.QueryRow("INSERT INTO environments (name, description) VALUES ($1, 'Ambiente Padrão') RETURNING id", "Ambiente de "+req.FullName).Scan(&envID)
			if err == nil {
				var groupID string
				db.QueryRow("INSERT INTO enterprise_groups (environment_id, name, description) VALUES ($1, 'Grupo Padrão', 'Grupo Inicial') RETURNING id", envID).Scan(&groupID)
				db.Exec("INSERT INTO user_environments (user_id, environment_id, role) VALUES ($1, $2, 'admin')", userID, envID)
				if groupID != "" {
					db.Exec("INSERT INTO companies (group_id, name, trade_name, owner_id) VALUES ($1, $2, $2, $3)", groupID, "Empresa de "+req.FullName, userID)
				}
			}
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully", "id": userID})
	}
}

// AdminUser extends User with hierarchy info for admin listing
type AdminUser struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	FullName        string    `json:"full_name"`
	IsVerified      bool      `json:"is_verified"`
	TrialEndsAt     time.Time `json:"trial_ends_at"`
	Role            string    `json:"role"`
	CreatedAt       string    `json:"created_at"`
	EnvironmentID   *string   `json:"environment_id"`
	EnvironmentName *string   `json:"environment_name"`
	GroupID         *string   `json:"group_id"`
	GroupName       *string   `json:"group_name"`
	CompanyID       *string   `json:"company_id"`
	CompanyName     *string   `json:"company_name"`
}

// ListUsersHandler returns all users with hierarchy info (Admin only)
func ListUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT DISTINCT ON (u.id)
			       u.id, u.email, u.full_name, u.is_verified, u.trial_ends_at, u.role, u.created_at,
			       e.id, e.name,
			       eg.id, eg.name,
			       c.id, c.name
			FROM users u
			LEFT JOIN user_environments ue ON u.id = ue.user_id
			LEFT JOIN environments e ON ue.environment_id = e.id
			LEFT JOIN enterprise_groups eg ON eg.environment_id = e.id
			LEFT JOIN companies c ON c.group_id = eg.id
			ORDER BY u.id, (c.owner_id = u.id) DESC NULLS LAST, c.created_at ASC NULLS LAST
		`)
		if err != nil {
			log.Printf("ListUsers error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []AdminUser
		for rows.Next() {
			var u AdminUser
			if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.IsVerified, &u.TrialEndsAt, &u.Role, &u.CreatedAt,
				&u.EnvironmentID, &u.EnvironmentName,
				&u.GroupID, &u.GroupName,
				&u.CompanyID, &u.CompanyName); err != nil {
				log.Printf("ListUsers scan error: %v", err)
				continue
			}
			users = append(users, u)
		}

		if users == nil {
			users = []AdminUser{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

// PromoteUserRequest struct
type PromoteUserRequest struct {
	Role       string `json:"role"`        // 'admin' or 'user'
	ExtendDays int    `json:"extend_days"` // Days to add to trial
	IsOfficial bool   `json:"is_official"` // If true, sets trial to 2099
	FullName   string `json:"full_name"`   // optional — rename user
}

// PromoteUserHandler updates user role or trial (Admin only)
func PromoteUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")
		if userID == "" || !isValidUUID(userID) {
			http.Error(w, "Valid User ID required", http.StatusBadRequest)
			return
		}

		var req PromoteUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Update full_name when provided (trim + min length guard)
		if trimmed := strings.TrimSpace(req.FullName); trimmed != "" {
			if len(trimmed) < 2 {
				http.Error(w, "Nome deve ter pelo menos 2 caracteres", http.StatusBadRequest)
				return
			}
			_, err := db.Exec("UPDATE users SET full_name = $1 WHERE id = $2", trimmed, userID)
			if err != nil {
				http.Error(w, "Failed to update name", http.StatusInternalServerError)
				return
			}
		}

		// Update logic
		if req.Role != "" {
			// Validar role para evitar valores arbitrários no banco (WR-03)
			if req.Role != "admin" && req.Role != "user" {
				http.Error(w, "Invalid role. Must be 'admin' or 'user'", http.StatusBadRequest)
				return
			}
			_, err := db.Exec("UPDATE users SET role = $1 WHERE id = $2", req.Role, userID)
			if err != nil {
				http.Error(w, "Failed to update role", http.StatusInternalServerError)
				return
			}
		}

		if req.IsOfficial {
			// Set to far future (Official Client)
			newEnd := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)
			_, err := db.Exec("UPDATE users SET trial_ends_at = $1 WHERE id = $2", newEnd, userID)
			if err != nil {
				http.Error(w, "Failed to update trial status", http.StatusInternalServerError)
				return
			}
		} else if req.ExtendDays > 0 {
			// Get current trial end
			var currentEnd time.Time
			err := db.QueryRow("SELECT trial_ends_at FROM users WHERE id = $1", userID).Scan(&currentEnd)
			if err != nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			// If expired, start from now. If not, add to existing.
			if currentEnd.Before(time.Now()) {
				currentEnd = time.Now()
			}
			newEnd := currentEnd.Add(time.Duration(req.ExtendDays) * 24 * time.Hour)

			_, err = db.Exec("UPDATE users SET trial_ends_at = $1 WHERE id = $2", newEnd, userID)
			if err != nil {
				http.Error(w, "Failed to update trial", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully"})
	}
}

// ReassignUserRequest struct
type ReassignUserRequest struct {
	UserID        string `json:"user_id"`
	EnvironmentID string `json:"environment_id"`
	GroupID       string `json:"group_id"`   // Optional
	CompanyID     string `json:"company_id"` // Optional
}

// ReassignUserHandler re-links a user to a different hierarchy (Admin only)
func ReassignUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ReassignUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.UserID == "" || req.EnvironmentID == "" {
			http.Error(w, "user_id and environment_id are required", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Remove old company ownership (clear owner_id where this user is owner)
		_, err = tx.Exec(`
			UPDATE companies SET owner_id = NULL
			WHERE owner_id = $1
		`, req.UserID)
		if err != nil {
			log.Printf("ReassignUser: Error clearing old company owner: %v", err)
		}

		// Remove old environment link
		_, err = tx.Exec("DELETE FROM user_environments WHERE user_id = $1", req.UserID)
		if err != nil {
			log.Printf("ReassignUser: Error removing old env link: %v", err)
			http.Error(w, "Failed to remove old environment link", http.StatusInternalServerError)
			return
		}

		// Insert new environment link com preferred_company_id se fornecido
		if req.CompanyID != "" {
			_, err = tx.Exec(`
				INSERT INTO user_environments (user_id, environment_id, role, preferred_company_id)
				VALUES ($1, $2, 'user', $3)
			`, req.UserID, req.EnvironmentID, req.CompanyID)
		} else {
			_, err = tx.Exec("INSERT INTO user_environments (user_id, environment_id, role) VALUES ($1, $2, 'user')", req.UserID, req.EnvironmentID)
		}
		if err != nil {
			log.Printf("ReassignUser: Error inserting new env link: %v", err)
			http.Error(w, "Failed to link to new environment", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit changes", http.StatusInternalServerError)
			return
		}

		log.Printf("ReassignUser: User %s reassigned to environment %s, company %s", req.UserID, req.EnvironmentID, req.CompanyID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "User reassigned successfully"})
	}
}

// DeleteUserHandler deletes a user and all their data (Admin only)
func DeleteUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")
		if userID == "" || !isValidUUID(userID) {
			http.Error(w, "Valid User ID required", http.StatusBadRequest)
			return
		}

		// Impedir auto-deleção: admin não pode deletar a si mesmo (CR-04)
		callerID := GetUserIDFromContext(r)
		if callerID != "" && callerID == userID {
			http.Error(w, "Cannot delete your own account", http.StatusForbidden)
			return
		}

		_, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
		if err != nil {
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
	}
}

