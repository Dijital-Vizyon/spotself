package spotself

import "time"

type Config struct {
	Addr           string
	DataDir        string
	PublicURL      string
	MaxUploadMB    int
	AdminToken     string
	AllowNoAuth    bool
	MaxImagePixels int
}

type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	AccessToken string    `json:"accessToken,omitempty"`
	Watermark   string    `json:"watermark,omitempty"`
	Retention   int       `json:"retentionDays"`
	CreatedAt   time.Time `json:"createdAt"`
	PhotoCount  int       `json:"photoCount"`
	GuestURL    string    `json:"guestUrl,omitempty"`
	DownloadURL string    `json:"downloadUrl,omitempty"`
}

type Photo struct {
	ID           string    `json:"id"`
	EventID      string    `json:"eventId"`
	FileName     string    `json:"fileName"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	Size         int64     `json:"size"`
	Fingerprint  uint64    `json:"fingerprint"`
	UploadedAt   time.Time `json:"uploadedAt"`
	URL          string    `json:"url,omitempty"`
}

type Match struct {
	Photo      Photo   `json:"photo"`
	Similarity float64 `json:"similarity"`
}

type Stats struct {
	EventCount  int       `json:"eventCount"`
	PhotoCount  int       `json:"photoCount"`
	TotalBytes  int64     `json:"totalBytes"`
	OldestPhoto time.Time `json:"oldestPhoto,omitempty"`
	NewestPhoto time.Time `json:"newestPhoto,omitempty"`
}

type manifest struct {
	Events []Event `json:"events"`
	Photos []Photo `json:"photos"`
}
