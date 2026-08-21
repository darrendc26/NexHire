package candidate

import (
	"net/http"
	"strings"

	"nexhire/backend/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		service: s,
	}
}

type StartSessionResponse struct {
	SessionID    string `json:"session_id"`
	SessionToken string `json:"session_token"`
	Status       string `json:"status"`
}

func (h *Handler) StartSession(c *gin.Context) {
	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shareToken := c.Param("shareToken")

	session, err := h.service.StartSession(c, shareToken, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":    session.ID,
		"session_token": session.RawToken,
		"status":       string(session.Status),
		"session": StartSessionResponse{
			SessionID:    session.ID,
			SessionToken: session.RawToken,
			Status:       string(session.Status),
		},
	})
}

func (h *Handler) GetSessionByID(c *gin.Context) {
	id := c.Param("id")

	session, err := h.service.GetSessionByID(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

func (h *Handler) GetSessionByToken(c *gin.Context) {
	token := c.Param("token")

	session, err := h.service.GetSessionByToken(c, token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

func (h *Handler) GetSessionsByInterviewID(c *gin.Context) {
	interviewID := c.Param("interviewID")

	sessions, err := h.service.GetSessionsByInterviewID(c, interviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func (h *Handler) GetInterviewByShareToken(c *gin.Context) {
	shareToken := c.Param("shareToken")
	if shareToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "share token is required"})
		return
	}

	interview, err := h.service.GetInterviewByShareToken(c, shareToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"interview": interview,
		"title":       interview.Title,
		"role":        interview.Role,
		"difficulty":  interview.Difficulty,
		"duration":    interview.Duration,
		"description": interview.Description,
		"status":      interview.Status,
	})
}

type SubmitAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

func (h *Handler) StartSessionQuestion(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session token is required"})
		return
	}

	resp, err := h.service.StartSessionQuestion(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SubmitAnswer(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session token is required"})
		return
	}

	var req SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.SubmitAnswer(c.Request.Context(), token, req.Answer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetReport(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session token is required"})
		return
	}

	report, err := h.service.GetReport(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"report": report})
}

func (h *Handler) GetReportBySessionID(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session ID is required"})
		return
	}

	report, err := h.service.GetReportBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"report": report})
}

// CandidateAuthMiddleware authenticates candidate requests using Bearer raw session token and sets "candidateSession" in gin Context
func CandidateAuthMiddleware(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		rawToken := ""

		if strings.HasPrefix(authHeader, "Bearer ") {
			rawToken = strings.TrimPrefix(authHeader, "Bearer ")
		} else if headerToken := c.GetHeader("X-Candidate-Token"); headerToken != "" {
			rawToken = headerToken
		} else if paramToken := c.Query("session_token"); paramToken != "" {
			rawToken = paramToken
		}

		if rawToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: candidate session token required"})
			c.Abort()
			return
		}

		session, err := service.GetSessionByToken(c.Request.Context(), rawToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: invalid candidate session token"})
			c.Abort()
			return
		}

		if session.Status == models.SessionCompleted {
			c.JSON(http.StatusForbidden, gin.H{"error": "Candidate session has already been completed"})
			c.Abort()
			return
		}

		c.Set("candidateSession", session)
		c.Next()
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/interviews/share/:shareToken", h.GetInterviewByShareToken)

	candidates := router.Group("/candidates")

	candidates.POST("", h.StartSession)
	candidates.POST("/:shareToken", h.StartSession)
	candidates.GET("/token/:shareToken", h.GetInterviewByShareToken)
	candidates.GET("/session/token/:token", h.GetSessionByToken)
	candidates.GET("/:id", h.GetSessionByID)
	candidates.GET("/interview/:interviewID", h.GetSessionsByInterviewID)
	candidates.GET("/reports/:sessionID", h.GetReportBySessionID)

	// AI turn & report routes
	candidates.POST("/sessions/:token/start", h.StartSessionQuestion)
	candidates.POST("/sessions/:token/answer", h.SubmitAnswer)
	candidates.GET("/sessions/:token/report", h.GetReport)
}


