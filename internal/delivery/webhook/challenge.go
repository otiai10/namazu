package webhook

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ChallengeRequest struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

type ChallengeResult struct {
	Success      bool
	ErrorMessage string
	ResponseTime time.Duration
}

type Challenger struct {
	client *http.Client
}

func NewChallenger(timeout time.Duration) *Challenger {
	return &Challenger{
		client: &http.Client{Timeout: timeout},
	}
}

func (c *Challenger) VerifyURL(ctx context.Context, url, secret string) ChallengeResult {
	start := time.Now()

	token, err := generateChallengeToken()
	if err != nil {
		return ChallengeResult{
			ErrorMessage: "failed to generate challenge token",
			ResponseTime: time.Since(start),
		}
	}

	challenge := ChallengeRequest{
		Type:      "url_verification",
		Challenge: token,
	}
	body, err := json.Marshal(challenge)
	if err != nil {
		return ChallengeResult{
			ErrorMessage: fmt.Sprintf("failed to marshal challenge: %v", err),
			ResponseTime: time.Since(start),
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ChallengeResult{
			ErrorMessage: fmt.Sprintf("failed to create request: %v", err),
			ResponseTime: time.Since(start),
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", Sign(secret, body))
	req.Header.Set("User-Agent", "namazu/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return ChallengeResult{
			ErrorMessage: fmt.Sprintf("request failed: %v", err),
			ResponseTime: time.Since(start),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ChallengeResult{
			ErrorMessage: fmt.Sprintf("webhook returned status %d, expected 200", resp.StatusCode),
			ResponseTime: time.Since(start),
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ChallengeResult{
			ErrorMessage: fmt.Sprintf("failed to read response: %v", err),
			ResponseTime: time.Since(start),
		}
	}

	var challengeResp ChallengeResponse
	if err := json.Unmarshal(respBody, &challengeResp); err != nil {
		return ChallengeResult{
			ErrorMessage: fmt.Sprintf("invalid response format: %v", err),
			ResponseTime: time.Since(start),
		}
	}

	if challengeResp.Challenge != token {
		return ChallengeResult{
			ErrorMessage: "challenge response does not match",
			ResponseTime: time.Since(start),
		}
	}

	return ChallengeResult{
		Success:      true,
		ResponseTime: time.Since(start),
	}
}

func generateChallengeToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
