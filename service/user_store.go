package service

import (
	"fmt"
	"sync"
)

type UserStore interface {
	Save(user *User) error
	Find(username string) (*User, error)
}

type InMemoryUserStore struct {
	mutex sync.RWMutex
	users map[string]*User
}

func NewInMemoryUserStore() UserStore {
	return &InMemoryUserStore{
		users: make(map[string]*User),
	}
}

func (store *InMemoryUserStore) Save(user *User) error {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	if store.users[user.Username] != nil {
		return ErrAlreadyExists
	}
	store.users[user.Username] = user.Clone()
	return nil
}

func (store *InMemoryUserStore) Find(username string) (*User, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	user := store.users[username]

	if user == nil {
		return nil, fmt.Errorf("cannot found user")
	}

	return user, nil
}
