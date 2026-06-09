package spotself

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEventUploadAndMatch(t *testing.T) {
	server, err := NewServer(Config{DataDir: t.TempDir(), PublicURL: "http://example.test", AllowNoAuth: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Routes()

	body := bytes.NewBufferString(`{"name":"Demo Event","retentionDays":14}`)
	req := httptest.NewRequest(http.MethodPost, "/api/events", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var event Event
	if err := json.NewDecoder(rec.Body).Decode(&event); err != nil {
		t.Fatal(err)
	}

	uploadBody, uploadType := multipartImage(t, "photos", "photo.png")
	req = httptest.NewRequest(http.MethodPost, "/api/events/"+event.ID+"/photos", uploadBody)
	req.Header.Set("Content-Type", uploadType)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body = %s", rec.Code, rec.Body.String())
	}

	matchBody, matchType := multipartImage(t, "selfie", "selfie.png")
	req = httptest.NewRequest(http.MethodPost, "/api/events/"+event.ID+"/match?token="+event.AccessToken, matchBody)
	req.Header.Set("Content-Type", matchType)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("match status = %d body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Matches []Match `json:"matches"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(response.Matches))
	}
}

func TestEmptyEventsReturnsArray(t *testing.T) {
	server, err := NewServer(Config{DataDir: t.TempDir(), PublicURL: "http://example.test", AllowNoAuth: true})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "{\"events\":[]}\n" {
		t.Fatalf("body = %q, want events array", got)
	}
}

func TestAdminTokenProtectsWrites(t *testing.T) {
	server, err := NewServer(Config{DataDir: t.TempDir(), PublicURL: "http://example.test", AdminToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{"name":"Locked"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want unauthorized", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{"name":"Locked"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestAdminTokenProtectsEventListing(t *testing.T) {
	server, err := NewServer(Config{DataDir: t.TempDir(), PublicURL: "http://example.test", AdminToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want unauthorized", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestGuestTokenProtectsMatch(t *testing.T) {
	server, err := NewServer(Config{DataDir: t.TempDir(), PublicURL: "http://example.test", AllowNoAuth: true})
	if err != nil {
		t.Fatal(err)
	}
	event := createTestEvent(t, server)
	uploadTestPhoto(t, server, event.ID)
	server.cfg.AllowNoAuth = false

	matchBody, matchType := multipartImage(t, "selfie", "selfie.png")
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+event.ID+"/match", matchBody)
	req.Header.Set("Content-Type", matchType)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want unauthorized", rec.Code)
	}

	matchBody, matchType = multipartImage(t, "selfie", "selfie.png")
	req = httptest.NewRequest(http.MethodPost, "/api/events/"+event.ID+"/match?token="+event.AccessToken, matchBody)
	req.Header.Set("Content-Type", matchType)
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestStatsAndDeletePhoto(t *testing.T) {
	server, err := NewServer(Config{DataDir: t.TempDir(), PublicURL: "http://example.test", AllowNoAuth: true})
	if err != nil {
		t.Fatal(err)
	}
	event := createTestEvent(t, server)
	photo := uploadTestPhoto(t, server, event.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d body = %s", rec.Code, rec.Body.String())
	}
	var stats Stats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.EventCount != 1 || stats.PhotoCount != 1 || stats.TotalBytes == 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/events/"+event.ID+"/photos/"+photo.ID, nil)
	rec = httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body = %s", rec.Code, rec.Body.String())
	}
	if _, ok := server.store.Photo(event.ID, photo.ID); ok {
		t.Fatal("photo still exists after delete")
	}
}

func TestPurgeExpiredEvents(t *testing.T) {
	server, err := NewServer(Config{DataDir: t.TempDir(), PublicURL: "http://example.test", AllowNoAuth: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.store.CreateEvent(Event{
		ID:        "old",
		Name:      "Old",
		Slug:      "old",
		Retention: 1,
		CreatedAt: time.Now().UTC().AddDate(0, 0, -3),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.store.AddPhoto(Photo{
		ID:           "photo",
		EventID:      "old",
		FileName:     "photo.png",
		OriginalName: "photo.png",
		UploadedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	removed, err := server.store.PurgeExpired(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, ok := server.store.Event("old"); ok {
		t.Fatal("expired event still exists")
	}
}

func multipartImage(t *testing.T, field, fileName string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(part, sampleImage()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}

func createTestEvent(t *testing.T, server *Server) Event {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{"name":"Demo Event","retentionDays":14}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var event Event
	if err := json.NewDecoder(rec.Body).Decode(&event); err != nil {
		t.Fatal(err)
	}
	return event
}

func uploadTestPhoto(t *testing.T, server *Server, eventID string) Photo {
	t.Helper()
	uploadBody, uploadType := multipartImage(t, "photos", "photo.png")
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/photos", uploadBody)
	req.Header.Set("Content-Type", uploadType)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Photos []Photo `json:"photos"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Photos) != 1 {
		t.Fatalf("photos = %d, want 1", len(response.Photos))
	}
	return response.Photos[0]
}

func sampleImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 10), G: uint8(y * 10), B: 120, A: 255})
		}
	}
	return img
}
