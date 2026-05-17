package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Store manages session persistence as JSON files in a directory.
type Store struct {
	dir         string
	maxSessions int
}

// NewStore creates a new Store that stores sessions in the given directory.
func NewStore(dir string, maxSessions int) *Store {
	return &Store{dir: dir, maxSessions: maxSessions}
}

// List returns all sessions sorted by UpdatedAt in descending order (newest first).
func (s *Store) List() ([]Session, error) {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, fmt.Errorf("create session dir %q: %w", s.dir, err)
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read session dir %q: %w", s.dir, err)
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Skip unreadable files
			fmt.Fprintf(os.Stderr, "warning: could not read session file %q: %v\n", filePath, err)
			continue
		}

		var sess Session
		if err := json.Unmarshal(data, &sess); err != nil {
			// Skip corrupted JSON
			fmt.Fprintf(os.Stderr, "warning: skipping corrupted session file %q: %v\n", filePath, err)
			continue
		}

		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// Get retrieves a single session by ID.
func (s *Store) Get(id string) (*Session, error) {
	filePath := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read session %q: %w", id, err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session %q: %w", id, err)
	}

	return &sess, nil
}

// Save persists a session to disk using atomic write (temp file + rename).
func (s *Store) Save(sess *Session) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("create session dir %q: %w", s.dir, err)
	}

	sess.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session %q: %w", sess.ID, err)
	}

	tmpPath := filepath.Join(s.dir, ".tmp."+sess.ID)
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	finalPath := filepath.Join(s.dir, sess.ID+".json")
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp to final: %w", err)
	}

	return nil
}

// Delete removes a session file from disk.
func (s *Store) Delete(id string) error {
	filePath := filepath.Join(s.dir, id+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete session %q: %w", id, err)
	}

	return nil
}
