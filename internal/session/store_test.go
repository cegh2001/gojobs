package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// --- T-07: Session types ---

func TestNewSessionGeneratesID(t *testing.T) {
	sess := NewSession("test-model")
	if sess.ID == "" {
		t.Fatal("NewSession() ID is empty")
	}
}

func TestNewSessionSetsTimestamps(t *testing.T) {
	before := time.Now()
	sess := NewSession("test-model")
	after := time.Now()

	if sess.CreatedAt.Before(before) || sess.CreatedAt.After(after) {
		t.Fatalf("CreatedAt = %v, expected between %v and %v", sess.CreatedAt, before, after)
	}

	if sess.UpdatedAt.Before(before) || sess.UpdatedAt.After(after) {
		t.Fatalf("UpdatedAt = %v, expected between %v and %v", sess.UpdatedAt, before, after)
	}
}

func TestNewSessionSetsModel(t *testing.T) {
	sess := NewSession("gemma-4-31b-it")
	if sess.Model != "gemma-4-31b-it" {
		t.Fatalf("Model = %q, want %q", sess.Model, "gemma-4-31b-it")
	}
}

func TestNewSessionInitializesEmptyMessages(t *testing.T) {
	sess := NewSession("test-model")
	if sess.Messages == nil {
		t.Fatal("NewSession() Messages is nil, want empty slice")
	}
	if len(sess.Messages) != 0 {
		t.Fatalf("len(Messages) = %d, want 0", len(sess.Messages))
	}
}

func TestAddMessageAppendsWithTimestamp(t *testing.T) {
	sess := NewSession("test-model")
	before := time.Now()
	sess.AddMessage("user", "Hello, world!")
	after := time.Now()

	if len(sess.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(sess.Messages))
	}

	msg := sess.Messages[0]
	if msg.Role != "user" {
		t.Fatalf("Role = %q, want %q", msg.Role, "user")
	}
	if msg.Content != "Hello, world!" {
		t.Fatalf("Content = %q, want %q", msg.Content, "Hello, world!")
	}
	if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
		t.Fatalf("Timestamp = %v, expected between %v and %v", msg.Timestamp, before, after)
	}
}

func TestGenerateNameTruncatesLongMessage(t *testing.T) {
	longMsg := strings.Repeat("A", 100)
	name := generateName(longMsg)

	if len(name) > 40 {
		t.Fatalf("len(name) = %d, want <= 40", len(name))
	}

	if !strings.HasPrefix(longMsg[:40], name) {
		t.Fatalf("name %q should be prefix of first 40 chars of %q", name, longMsg[:40])
	}
}

func TestGenerateNameTruncatesShortMessage(t *testing.T) {
	shortMsg := "Hello"
	name := generateName(shortMsg)

	if name != "Hello" {
		t.Fatalf("name = %q, want %q", name, "Hello")
	}
}

func TestGenerateNameTrimsWhitespace(t *testing.T) {
	msg := "  hello world  "
	name := generateName(msg)

	if name != "hello world" {
		t.Fatalf("name = %q, want %q", name, "hello world")
	}
}

func TestGenerateNameHandlesEmpty(t *testing.T) {
	name := generateName("")
	if name != "New chat" {
		t.Fatalf("name = %q, want %q", name, "New chat")
	}
}

// --- T-08: Session store ---

func TestStoreSaveAndGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	sess := NewSession("gemma-4-31b-it")
	sess.AddMessage("user", "Hello")
	sess.AddMessage("assistant", "Hi there!")
	sess.Name = "Test Session"
	sess.JobURL = "https://example.com/job"
	sess.ProfilePath = "profiles/test.json"

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if loaded.ID != sess.ID {
		t.Fatalf("ID = %q, want %q", loaded.ID, sess.ID)
	}
	if loaded.Name != "Test Session" {
		t.Fatalf("Name = %q, want %q", loaded.Name, "Test Session")
	}
	if loaded.Model != "gemma-4-31b-it" {
		t.Fatalf("Model = %q, want %q", loaded.Model, "gemma-4-31b-it")
	}
	if loaded.JobURL != "https://example.com/job" {
		t.Fatalf("JobURL = %q, want %q", loaded.JobURL, "https://example.com/job")
	}
	if loaded.ProfilePath != "profiles/test.json" {
		t.Fatalf("ProfilePath = %q, want %q", loaded.ProfilePath, "profiles/test.json")
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Hello" {
		t.Fatalf("Messages[0].Content = %q, want %q", loaded.Messages[0].Content, "Hello")
	}
	if loaded.Messages[1].Content != "Hi there!" {
		t.Fatalf("Messages[1].Content = %q, want %q", loaded.Messages[1].Content, "Hi there!")
	}
}

