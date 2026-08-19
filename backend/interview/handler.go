package interview

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(c *gin.Context) {
	recruiterID, exists := c.Get("userID")

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, ok := recruiterID.(string)

	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user",
		})
		return
	}

	var req CreateInterviewRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	interview, err := h.service.Create(
		c.Request.Context(),
		userID,
		req,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"interview":  interview,
		"share_link": "http://localhost:8080/api/candidates/token/" + interview.ShareToken,
	})
}

func (h *Handler) GetMyInterviews(c *gin.Context) {
	userID, exists := c.Get("userID")

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	recruiterID, ok := userID.(string)

	if !ok || recruiterID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user",
		})
		return
	}

	interviews, err := h.service.GetByRecruiterID(c.Request.Context(), recruiterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"interviews": interviews,
	})
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	rawUserID, exists := c.Get("userID")

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	recruiterID, ok := rawUserID.(string)

	if !ok || recruiterID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user",
		})
		return
	}

	interview, err := h.service.GetByID(c.Request.Context(), id, recruiterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"interview": interview,
	})
}

func (h *Handler) RegisterRoutes(
	router *gin.RouterGroup,
	authMiddleware gin.HandlerFunc,
) {

	interviews := router.Group("/interviews")

	interviews.Use(authMiddleware)

	{
		interviews.POST("", h.Create)
		interviews.GET("", h.GetMyInterviews)
		interviews.GET("/:id", h.Get)
	}
}
