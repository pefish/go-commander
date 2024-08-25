package persistence

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"sync"

	"github.com/pkg/errors"
)

type PersistenceType struct {
	data    []byte
	cacheFs *os.File
	lock    sync.Mutex
}

func NewPersistenceType(filename string) (*PersistenceType, error) {
	p := &PersistenceType{}

	err := os.MkdirAll(path.Dir(filename), 0755)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	p.cacheFs = f
	b, err := io.ReadAll(p.cacheFs)
	if err != nil {
		return nil, err
	}
	if len(b) != 0 {
		p.lock.Lock()
		defer p.lock.Unlock()
		p.data = b
	}
	return p, nil
}

func (c *PersistenceType) Close() error {
	return c.cacheFs.Close()
}

func (c *PersistenceType) Save(data interface{}) error {
	if c.cacheFs == nil {
		return errors.New("Cache must be init first.")
	}
	err := c.cacheFs.Truncate(0)
	if err != nil {
		return err
	}
	result, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.data = result
	_, err = c.cacheFs.WriteAt(c.data, 0)
	if err != nil {
		return err
	}
	err = c.cacheFs.Sync()
	if err != nil {
		return err
	}
	return nil
}

func (c *PersistenceType) Load(out interface{}) (notFound bool, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.data == nil { // 代表没有数据
		return true, nil
	}
	err = json.Unmarshal(c.data, out)
	if err != nil {
		return false, err
	}
	return false, nil
}
