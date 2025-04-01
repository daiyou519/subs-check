package service

import (
	"errors"
	"sync"
)

var (
	ErrContentNotFound = errors.New("subscription content not found")
)

var (
	subContentStore      = make(map[int64]string)
	subContentStoreMutex sync.RWMutex
)

func StoreSubContent(subID int64, content string) error {
	subContentStoreMutex.Lock()
	defer subContentStoreMutex.Unlock()

	subContentStore[subID] = content
	return nil
}

func GetSubContent(subID int64) (string, error) {
	subContentStoreMutex.RLock()
	defer subContentStoreMutex.RUnlock()

	content, exists := subContentStore[subID]
	if !exists {
		return "", ErrContentNotFound
	}

	return content, nil
}

func DeleteSubContent(subID int64) {
	subContentStoreMutex.Lock()
	defer subContentStoreMutex.Unlock()

	delete(subContentStore, subID)
}

func ClearAllContent() {
	subContentStoreMutex.Lock()
	defer subContentStoreMutex.Unlock()

	subContentStore = make(map[int64]string)
}
