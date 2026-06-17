package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Gemini struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGemini(apiKey, model string) *Gemini {
	if model == "" {
		model = "gemini-2.5-flash-lite"
	}
	return &Gemini{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (g *Gemini) Generate(ctx context.Context, prompt string) (string, error) {
	if g.apiKey == "" {
		return "Gemini API key is not configured. Set GEMINI_API_KEY to enable generation.", nil
	}
	body := map[string]any{
		"contents": []map[string]any{{
			"parts": []map[string]string{{"text": prompt}},
		}},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + g.model + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)
	res, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		detail := strings.TrimSpace(string(message))
		if detail == "" {
			return "", fmt.Errorf("gemini returned %s", res.Status)
		}
		return "", fmt.Errorf("gemini returned %s: %s", res.Status, detail)
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini returned no text")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}
