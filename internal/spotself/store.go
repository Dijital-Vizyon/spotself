package spotself

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

type Store struct {
	mu       sync.RWMutex
	dataDir  string
	filePath string
	data     manifest
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "events"), 0o755); err != nil {
		return nil, err
	}
	store := &Store{
		dataDir:  dataDir,
		filePath: filepath.Join(dataDir, "manifest.json"),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) load() error {
	file, err := os.Open(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		s.data = manifest{Events: []Event{}, Photos: []Photo{}}
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&s.data); err != nil {
		return err
	}
	s.normalizeLocked()
	return nil
}

func (s *Store) saveLocked() error {
	tmp := s.filePath + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

func (s *Store) Events() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := append([]Event{}, s.data.Events...)
	for i := range events {
		events[i].PhotoCount = s.countPhotosLocked(events[i].ID)
	}
	slices.SortFunc(events, func(a, b Event) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return events
}

func (s *Store) CreateEvent(event Event) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.AccessToken == "" {
		event.AccessToken = randomToken()
	}
	for _, existing := range s.data.Events {
		if existing.ID == event.ID || existing.Slug == event.Slug {
			return Event{}, fmt.Errorf("event already exists")
		}
	}
	s.data.Events = append(s.data.Events, event)
	if err := os.MkdirAll(s.EventDir(event.ID), 0o755); err != nil {
		return Event{}, err
	}
	return event, s.saveLocked()
}

func (s *Store) UpdateEvent(id string, update Event) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, event := range s.data.Events {
		if event.ID == id || event.Slug == id {
			if update.Name != "" {
				nextSlug := slugify(update.Name)
				for _, existing := range s.data.Events {
					if existing.ID != event.ID && existing.Slug == nextSlug {
						return Event{}, fmt.Errorf("event slug already exists")
					}
				}
				event.Name = update.Name
				event.Slug = nextSlug
			}
			event.Watermark = update.Watermark
			if update.Retention > 0 {
				event.Retention = update.Retention
			}
			s.data.Events[i] = event
			return event, s.saveLocked()
		}
	}
	return Event{}, fmt.Errorf("event not found")
}

func (s *Store) DeleteEvent(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.eventLocked(id)
	if !ok {
		return fmt.Errorf("event not found")
	}
	events := s.data.Events[:0]
	for _, existing := range s.data.Events {
		if existing.ID != event.ID {
			events = append(events, existing)
		}
	}
	photos := s.data.Photos[:0]
	for _, photo := range s.data.Photos {
		if photo.EventID != event.ID {
			photos = append(photos, photo)
		}
	}
	s.data.Events = events
	s.data.Photos = photos
	if err := os.RemoveAll(s.EventDir(event.ID)); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *Store) Event(id string) (Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, event := range s.data.Events {
		if event.ID == id || event.Slug == id {
			event.PhotoCount = s.countPhotosLocked(event.ID)
			return event, true
		}
	}
	return Event{}, false
}

func (s *Store) AddPhoto(photo Photo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.eventLocked(photo.EventID); !ok {
		return fmt.Errorf("event not found")
	}
	s.data.Photos = append(s.data.Photos, photo)
	return s.saveLocked()
}

func (s *Store) Photo(eventID, photoID string) (Photo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, photo := range s.data.Photos {
		if photo.EventID == eventID && (photo.ID == photoID || photo.FileName == photoID) {
			return photo, true
		}
	}
	return Photo{}, false
}

func (s *Store) DeletePhoto(eventID, photoID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var found Photo
	photos := s.data.Photos[:0]
	for _, photo := range s.data.Photos {
		if photo.EventID == eventID && (photo.ID == photoID || photo.FileName == photoID) {
			found = photo
			continue
		}
		photos = append(photos, photo)
	}
	if found.ID == "" {
		return fmt.Errorf("photo not found")
	}
	s.data.Photos = photos
	if err := os.Remove(s.PhotoPath(found)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return s.saveLocked()
}

func (s *Store) Photos(eventID string) []Photo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	photos := []Photo{}
	for _, photo := range s.data.Photos {
		if photo.EventID == eventID {
			photos = append(photos, photo)
		}
	}
	slices.SortFunc(photos, func(a, b Photo) int {
		return b.UploadedAt.Compare(a.UploadedAt)
	})
	return photos
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := Stats{
		EventCount: len(s.data.Events),
		PhotoCount: len(s.data.Photos),
	}
	for _, photo := range s.data.Photos {
		stats.TotalBytes += photo.Size
		if stats.OldestPhoto.IsZero() || photo.UploadedAt.Before(stats.OldestPhoto) {
			stats.OldestPhoto = photo.UploadedAt
		}
		if stats.NewestPhoto.IsZero() || photo.UploadedAt.After(stats.NewestPhoto) {
			stats.NewestPhoto = photo.UploadedAt
		}
	}
	return stats
}

func (s *Store) PurgeExpired(now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiredEvents := map[string]bool{}
	for _, event := range s.data.Events {
		if event.Retention > 0 && now.After(event.CreatedAt.AddDate(0, 0, event.Retention)) {
			expiredEvents[event.ID] = true
		}
	}
	if len(expiredEvents) == 0 {
		return 0, nil
	}
	events := s.data.Events[:0]
	for _, event := range s.data.Events {
		if !expiredEvents[event.ID] {
			events = append(events, event)
		} else if err := os.RemoveAll(s.EventDir(event.ID)); err != nil {
			return 0, err
		}
	}
	removedPhotos := 0
	photos := s.data.Photos[:0]
	for _, photo := range s.data.Photos {
		if expiredEvents[photo.EventID] {
			removedPhotos++
			continue
		}
		photos = append(photos, photo)
	}
	s.data.Events = events
	s.data.Photos = photos
	return removedPhotos, s.saveLocked()
}

func (s *Store) EventDir(eventID string) string {
	return filepath.Join(s.dataDir, "events", eventID)
}

func (s *Store) PhotoPath(photo Photo) string {
	return filepath.Join(s.EventDir(photo.EventID), photo.FileName)
}

func (s *Store) eventLocked(id string) (Event, bool) {
	for _, event := range s.data.Events {
		if event.ID == id || event.Slug == id {
			return event, true
		}
	}
	return Event{}, false
}

func (s *Store) countPhotosLocked(eventID string) int {
	count := 0
	for _, photo := range s.data.Photos {
		if photo.EventID == eventID {
			count++
		}
	}
	return count
}

func (s *Store) normalizeLocked() {
	if s.data.Events == nil {
		s.data.Events = []Event{}
	}
	for i := range s.data.Events {
		if s.data.Events[i].AccessToken == "" {
			s.data.Events[i].AccessToken = randomToken()
		}
	}
	if s.data.Photos == nil {
		s.data.Photos = []Photo{}
	}
}
