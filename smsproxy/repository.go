package smsproxy

import (
	"errors"
	"sync"
)

type repository interface {
	update(id MessageID, newStatus MessageStatus) error
	save(id MessageID) error
	get(id MessageID) (MessageStatus, error)
}

type inMemoryRepository struct {
	db   map[MessageID]MessageStatus
	lock sync.RWMutex
}

func (r *inMemoryRepository) save(id MessageID) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, exists := r.db[id]; exists {
		return errors.New("a message with specifed ID already exists")
	}
	r.db[id] = Accepted
	return nil
}

func (r *inMemoryRepository) get(id MessageID) (MessageStatus, error) {

	r.lock.RLock()
	defer r.lock.RUnlock()

	status, exists := r.db[id]
	if !exists {
		return NotFound, nil
	}
	return status, nil
}

func (r *inMemoryRepository) update(id MessageID, newStatus MessageStatus) error {

	r.lock.Lock()
	defer r.lock.Unlock()

	currentStatus, exists := r.db[id]
	if !exists {
		return errors.New("message with specifed ID not found")
	}

	if (currentStatus != Accepted && currentStatus != Confirmed) && (currentStatus != Failed && currentStatus != Delivered) {
		r.db[id] = newStatus
		return errors.New("Message is not in ACCEPTED state")
	}

	if currentStatus == Failed || currentStatus == Delivered {
		return errors.New("messages status final. Cannot modify")
	}
	r.db[id] = newStatus

	return nil
}

func newRepository() repository {
	return &inMemoryRepository{db: make(map[MessageID]MessageStatus), lock: sync.RWMutex{}}
}
