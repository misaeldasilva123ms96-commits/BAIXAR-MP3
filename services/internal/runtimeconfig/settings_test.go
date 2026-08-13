package runtimeconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/misaeldasilva123ms96-commits/baixar-mp3/services/internal/core"
)

func TestConcurrentSettingsWritesRemainConsistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := NewFileSettingsStore(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var writers sync.WaitGroup
	for _, quality := range []core.Quality{core.Quality128, core.Quality192, core.Quality256, core.Quality320} {
		writers.Add(1)
		go func(value core.Quality) {
			defer writers.Done()
			settings := store.Get()
			settings.DefaultQuality = value
			if err := store.Save(settings); err != nil {
				t.Errorf("Save: %v", err)
			}
		}(quality)
	}
	writers.Wait()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted core.Settings
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != store.Get() {
		t.Fatalf("persisted settings diverged from memory: %#v != %#v", persisted, store.Get())
	}
}
