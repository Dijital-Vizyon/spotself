package spotself

import (
	"archive/zip"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	cfg   Config
	store *Store
}

func NewServer(cfg Config) (*Server, error) {
	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}
	if cfg.PublicURL == "" {
		cfg.PublicURL = "http://localhost:8080"
	}
	if cfg.MaxUploadMB <= 0 {
		cfg.MaxUploadMB = 64
	}
	if cfg.MaxImagePixels <= 0 {
		cfg.MaxImagePixels = 24000000
	}
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, store: store}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin.html", http.StatusFound)
	})
	mux.Handle("/", noCache(http.FileServer(http.Dir("web"))))
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/maintenance/purge", s.handlePurge)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/events/", s.handleEvent)
	mux.HandleFunc("/media/", s.handleMedia)
	return logRequests(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.adminAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, s.store.Stats())
}

func (s *Server) handlePurge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.adminAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	removed, err := s.store.PurgeExpired(time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"removedPhotos": removed})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !s.adminAuthorized(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		events := s.store.Events()
		for i := range events {
			s.decorateEvent(&events[i])
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": events})
	case http.MethodPost:
		if !s.adminAuthorized(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		var req struct {
			Name      string `json:"name"`
			Watermark string `json:"watermark"`
			Retention int    `json:"retentionDays"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "event name is required")
			return
		}
		event := Event{
			ID:          newID("evt"),
			Name:        req.Name,
			Slug:        slugify(req.Name),
			AccessToken: randomToken(),
			Watermark:   strings.TrimSpace(req.Watermark),
			Retention:   req.Retention,
			CreatedAt:   time.Now().UTC(),
		}
		if event.Retention <= 0 {
			event.Retention = 30
		}
		created, err := s.store.CreateEvent(event)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		s.decorateEvent(&created)
		writeJSON(w, http.StatusCreated, created)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/events/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	event, ok := s.store.Event(parts[0])
	if !ok {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	if len(parts) == 1 {
		s.handleEventResource(w, r, event)
		return
	}

	switch parts[1] {
	case "photos":
		if len(parts) == 3 {
			s.handlePhotoResource(w, r, event, parts[2])
			return
		}
		if r.Method == http.MethodGet {
			s.handleListPhotos(w, r, event)
			return
		}
		if r.Method == http.MethodPost {
			s.handleUploadPhotos(w, r, event)
			return
		}
	case "match":
		if r.Method == http.MethodPost {
			s.handleMatch(w, r, event)
			return
		}
	case "download":
		if r.Method == http.MethodGet {
			s.handleDownload(w, r, event)
			return
		}
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *Server) handleEventResource(w http.ResponseWriter, r *http.Request, event Event) {
	switch r.Method {
	case http.MethodGet:
		if !s.adminAuthorized(r) && !s.eventAuthorized(r, event) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		s.decorateEvent(&event)
		writeJSON(w, http.StatusOK, event)
	case http.MethodPatch:
		if !s.adminAuthorized(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		var req struct {
			Name      string `json:"name"`
			Watermark string `json:"watermark"`
			Retention int    `json:"retentionDays"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		updated, err := s.store.UpdateEvent(event.ID, Event{
			Name:      strings.TrimSpace(req.Name),
			Watermark: strings.TrimSpace(req.Watermark),
			Retention: req.Retention,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.decorateEvent(&updated)
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if !s.adminAuthorized(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err := s.store.DeleteEvent(event.ID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, PATCH, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handlePhotoResource(w http.ResponseWriter, r *http.Request, event Event, photoID string) {
	photo, ok := s.store.Photo(event.ID, filepath.Base(photoID))
	if !ok {
		writeError(w, http.StatusNotFound, "photo not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !s.adminAuthorized(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		s.decoratePhoto(&photo)
		writeJSON(w, http.StatusOK, photo)
	case http.MethodDelete:
		if !s.adminAuthorized(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err := s.store.DeletePhoto(event.ID, photo.ID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleListPhotos(w http.ResponseWriter, r *http.Request, event Event) {
	if !s.adminAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	photos := s.store.Photos(event.ID)
	for i := range photos {
		s.decoratePhoto(&photos[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"photos": photos})
}

func (s *Server) handleUploadPhotos(w http.ResponseWriter, r *http.Request, event Event) {
	if !s.adminAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	r.Body = http.MaxBytesReader(nil, r.Body, int64(s.cfg.MaxUploadMB)<<20)
	if err := r.ParseMultipartForm(int64(s.cfg.MaxUploadMB) << 20); err != nil {
		writeError(w, http.StatusBadRequest, "multipart upload is required")
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		files = r.MultipartForm.File["photo"]
	}
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "no photos uploaded")
		return
	}

	var uploaded []Photo
	for _, header := range files {
		photo, err := s.saveUploadedPhoto(event, headerFilename(header.Filename), func() (io.ReadCloser, error) {
			return header.Open()
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		uploaded = append(uploaded, photo)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"photos": uploaded})
}

func (s *Server) handleMatch(w http.ResponseWriter, r *http.Request, event Event) {
	if !s.eventAuthorized(r, event) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	r.Body = http.MaxBytesReader(nil, r.Body, int64(s.cfg.MaxUploadMB)<<20)
	if err := r.ParseMultipartForm(int64(s.cfg.MaxUploadMB) << 20); err != nil {
		writeError(w, http.StatusBadRequest, "multipart upload is required")
		return
	}
	files := r.MultipartForm.File["selfie"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "selfie file is required")
		return
	}
	src, err := files[0].Open()
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read selfie")
		return
	}
	defer src.Close()
	hash, err := fingerprintLimited(src, s.cfg.MaxImagePixels)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	threshold := 0.56
	if value := r.FormValue("threshold"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed >= 0 && parsed <= 1 {
			threshold = parsed
		}
	}

	var matches []Match
	for _, photo := range s.store.Photos(event.ID) {
		score := similarity(hash, photo.Fingerprint)
		if score >= threshold {
			s.decoratePhoto(&photo)
			s.decoratePhotoWithToken(&photo, event.AccessToken)
			matches = append(matches, Match{Photo: photo, Similarity: score})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})
	writeJSON(w, http.StatusOK, map[string]any{"matches": matches, "threshold": threshold})
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/media/"), "/")
	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	event, ok := s.store.Event(parts[0])
	if !ok {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	if !s.adminAuthorized(r) && !s.eventAuthorized(r, event) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	fileName := filepath.Base(parts[1])
	for _, photo := range s.store.Photos(event.ID) {
		if photo.FileName == fileName {
			http.ServeFile(w, r, s.store.PhotoPath(photo))
			return
		}
	}
	writeError(w, http.StatusNotFound, "photo not found")
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request, event Event) {
	if !s.adminAuthorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	photos := s.store.Photos(event.ID)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", event.Slug+"-photos.zip"))
	zw := zip.NewWriter(w)
	defer zw.Close()
	for _, photo := range photos {
		src, err := os.Open(s.store.PhotoPath(photo))
		if err != nil {
			continue
		}
		dst, err := zw.Create(safeZipName(photo.OriginalName))
		if err == nil {
			_, _ = io.Copy(dst, src)
		}
		src.Close()
	}
}

func (s *Server) saveUploadedPhoto(event Event, originalName string, open func() (io.ReadCloser, error)) (Photo, error) {
	src, err := open()
	if err != nil {
		return Photo{}, err
	}
	hash, err := fingerprintLimited(src, s.cfg.MaxImagePixels)
	src.Close()
	if err != nil {
		return Photo{}, err
	}

	src, err = open()
	if err != nil {
		return Photo{}, err
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		ext = ".jpg"
	}
	if !allowedImageExtension(ext) {
		return Photo{}, fmt.Errorf("unsupported image type %q", ext)
	}
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	photo := Photo{
		ID:           newID("img"),
		EventID:      event.ID,
		FileName:     newID("file") + ext,
		OriginalName: originalName,
		ContentType:  contentType,
		Fingerprint:  hash,
		UploadedAt:   time.Now().UTC(),
	}
	dstPath := s.store.PhotoPath(photo)
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return Photo{}, err
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return Photo{}, err
	}
	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return Photo{}, copyErr
	}
	if closeErr != nil {
		return Photo{}, closeErr
	}
	photo.Size = written
	if err := s.store.AddPhoto(photo); err != nil {
		return Photo{}, err
	}
	s.decoratePhoto(&photo)
	return photo, nil
}

func (s *Server) decorateEvent(event *Event) {
	event.GuestURL = strings.TrimRight(s.cfg.PublicURL, "/") + "/?event=" + event.ID + "&token=" + event.AccessToken
	event.DownloadURL = strings.TrimRight(s.cfg.PublicURL, "/") + "/api/events/" + event.ID + "/download"
}

func (s *Server) decoratePhoto(photo *Photo) {
	photo.URL = "/media/" + photo.EventID + "/" + photo.FileName
}

func (s *Server) decoratePhotoWithToken(photo *Photo, token string) {
	photo.URL = "/media/" + photo.EventID + "/" + photo.FileName + "?token=" + token
}

func (s *Server) adminAuthorized(r *http.Request) bool {
	if s.cfg.AllowNoAuth {
		return true
	}
	if s.cfg.AdminToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.cfg.AdminToken)) == 1
}

func (s *Server) eventAuthorized(r *http.Request, event Event) bool {
	if s.adminAuthorized(r) {
		return true
	}
	token := r.URL.Query().Get("token")
	if event.AccessToken == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(event.AccessToken)) == 1
}

func allowedImageExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func newID(prefix string) string {
	return prefix + "_" + randomToken()
}

func randomToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return newID("event")
	}
	return slug
}

func headerFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" {
		return "upload.jpg"
	}
	return name
}

func safeZipName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" {
		return "photo"
	}
	return strings.ReplaceAll(name, "\\", "_")
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/media/") {
			fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}
