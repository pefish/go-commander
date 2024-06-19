package commander

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
)

type Cache struct {
	data    []byte
	cacheFs *os.File
	lock    sync.Mutex
}

func (c *Cache) Save(data interface{}) error {
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
	err = c.cacheFs.Close()
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Init(filename string) error {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	c.cacheFs = f
	b, err := io.ReadAll(c.cacheFs)
	if err != nil {
		return err
	}
	if len(b) != 0 {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.data = b
	}
	return nil
}

func (c *Cache) Load(out interface{}) (notFound bool, err error) {
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
