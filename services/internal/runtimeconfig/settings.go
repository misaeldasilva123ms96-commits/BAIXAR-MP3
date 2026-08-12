package runtimeconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
)

type FileSettingsStore struct {
	mu    sync.RWMutex
	path  string
	value core.Settings
}

func NewFileSettingsStore(path, downloadDirectory string) (*FileSettingsStore, error) {
	store := &FileSettingsStore{path: path, value: core.Settings{DefaultQuality: core.QualityVBR0, DownloadDirectory: downloadDirectory, OrganizePlaylist: true, AvoidDuplicates: true, EmbedThumbnail: true, EmbedMetadata: true, OpenFolderWhenDone: true}}
	data, err := os.ReadFile(path)
	if err == nil {
		var loaded core.Settings
		if json.Unmarshal(data, &loaded) == nil && loaded.Validate() == nil {
			store.value = loaded
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}
func (s *FileSettingsStore) Get() core.Settings { s.mu.RLock(); defer s.mu.RUnlock(); return s.value }
func (s *FileSettingsStore) Save(value core.Settings) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err = os.WriteFile(temp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err = os.Rename(temp, s.path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()
	return nil
}
