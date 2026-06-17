package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) generateDescription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title     string `json:"title"`
		Condition string `json:"condition"`
		Notes     string `json:"notes"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	prompt := "学生向けフリマアプリの商品説明を日本語で作ってください。誠実で短く、状態・おすすめ用途・受け渡し時の注意を含めてください。\n商品名: " + req.Title + "\n状態: " + req.Condition + "\nメモ: " + req.Notes
	text, err := s.gemini.Generate(r.Context(), prompt)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"description": strings.TrimSpace(text)})
}
