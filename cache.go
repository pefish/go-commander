package commander

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
)

type Cache struct {
	data []byte
	cacheFs *os.File
}

func (c *Cache) Save(data interface{}) error {
	if c.cacheFs == nil {
		return errors.New("cache must be init first")
	}
	err := c.cacheFs.Truncate(0)
	if err != nil {
		return err
	}
	result, err := json.Marshal(data)
	if err != nil {
		return err
	}
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
	b, err := ioutil.ReadAll(c.cacheFs)
	if err != nil {
		return err
	}
	if len(b) != 0 {
		c.data = b
	}
	return nil
}

func (c *Cache) Load(out interface{}) (notFound bool, err error) {
	if c.data == nil {  // 代表没有数据
		return true, nil
	}
	err = json.Unmarshal(c.data, out)
	if err != nil {
		return false, err
	}
	return false, nil
}
