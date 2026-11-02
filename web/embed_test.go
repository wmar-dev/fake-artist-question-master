package web

import (
	"io/fs"
	"testing"
)

const maxPageBytes = 100 << 10

func TestPageFilesAreWithinBudget(t *testing.T) {
	var total int64
	err := fs.WalkDir(FS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("page files: %d bytes", total)
	if total == 0 || total > maxPageBytes {
		t.Errorf("page files are %d bytes, want 1 to %d", total, maxPageBytes)
	}
}
