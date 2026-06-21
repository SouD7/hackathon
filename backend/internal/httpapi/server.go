package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"campus-market/backend/internal/ai"
	"campus-market/backend/internal/app"
	"campus-market/backend/internal/db"
)

type Server struct {
	cfg    app.Config
	store  *db.Store
	gemini *ai.Gemini
}

type contextUserKey struct{}

func NewServer(cfg app.Config, store *db.Store) http.Handler {
	s := &Server{cfg: cfg, store: store, gemini: ai.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.VertexProject, cfg.VertexLocation, cfg.VertexModel)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.Handle("GET /api/me", s.requireAuth(http.HandlerFunc(s.me)))
	mux.Handle("POST /api/me/profile", s.requireAuth(http.HandlerFunc(s.updateProfile)))
	mux.Handle("GET /api/users/", s.requireAuth(http.HandlerFunc(s.publicUser)))
	mux.HandleFunc("GET /api/listings", s.listListings)
	mux.Handle("POST /api/listings", s.requireAuth(http.HandlerFunc(s.createListing)))
	mux.Handle("POST /api/listings/", s.requireAuth(http.HandlerFunc(s.listingAction)))
	mux.Handle("GET /api/conversations", s.requireAuth(http.HandlerFunc(s.listConversations)))
	mux.Handle("POST /api/conversations", s.requireAuth(http.HandlerFunc(s.startConversation)))
	mux.Handle("GET /api/conversations/", s.requireAuth(http.HandlerFunc(s.conversationAction)))
	mux.Handle("POST /api/conversations/", s.requireAuth(http.HandlerFunc(s.conversationAction)))
	mux.Handle("GET /api/notifications/purchases", s.requireAuth(http.HandlerFunc(s.listPurchaseNotifications)))
	mux.Handle("POST /api/notifications/purchases/", s.requireAuth(http.HandlerFunc(s.purchaseNotificationAction)))
	mux.Handle("POST /api/ai/description", s.requireAuth(http.HandlerFunc(s.generateDescription)))
	return s.withMiddleware(mux)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.cfg.CORSOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("X-Request-ID", randomRequestID())
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		claims, err := s.parseToken(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		user, err := s.store.FindUserByID(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "user not found")
			return
		}
		ctx := context.WithValue(r.Context(), contextUserKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "name, email and password(8+ chars) are required")
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	user, err := s.store.CreateUser(r.Context(), req.Name, req.Email, hash)
	if err != nil {
		writeError(w, http.StatusConflict, "email may already be registered")
		return
	}
	token, err := s.issueToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user, "token": token})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := s.store.FindUserByEmail(r.Context(), req.Email)
	if err != nil || !verifyPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := s.issueToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "token": token})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	var req struct {
		Name            string `json:"name"`
		Bio             string `json:"bio"`
		ProfileImageURL string `json:"profile_image_url"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	bio := strings.TrimSpace(req.Bio)
	if name == "" || len([]rune(name)) > 20 {
		writeError(w, http.StatusBadRequest, "name is required and must be 20 chars or fewer")
		return
	}
	if len([]rune(bio)) > 1000 {
		writeError(w, http.StatusBadRequest, "bio must be 1000 chars or fewer")
		return
	}
	if len(req.ProfileImageURL) > 750000 {
		writeError(w, http.StatusBadRequest, "profile image is too large")
		return
	}
	updated, err := s.store.UpdateProfile(r.Context(), user.ID, name, bio, req.ProfileImageURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) publicUser(w http.ResponseWriter, r *http.Request) {
	id, err := userIDFromPath(r.URL.Path, "/api/users/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	user, err := s.store.FindPublicUserByID(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                user.ID,
		"name":              user.Name,
		"profile_image_url": user.ProfileImageURL,
		"bio":               user.Bio,
	})
}

func (s *Server) listListings(w http.ResponseWriter, r *http.Request) {
	listings, err := s.store.ListListings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list items")
		return
	}
	writeJSON(w, http.StatusOK, listings)
}

func (s *Server) createListing(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Price       int      `json:"price"`
		ImageURL    string   `json:"image_url"`
		ImageURLs   []string `json:"image_urls"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Title) == "" || req.Price <= 0 {
		writeError(w, http.StatusBadRequest, "title and positive price are required")
		return
	}
	if len(req.ImageURLs) > 5 {
		writeError(w, http.StatusBadRequest, "images must be 5 or fewer")
		return
	}
	if len(req.ImageURLs) == 0 && req.ImageURL != "" {
		req.ImageURLs = []string{req.ImageURL}
	}
	if req.ImageURL == "" && len(req.ImageURLs) > 0 {
		req.ImageURL = req.ImageURLs[0]
	}
	for _, imageURL := range req.ImageURLs {
		if len(imageURL) > 750000 {
			writeError(w, http.StatusBadRequest, "image is too large")
			return
		}
	}
	if len(req.ImageURL) > 750000 {
		writeError(w, http.StatusBadRequest, "image is too large")
		return
	}
	listing, err := s.store.CreateListing(r.Context(), user.ID, req.Title, req.Description, req.Price, req.ImageURL, req.ImageURLs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create listing")
		return
	}
	writeJSON(w, http.StatusCreated, listing)
}