func TestStoreListReturnsSessionsSortedByUpdatedAt(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	// Create sessions with different UpdatedAt times
	sess1 := NewSession("gemma-4-31b-it")
	sess1.Name = "Session 1"
	sess1.UpdatedAt = time.Now().Add(-3 * time.Hour)
	sess1.AddMessage("user", "Old")
	if err := store.Save(sess1); err != nil {
		t.Fatalf("Save(sess1) error = %v", err)
	}

	time.Sleep(10 * time.Millisecond) // ensure different timestamps

	sess2 := NewSession("gemma-4-31b-it")
	sess2.Name = "Session 2"
	sess2.UpdatedAt = time.Now().Add(-1 * time.Hour)
	sess2.AddMessage("user", "Middle")
	if err := store.Save(sess2); err != nil {
		t.Fatalf("Save(sess2) error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	sess3 := NewSession("gemma-4-31b-it")
	sess3.Name = "Session 3"
	sess3.UpdatedAt = time.Now()
	sess3.AddMessage("user", "New")
	if err := store.Save(sess3); err != nil {
		t.Fatalf("Save(sess3) error = %v", err)
	}

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("len(sessions) = %d, want 3", len(sessions))
	}

	// Should be sorted by UpdatedAt descending (newest first)
	if sessions[0].Name != "Session 3" {
		t.Fatalf("sessions[0].Name = %q, want %q", sessions[0].Name, "Session 3")
	}
	if sessions[1].Name != "Session 2" {
		t.Fatalf("sessions[1].Name = %q, want %q", sessions[1].Name, "Session 2")
	}
	if sessions[2].Name != "Session 1" {
		t.Fatalf("sessions[2].Name = %q, want %q", sessions[2].Name, "Session 1")
	}
}

func TestStoreDeleteRemovesFile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	sess := NewSession("gemma-4-31b-it")
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(dir, sess.ID+".json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("expected file to exist after Save")
	}

	if err := store.Delete(sess.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatal("expected file to not exist after Delete")
	}

	// Get should return error
	_, err := store.Get(sess.ID)
	if err == nil {
		t.Fatal("Get() after Delete expected error, got nil")
	}
}

func TestStoreGetNonExistentReturnsError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	_, err := store.Get("nonexistent-id")
	if err == nil {
		t.Fatal("Get() expected error for nonexistent session, got nil")
	}
}

func TestStoreAutoCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions", "sub")
	store := NewStore(dir, 10)

	sess := NewSession("gemma-4-31b-it")
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.ID != sess.ID {
		t.Fatalf("ID = %q, want %q", loaded.ID, sess.ID)
	}
}

func TestStoreListSkipsCorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	// Write a valid session
	sess := NewSession("gemma-4-31b-it")
	sess.Name = "Valid"
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Write a corrupted file
	corruptPath := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(corruptPath, []byte("not valid json{{{"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Should only have the valid one
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1 (corrupted file skipped)", len(sessions))
	}
	if sessions[0].Name != "Valid" {
		t.Fatalf("sessions[0].Name = %q, want %q", sessions[0].Name, "Valid")
	}
}

func TestStoreListHandlesEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sessions) != 0 {
		t.Fatalf("len(sessions) = %d, want 0", len(sessions))
	}
}

func TestStoreAtomicWritePreventsPartialFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	sess := NewSession("gemma-4-31b-it")
	sess.Name = "Atomic Test"
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check no .tmp files remain
	tmpFiles, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(tmpFiles) > 0 {
		t.Fatalf("found %d .tmp files after Save, want 0", len(tmpFiles))
	}

	// Check the .json file exists
	jsonFiles, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(jsonFiles) != 1 {
		t.Fatalf("found %d .json files after Save, want 1", len(jsonFiles))
	}
}

func TestStoreUpdateSessionPreservesID(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	sess := NewSession("gemma-4-31b-it")
	sess.AddMessage("user", "First")
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Update session
	sess.AddMessage("assistant", "Response")
	sess.Name = "Updated Name"
	sess.UpdatedAt = time.Now()
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if loaded.ID != sess.ID {
		t.Fatalf("ID = %q, want %q", loaded.ID, sess.ID)
	}
	if loaded.Name != "Updated Name" {
		t.Fatalf("Name = %q, want %q", loaded.Name, "Updated Name")
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(loaded.Messages))
	}
}

