package project

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ffmpegstudio/internal/core"
)

const maxProjectSize = 1024 * 1024

func Load(path string) (core.Project, error) {
	file, err := os.Open(path)
	if err != nil {
		return core.Project{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return core.Project{}, err
	}
	if !stat.Mode().IsRegular() || stat.Size() > maxProjectSize {
		return core.Project{}, fmt.Errorf("project must be a regular file of at most 1 MiB")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxProjectSize+1))
	decoder.DisallowUnknownFields()
	var result core.Project
	if err := decoder.Decode(&result); err != nil {
		return core.Project{}, fmt.Errorf("read project: %w", err)
	}
	if result.SchemaVersion != core.ProjectSchemaVersion {
		return core.Project{}, fmt.Errorf("unsupported project schema version %d", result.SchemaVersion)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return core.Project{}, fmt.Errorf("project contains trailing JSON or exceeds size limit")
	}
	return result, nil
}

func Save(path string, project core.Project) error {
	if project.SchemaVersion != core.ProjectSchemaVersion {
		return fmt.Errorf("unsupported project schema version %d", project.SchemaVersion)
	}
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxProjectSize {
		return fmt.Errorf("project exceeds 1 MiB limit")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".ffproject-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
