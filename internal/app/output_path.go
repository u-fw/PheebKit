package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"pheebkit/internal/localstore"
	"strings"
)

func (s *Service) RememberOutputDirectory(output string) error {
	if !filepath.IsAbs(output) || strings.ContainsRune(output, 0) {
		return errors.New("输出必须是本地绝对路径")
	}
	directory := filepath.Dir(output)
	info, err := os.Stat(directory)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("输出文件夹不存在")
	}
	path, err := workspaceFile("preferences.json")
	if err != nil {
		return err
	}
	data := WorkspacePreferences{Version: 1}
	return localstore.Update(path, &data, func() error {
		if data.Version != 1 {
			return errors.New("常用设置版本不受支持")
		}
		data.LastOutputDirectory = directory
		return nil
	})
}

// A suggestion is not a reservation. Compile and final publication still refuse overwrite.
func (s *Service) SuggestOutputPath(output string) (string, error) {
	if !filepath.IsAbs(output) || strings.ContainsRune(output, 0) {
		return "", errors.New("输出必须是本地绝对路径")
	}
	info, err := os.Stat(filepath.Dir(output))
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("输出文件夹不存在")
	}
	busy := map[string]bool{}
	for _, job := range s.jobs.List() {
		if activeJob(job.State) {
			busy[strings.ToLower(filepath.Clean(job.Output.URI))] = true
		}
	}
	ext := filepath.Ext(output)
	stem := strings.TrimSuffix(output, ext)
	for n := 1; n <= 10000; n++ {
		candidate := output
		if n > 1 {
			candidate = fmt.Sprintf("%s-%d%s", stem, n, ext)
		}
		if busy[strings.ToLower(filepath.Clean(candidate))] {
			continue
		}
		_, err := os.Lstat(candidate)
		if os.IsNotExist(err) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("重名文件过多，请修改文件名")
}
