package speech

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type deepgramGrantRequest struct {
	TTLSeconds int `json:"ttl_seconds"`
}

type deepgramGrantResponse struct {
	AccessToken string  `json:"access_token"`
	ExpiresIn   float64 `json:"expires_in"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GenerateSTTToken(ctx context.Context) (string, error) {
	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	if apiKey == "" {
		return "", errors.New("STT service not configured")
	}

	grantReq := deepgramGrantRequest{
		TTLSeconds: 120,
	}

	reqBytes, err := json.Marshal(grantReq)
	if err != nil {
		return apiKey, nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.deepgram.com/v1/auth/grant",
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return apiKey, nil
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return apiKey, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiKey, nil
	}

	if resp.StatusCode != http.StatusOK {
		// Fallback to configured DEEPGRAM_API_KEY if grant endpoint fails or lacks permissions
		return apiKey, nil
	}

	var grantResp deepgramGrantResponse
	if err := json.Unmarshal(body, &grantResp); err != nil || grantResp.AccessToken == "" {
		return apiKey, nil
	}

	return grantResp.AccessToken, nil
}

func (s *Service) GenerateTTS(ctx context.Context, text string) ([]byte, error) {
	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	if apiKey == "" {
		return nil, errors.New("DEEPGRAM_API_KEY not configured")
	}

	bodyMap := map[string]string{"text": text}
	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TTS request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.deepgram.com/v1/speak?model=aura-asteria-en",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact Deepgram TTS API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Deepgram TTS API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}
