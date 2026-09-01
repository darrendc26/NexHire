package candidate

import (
	"errors"
	"net/http"

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
		status := http.StatusInternalServerError
		if errors.Is(err, ErrEmailUnverified) {
			status = http.StatusForbidden
		} else if err.Error() == "name and email are required" ||
			err.Error() == "share token or interview_id is required" ||
			err.Error() == "interview is closed" {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":    session.ID,
		"session_token": session.RawToken,
		"status":        string(session.Status),
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
		"interview":   interview,
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

type SendEmailOTPRequest struct {
	Email       string `json:"email" binding:"required,email"`
	InterviewID string `json:"interview_id" binding:"required"`
}

func (h *Handler) SendEmailOTP(c *gin.Context) {
	var req SendEmailOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SendEmailOtp(c.Request.Context(), req.Email, req.InterviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
}

type VerifyEmailOTPRequest struct {
	Email       string `json:"email" binding:"required,email"`
	InterviewID string `json:"interview_id" binding:"required"`
	OTP         string `json:"otp" binding:"required"`
}

func (h *Handler) VerifyEmailOTP(c *gin.Context) {
	var req VerifyEmailOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.VerifyEmailOTP(c.Request.Context(), req.Email, req.InterviewID, req.OTP)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrOTPExpired) || errors.Is(err, ErrOTPInvalid) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/interviews/share/:shareToken", h.GetInterviewByShareToken)

	candidates := router.Group("/candidates")

	candidates.POST("/send-otp", h.SendEmailOTP)
	candidates.POST("/verify-otp", h.VerifyEmailOTP)
	candidates.POST("", h.StartSession)
	candidates.POST("/:shareToken", h.StartSession)
	candidates.GET("/token/:shareToken", h.GetInterviewByShareToken)
	candidates.GET("/session/token/:token", h.GetSessionByToken)
	candidates.GET("/interview/:interviewID", h.GetSessionsByInterviewID)
	candidates.GET("/reports/:sessionID", h.GetReportBySessionID)
	candidates.GET("/:id", h.GetSessionByID)

	// AI turn & report routes
	candidates.POST("/sessions/:token/start", h.StartSessionQuestion)
	candidates.POST("/sessions/:token/answer", h.SubmitAnswer)
	candidates.GET("/sessions/:token/report", h.GetReport)
}
