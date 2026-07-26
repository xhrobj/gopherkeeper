package binaryfile

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.bin")
	want := []byte{0x00, 0x01, 0x02, 0xff}

	if err := Write(path, want); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("output data = %v, want %v", got, want)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat output file: %v", err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("output mode = %o, want 600", mode)
		}
	}
}

func TestWrite_RejectsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.bin")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	err := Write(path, []byte("replacement"))
	if err == nil || !strings.Contains(err.Error(), "create output file") {
		t.Fatalf("Write() error = %v, want create error", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read existing file: %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("existing content = %q, want original", got)
	}
}

func TestWrite_RemovesPartialFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.bin")
	writeErr := errors.New("disk full")

	err := writeWith(
		path,
		[]byte("backup"),
		func(outputPath string) (outputFile, error) {
			file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				return nil, err
			}
			return &failingOutputFile{file: file, err: writeErr}, nil
		},
		os.Remove,
	)
	if !errors.Is(err, writeErr) {
		t.Fatalf("writeWith() error = %v, want %v", err, writeErr)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("partial output stat error = %v, want not exist", statErr)
	}
}

type failingOutputFile struct {
	file *os.File
	err  error
}

func (file *failingOutputFile) Write(data []byte) (int, error) {
	written := len(data) / 2
	if written == 0 && len(data) > 0 {
		written = 1
	}
	if _, err := io.CopyN(file.file, bytes.NewReader(data), int64(written)); err != nil {
		return 0, err
	}
	return written, file.err
}

func (file *failingOutputFile) Close() error {
	return file.file.Close()
}