// --- T-09: LRU eviction ---

func TestLRUEvictOldestRemovesOldestWhenAtCapacity(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 5)

	var sessions []Session
	now := time.Now()

	for i := 0; i < 5; i++ {
		sess := NewSession("gemma-4-31b-it")
		sess.Name = "Session " + string(rune('A'+i))
		if err := store.Save(sess); err != nil {
			t.Fatalf("Save(%d) error = %v", i, err)
		}
		// UpdatedAt must be set AFTER Save because Save overwrites it with time.Now()
		sess.UpdatedAt = now.Add(time.Duration(i) * time.Minute)
		if err := store.Save(sess); err != nil {
			t.Fatalf("Save(%d) after UpdatedAt fix error = %v", i, err)
		}
		sessions = append(sessions, *sess)
	}

	// All 5 saved
	loaded, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(loaded) != 5 {
		t.Fatalf("len(List()) = %d, want 5", len(loaded))
	}

	// Evict oldest (Session A)
	if err := evictOldest(store, loaded); err != nil {
		t.Fatalf("evictOldest() error = %v", err)
	}

	// Verify Session A is gone
	_, err = store.Get(sessions[0].ID)
	if err == nil {
		t.Fatal("Get(oldest) expected error after eviction, got nil")
	}

	// Verify other 4 remain
	for i := 1; i < 5; i++ {
		s, err := store.Get(sessions[i].ID)
		if err != nil {
			t.Fatalf("Get(session %d) unexpected error: %v", i, err)
		}
		if s.Name != "Session "+string(rune('A'+i)) {
			t.Fatalf("Name = %q, want %q", s.Name, "Session "+string(rune('A'+i)))
		}
	}
}

func TestLRUEvictOldestDoesNothingUnderCapacity(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10)

	for i := 0; i < 3; i++ {
		sess := NewSession("gemma-4-31b-it")
		sess.Name = "Session " + string(rune('A'+i))
		if err := store.Save(sess); err != nil {
			t.Fatalf("Save(%d) error = %v", i, err)
		}
	}

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if err := evictOldest(store, sessions); err != nil {
		t.Fatalf("evictOldest() error = %v", err)
	}

	// All 3 should remain
	afterList, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(afterList) != 3 {
		t.Fatalf("len(List()) = %d, want 3 (no eviction under capacity)", len(afterList))
	}
}

func TestSessionJSONRoundTrip(t *testing.T) {
	sess := NewSession("gemma-4-31b-it")
	sess.Name = "My Chat"
	sess.AddMessage("user", "Hello")
	sess.AddMessage("assistant", "Hi!")
	sess.JobURL = "https://jobs.example.com/123"
	sess.ProfilePath = "profiles/dev.json"

	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if loaded.ID != sess.ID {
		t.Fatalf("ID = %q, want %q", loaded.ID, sess.ID)
	}
	if loaded.Name != "My Chat" {
		t.Fatalf("Name = %q, want %q", loaded.Name, "My Chat")
	}
	if loaded.Model != "gemma-4-31b-it" {
		t.Fatalf("Model = %q, want %q", loaded.Model, "gemma-4-31b-it")
	}
	if loaded.JobURL != "https://jobs.example.com/123" {
		t.Fatalf("JobURL = %q, want %q", loaded.JobURL, "https://jobs.example.com/123")
	}
	if loaded.ProfilePath != "profiles/dev.json" {
		t.Fatalf("ProfilePath = %q, want %q", loaded.ProfilePath, "profiles/dev.json")
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(loaded.Messages))
	}
	if loaded.Messages[0].Role != "user" {
		t.Fatalf("Messages[0].Role = %q, want %q", loaded.Messages[0].Role, "user")
	}
	if loaded.Messages[0].Content != "Hello" {
		t.Fatalf("Messages[0].Content = %q, want %q", loaded.Messages[0].Content, "Hello")
	}
}

// Helper to check sort order
func TestSortByUpdatedAtDesc(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{ID: "a", UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "b", UpdatedAt: now},
		{ID: "c", UpdatedAt: now.Add(-1 * time.Hour)},
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	if sessions[0].ID != "b" {
		t.Fatalf("sessions[0].ID = %q, want %q", sessions[0].ID, "b")
	}
	if sessions[1].ID != "c" {
		t.Fatalf("sessions[1].ID = %q, want %q", sessions[1].ID, "c")
	}
	if sessions[2].ID != "a" {
		t.Fatalf("sessions[2].ID = %q, want %q", sessions[2].ID, "a")
	}
}
