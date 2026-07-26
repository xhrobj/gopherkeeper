package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestReadBinaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.bin")
	want := []byte{0x00, 0x01, 0x02, 0xff}
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write binary file: %v", err)
	}

	filename, data, err := readBinaryFile(path)
	if err != nil {
		t.Fatalf("readBinaryFile() error = %v", err)
	}
	if filename != "backup.bin" {
		t.Errorf("filename = %q, want backup.bin", filename)
	}
	if !bytes.Equal(data, want) {
		t.Errorf("data = %v, want %v", data, want)
	}
}

func TestReadBinaryFile_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	_, data, err := readBinaryFile(path)
	if err != nil {
		t.Fatalf("readBinaryFile() error = %v", err)
	}
	if data == nil || len(data) != 0 {
		t.Fatalf("data = %#v, want non-nil empty slice", data)
	}
}

func TestReadBinaryFile_MaximumSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "maximum.bin")
	want := bytes.Repeat([]byte{0x2a}, model.BinaryPayloadMaxSize)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write maximum file: %v", err)
	}

	_, data, err := readBinaryFile(path)
	if err != nil {
		t.Fatalf("readBinaryFile() error = %v", err)
	}
	if !bytes.Equal(data, want) {
		t.Fatal("readBinaryFile() changed maximum-size data")
	}
}

func TestReadBinaryFile_RejectsTooLargeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.bin")
	if err := os.WriteFile(path, bytes.Repeat([]byte{0x2a}, model.BinaryPayloadMaxSize+1), 0o600); err != nil {
		t.Fatalf("write large file: %v", err)
	}

	_, _, err := readBinaryFile(path)
	if !errors.Is(err, model.ErrPayloadTooLarge) {
		t.Fatalf("readBinaryFile() error = %v, want ErrPayloadTooLarge", err)
	}
}

func TestReadBinaryFile_RejectsDirectory(t *testing.T) {
	_, _, err := readBinaryFile(t.TempDir())
	if err == nil || err.Error() != "binary file is a directory" {
		t.Fatalf("readBinaryFile() error = %v, want directory error", err)
	}
}

func TestWriteBinaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restored-backup.bin")
	want := []byte{0x00, 0x01, 0x02, 0xff}

	if err := writeBinaryFile(path, want); err != nil {
		t.Fatalf("writeBinaryFile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output data = %v, want %v", got, want)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat output file: %v", err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("output mode = %o, want 600", mode)
		}
	}
}

func TestWriteBinaryFile_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	if err := writeBinaryFile(path, []byte{}); err != nil {
		t.Fatalf("writeBinaryFile() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat output file: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("output size = %d, want 0", info.Size())
	}
}

func TestWriteBinaryFile_RejectsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.bin")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	err := writeBinaryFile(path, []byte("replacement"))
	if err == nil || !strings.Contains(err.Error(), "create output file") {
		t.Fatalf("writeBinaryFile() error = %v, want create error", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read existing file: %v", readErr)
	}
	if string(got) != "original" {
		t.Errorf("existing content = %q, want original", got)
	}
}

func TestWriteBinaryFile_RejectsMissingParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "backup.bin")
	err := writeBinaryFile(path, []byte("backup"))
	if err == nil || !strings.Contains(err.Error(), "create output file") {
		t.Fatalf("writeBinaryFile() error = %v, want create error", err)
	}
}
