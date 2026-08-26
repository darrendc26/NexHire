package speech

import (
	"net/http"
	"strconv"

	"nexhire/backend/candidate"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	speechService     *Service
	candidateService  *candidate.Service
	voiceAgentService *VoiceAgentService
}

func NewHandler(speechService *Service, candidateService *candidate.Service) *Handler {
	return &Handler{
		speechService:     speechService,
		candidateService:  candidateService,
		voiceAgentService: NewVoiceAgentService(candidateService),
	}
}

type TTSRequest struct {
	Text string `json:"text"`
}

func (h *Handler) GetSTTToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session token is required"})
		return
	}

	if h.candidateService != nil {
		_, _, err := h.candidateService.ValidateActiveSession(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
	}

	sttToken, err := h.speechService.GenerateSTTToken(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": sttToken})
}

func (h *Handler) GetTTS(c *gin.Context) {
	text := c.Query("text")
	if text == "" {
		var req TTSRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.Text != "" {
			text = req.Text
		}
	}

	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	audioBytes, err := h.speechService.GenerateTTS(c.Request.Context(), text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "audio/mp3")
	c.Header("Content-Length", strconv.Itoa(len(audioBytes)))
	c.Data(http.StatusOK, "audio/mp3", audioBytes)
}

func (h *Handler) GetVoiceAgentConfig(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session token is required"})
		return
	}

	cfg, session, err := h.voiceAgentService.GetVoiceAgentConfig(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config":        cfg,
		"candidate":     session.Name,
		"session_token": token,
	})
}

func (h *Handler) ProcessVoiceAgentTool(c *gin.Context) {
	var payload ToolCallPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := c.Param("token")
	if token != "" && payload.SessionToken == "" {
		payload.SessionToken = token
	}

	result, err := h.voiceAgentService.ProcessVoiceAgentToolCall(c.Request.Context(), payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	// Candidate compatibility endpoints
	router.GET("/candidates/tts", h.GetTTS)
	router.POST("/candidates/tts", h.GetTTS)
	router.GET("/candidates/sessions/:token/stt-token", h.GetSTTToken)
	router.GET("/candidates/sessions/:token/voice-agent", h.GetVoiceAgentConfig)
	router.POST("/candidates/sessions/:token/voice-agent/tool", h.ProcessVoiceAgentTool)

	// Speech package endpoints
	speech := router.Group("/speech")
	speech.GET("/tts", h.GetTTS)
	speech.POST("/tts", h.GetTTS)
	speech.GET("/stt-token/:token", h.GetSTTToken)
	speech.GET("/agent/config/:token", h.GetVoiceAgentConfig)
	speech.POST("/agent/tool", h.ProcessVoiceAgentTool)
}
