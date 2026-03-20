package handler

import (
	"database/sql"
	"iam-service/internal/session"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	sessionStore session.Store
	db           *sql.DB
}

func New(db *sql.DB, store session.Store) *AdminHandler {
	return &AdminHandler{
		db:           db,
		sessionStore: store,
	}
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type UserDetailResponse struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Status         string `json:"status"`
	SessionVersion int    `json:"session_version"`
	CreatedAt      string `json:"created_at"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

func (h *AdminHandler) ListUsers(c *gin.Context) {

	rows, err := h.db.Query(`
		SELECT id, email, status, created_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch users",
		})
		return
	}
	defer rows.Close()

	var users []UserResponse

	for rows.Next() {

		var u UserResponse

		err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.Status,
			&u.CreatedAt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to scan user",
			})
			return
		}

		users = append(users, u)
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

func (h *AdminHandler) GetUser(c *gin.Context) {

	userID := c.Param("id")

	var user UserDetailResponse

	err := h.db.QueryRow(`
		SELECT id, email, status, session_version, created_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Status,
		&user.SessionVersion,
		&user.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {

	userID := c.Param("id")

	actorID := c.GetString("userID")
	if actorID == userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "admin cannot modify their own account",
		})
		return
	}

	var req UpdateUserStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if req.Status != "active" && req.Status != "disabled" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid status value",
		})
		return
	}

	_, err := h.db.Exec(`
		UPDATE users
		SET status = $1,
		    session_version = session_version + 1
		WHERE id = $2
	`, req.Status, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update user status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user status updated",
	})
}

func (h *AdminHandler) LogoutAllSessions(c *gin.Context) {

	userID := c.Param("id")

	actorID := c.GetString("userID")
	if actorID == userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "admin cannot revoke their own session",
		})
		return
	}

	err := h.sessionStore.DeleteAllUserSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to logout user sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "all user sessions revoked",
	})
}
