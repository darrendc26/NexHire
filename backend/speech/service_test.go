package speech

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSpeechService_Initialization(t *testing.T) {
	svc := NewService()
	if svc == nil {
		t.Fatal("Expected speech.NewService() to return non-nil service")
	}

	// STT Token request should return error if DEEPGRAM_API_KEY is unset
	_, err := svc.GenerateSTTToken(context.Background())
	if err == nil {
		t.Errorf("Expected error when DEEPGRAM_API_KEY is not set")
	}

	// TTS request should return error if DEEPGRAM_API_KEY is unset
	_, err = svc.GenerateTTS(context.Background(), "test")
	if err == nil {
		t.Errorf("Expected error when DEEPGRAM_API_KEY is not set")
	}
}

func TestSpeechHandler_GetTTS_MissingText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewService()
	h := NewHandler(svc, nil)

	router := gin.New()
	api := router.Group("/api")
	h.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/candidates/tts", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing text parameter, got %d", w.Code)
	}
}

func TestVoiceAgentService_Uninitialized(t *testing.T) {
	va := NewVoiceAgentService(nil)
	ctx := context.Background()

	_, _, err := va.GetVoiceAgentConfig(ctx, "any_token")
	if err == nil {
		t.Errorf("Expected error when candidate service is nil")
	}

	_, err = va.ProcessVoiceAgentToolCall(ctx, ToolCallPayload{
		ToolName:     "submit_answer",
		SessionToken: "token",
		AnswerText:   "hello",
	})
	if err == nil {
		t.Errorf("Expected error when candidate service is nil")
	}
}

func TestVoiceAgentService_UnknownTool(t *testing.T) {
	va := NewVoiceAgentService(nil)
	// Even if candidateService is set, an unknown tool should return an error result
	res, err := va.ProcessVoiceAgentToolCall(context.Background(), ToolCallPayload{
		ToolName: "unknown_tool",
	})
	if err == nil && res.Success {
		t.Errorf("Expected failure for unknown tool")
	}
}
