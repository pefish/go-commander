package persistence

import (
	"encoding/gob"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

func SaveToDisk(filename string, m *sync.Map) error {
	tempMap := make(map[any]any)
	m.Range(func(key, value any) bool {
		tempMap[key] = value
		return true
	})

	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(tempMap)

	if err != nil {
		return errors.Wrap(err, "")
	}

	file.Sync()
	return nil
}

func LoadFromDisk(filename string, m *sync.Map) error {
	err := os.MkdirAll(path.Dir(filename), 0755)
	if err != nil {
		return errors.Wrap(err, "")
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer file.Close()

	tempMap := make(map[any]any)
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&tempMap)
	if err != nil {
		if strings.Contains(err.Error(), "EOF") {
			return nil
		}
		return errors.Wrap(err, "")
	}

	for key, value := range tempMap {
		m.Store(key, value)
	}
	return nil
}
