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
	apiKey         string
	model          string
	vertexProject  string
	vertexLocation string
	vertexModel    string
	client         *http.Client
}

func NewGemini(apiKey, model, vertexProject, vertexLocation, vertexModel string) *Gemini {
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	vertexProject = strings.TrimSpace(vertexProject)
	vertexLocation = strings.TrimSpace(vertexLocation)
	vertexModel = strings.TrimSpace(vertexModel)
	if model == "" {
		model = "gemini-2.5-flash-lite"
	}
	if vertexLocation == "" {
		vertexLocation = "us-central1"
	}
	if vertexModel == "" {
		vertexModel = "gemini-2.5-flash-lite"
	}
	return &Gemini{
		apiKey:         apiKey,
		model:          model,
		vertexProject:  vertexProject,
		vertexLocation: vertexLocation,
		vertexModel:    vertexModel,
		client:         &http.Client{Timeout: 20 * time.Second},
	}
}

func (g *Gemini) Generate(ctx context.Context, prompt string) (string, error) {
	if g.vertexProject != "" {
		return g.generateVertex(ctx, prompt)
	}
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

func (g *Gemini) generateVertex(ctx context.Context, prompt string) (string, error) {
	token, err := g.metadataAccessToken(ctx)
	if err != nil {
		return "", err
	}
	body := map[string]any{
		"contents": []map[string]any{{
			"role":  "user",
			"parts": []map[string]string{{"text": prompt}},
		}},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := "https://" + g.vertexLocation + "-aiplatform.googleapis.com/v1/projects/" + g.vertexProject + "/locations/" + g.vertexLocation + "/publishers/google/models/" + g.vertexModel + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		detail := strings.TrimSpace(string(message))
		if detail == "" {
			return "", fmt.Errorf("vertex gemini returned %s", res.Status)
		}
		return "", fmt.Errorf("vertex gemini returned %s: %s", res.Status, detail)
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
		return "", errors.New("vertex gemini returned no text")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

func (g *Gemini) metadataAccessToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	res, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("metadata token: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return "", fmt.Errorf("metadata token returned %s: %s", res.Status, strings.TrimSpace(string(message)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", errors.New("metadata token was empty")
	}
	return out.AccessToken, nil
}
