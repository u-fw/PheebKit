package jobs

import "fmt"

// QueueStatus describes scheduler state independently of job progress. Revision
// lets clients discard a slower request that arrives after a newer event.
type QueueStatus struct {
	Paused     bool     `json:"paused"`
	PendingIDs []string `json:"pendingIDs"`
	Revision   uint64   `json:"revision"`
}

func (m *Manager) QueueStatus() QueueStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queueStatusLocked()
}

func (m *Manager) queueStatusLocked() QueueStatus {
	return QueueStatus{Paused: m.paused, PendingIDs: append([]string{}, m.pending...), Revision: m.queueRevision}
}

// Pausing affects only jobs that have not started. A running encoder continues.
func (m *Manager) SetQueuePaused(paused bool) (QueueStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return m.queueStatusLocked(), fmt.Errorf("job manager is closed")
	}
	if m.paused != paused {
		m.paused = paused
		m.queueRevision++
	}
	if !paused {
		m.wakeLocked()
	}
	return m.queueStatusLocked(), nil
}

func (m *Manager) MoveQueued(id string, direction int) (QueueStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return m.queueStatusLocked(), fmt.Errorf("job manager is closed")
	}
	if direction != -1 && direction != 1 {
		return m.queueStatusLocked(), fmt.Errorf("queue direction must be -1 or 1")
	}
	for i, pendingID := range m.pending {
		if id != pendingID {
			continue
		}
		next := i + direction
		if next < 0 || next >= len(m.pending) {
			return m.queueStatusLocked(), fmt.Errorf("job is already at the queue boundary")
		}
		m.pending[i], m.pending[next] = m.pending[next], m.pending[i]
		m.queueRevision++
		return m.queueStatusLocked(), nil
	}
	return m.queueStatusLocked(), fmt.Errorf("job is no longer waiting in the queue")
}

func (m *Manager) wakeLocked() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}
