// Package localstore provides bounded atomic JSON updates with process locking.
package localstore

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func Update(path string, value any, update func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	releaseLocal, err := lockPath(path)
	if err != nil {
		return err
	}
	defer releaseLocal()
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	release, err := lockFile(lock)
	if err != nil {
		return err
	}
	defer release()
	if err := Read(path, value); err != nil {
		return err
	}
	if update == nil {
		return nil
	}
	if err := update(); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > 24*1024*1024 {
		return errors.New("本地记录超过大小限制，未覆盖原文件")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".workspace-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func Read(path string, value any) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 24*1024*1024 {
		return errors.New("本地记录超过大小限制，请备份后处理")
	}
	decoder := json.NewDecoder(io.LimitReader(file, 24*1024*1024))
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("本地记录格式无效")
	}
	return nil
}