func (s *Server) listingAction(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/listings/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing id")
		return
	}
	user, _ := s.currentUser(r)
	if parts[1] == "cancel" {
		listing, err := s.store.CancelListing(r.Context(), id, user.ID)
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusConflict, "item is unavailable or does not belong to you")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not cancel item")
			return
		}
		writeJSON(w, http.StatusOK, listing)
		return
	}
	if parts[1] != "purchase" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req struct {
		ShippingAddress string `json:"shipping_address"`
		RecipientName   string `json:"recipient_name"`
	}
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	messageBody := purchaseMessage(req.ShippingAddress, req.RecipientName)
	result, err := s.store.PurchaseListing(r.Context(), id, user.ID, messageBody)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusConflict, "item is unavailable or belongs to you")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not purchase item")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func purchaseMessage(address, recipient string) string {
	address = strings.TrimSpace(address)
	recipient = strings.TrimSpace(recipient)
	if address == "" {
		address = "未設定"
	}
	if recipient == "" {
		recipient = "未設定"
	}
	return "配送先: " + recipient + "\n" + address
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	conversations, err := s.store.ListConversations(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list conversations")
		return
	}
	writeJSON(w, http.StatusOK, conversations)
}

func (s *Server) startConversation(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	var req struct {
		ListingID int64 `json:"listing_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	conversation, err := s.store.StartConversation(r.Context(), req.ListingID, user.ID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "listing not found or belongs to you")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start conversation")
		return
	}
	writeJSON(w, http.StatusCreated, conversation)
}

func (s *Server) conversationAction(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "messages" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	conversationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	user, _ := s.currentUser(r)
	if r.Method == http.MethodGet {
		messages, err := s.store.ListMessages(r.Context(), conversationID, user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not list messages")
			return
		}
		writeJSON(w, http.StatusOK, messages)
		return
	}
	var req struct {
		Body          string `json:"body"`
		AttachmentURL string `json:"attachment_url"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Body) == "" && strings.TrimSpace(req.AttachmentURL) == "" {
		writeError(w, http.StatusBadRequest, "body or attachment is required")
		return
	}
	if len(req.AttachmentURL) > 750000 {
		writeError(w, http.StatusBadRequest, "attachment is too large")
		return
	}
	message, err := s.store.CreateMessage(r.Context(), conversationID, user.ID, req.Body, req.AttachmentURL)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusForbidden, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create message")
		return
	}
	writeJSON(w, http.StatusCreated, message)
}

func (s *Server) listPurchaseNotifications(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	notifications, err := s.store.ListUnreadPurchaseNotifications(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list notifications")
		return
	}
	writeJSON(w, http.StatusOK, notifications)
}

func (s *Server) purchaseNotificationAction(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/read") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	id, err := userIDFromPath(strings.TrimSuffix(r.URL.Path, "/read"), "/api/notifications/purchases/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	user, _ := s.currentUser(r)
	if err := s.store.MarkPurchaseNotificationRead(r.Context(), id, user.ID); errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update notification")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
