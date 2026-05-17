package session

// evictOldest removes the session with the oldest UpdatedAt timestamp
// when the session count is at or above the store's capacity.
func evictOldest(store *Store, sessions []Session) error {
	if len(sessions) < store.maxSessions {
		return nil
	}

	// Find the session with the oldest UpdatedAt
	var oldest *Session
	for i := range sessions {
		if oldest == nil || sessions[i].UpdatedAt.Before(oldest.UpdatedAt) {
			oldest = &sessions[i]
		}
	}

	if oldest != nil {
		return store.Delete(oldest.ID)
	}

	return nil
}
