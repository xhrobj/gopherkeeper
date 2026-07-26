package binaryfile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.bin")
	want := []byte{0x00, 0x01, 0x02, 0xff}
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	filename, data, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if filename != "backup.bin" {
		t.Fatalf("filename = %q, want backup.bin", filename)
	}
	if !bytes.Equal(data, want) {
		t.Fatalf("data = %v, want %v", data, want)
	}
}

func TestRead_AcceptsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, data, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if data == nil || len(data) != 0 {
		t.Fatalf("data = %#v, want non-nil empty slice", data)
	}
}

func TestRead_RejectsDirectoryAndOversizedFile(t *testing.T) {
	t.Run("directory", func(t *testing.T) {
		_, _, err := Read(t.TempDir())
		if err == nil || err.Error() != "binary file is a directory" {
			t.Fatalf("Read() error = %v", err)
		}
	})

	t.Run("oversized file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "large.bin")
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("create fixture: %v", err)
		}
		if err := file.Truncate(int64(model.BinaryPayloadMaxSize + 1)); err != nil {
			_ = file.Close()
			t.Fatalf("truncate fixture: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close fixture: %v", err)
		}

		_, _, err = Read(path)
		if !errors.Is(err, model.ErrPayloadTooLarge) {
			t.Fatalf("Read() error = %v, want ErrPayloadTooLarge", err)
		}
	})
}
