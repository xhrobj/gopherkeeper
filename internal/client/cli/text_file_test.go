package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestReadRequiredTextFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	want := "Alice's private note"
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatalf("write text file: %v", err)
	}

	got, err := readRequiredTextFile(path)
	if err != nil {
		t.Fatalf("readRequiredTextFile() error = %v", err)
	}
	if got != want {
		t.Errorf("readRequiredTextFile() = %q, want %q", got, want)
	}
}

func TestReadOptionalTextFile_EmptyPath(t *testing.T) {
	got, err := readOptionalTextFile("")
	if err != nil {
		t.Fatalf("readOptionalTextFile() error = %v", err)
	}
	if got != "" {
		t.Errorf("readOptionalTextFile() = %q, want empty string", got)
	}
}

func TestReadLimitedTextFile_MaximumSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.txt")
	want := strings.Repeat("a", model.MetadataMaxSize)
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatalf("write maximum metadata file: %v", err)
	}

	got, err := readLimitedTextFile(path, "metadata file", model.MetadataMaxSize)
	if err != nil {
		t.Fatalf("readLimitedTextFile() error = %v", err)
	}
	if got != want {
		t.Fatal("readLimitedTextFile() changed maximum-size data")
	}
}

func TestReadLimitedTextFile_RejectsTooLargeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", model.MetadataMaxSize+1)), 0o600); err != nil {
		t.Fatalf("write large metadata file: %v", err)
	}

	_, err := readLimitedTextFile(path, "metadata file", model.MetadataMaxSize)
	if !errors.Is(err, model.ErrPayloadTooLarge) {
		t.Fatalf("readLimitedTextFile() error = %v, want ErrPayloadTooLarge", err)
	}
}

func TestReadLimitedTextFile_RejectsDataThatGrewAfterStat(t *testing.T) {
	file := &textInputFileStub{
		Reader: strings.NewReader("12345"),
		info:   textFileInfoStub{size: 4},
	}

	_, err := readLimitedTextFileWith(
		"note.txt",
		"text file",
		4,
		func(string) (textInputFile, error) { return file, nil },
	)
	if !errors.Is(err, model.ErrPayloadTooLarge) {
		t.Fatalf("readLimitedTextFileWith() error = %v, want ErrPayloadTooLarge", err)
	}
	if !file.closed {
		t.Error("readLimitedTextFileWith() did not close the file")
	}
}

func TestReadLimitedTextFile_RejectsDirectory(t *testing.T) {
	_, err := readLimitedTextFile(t.TempDir(), "text file", model.TextPayloadMaxSize)
	if err == nil || err.Error() != "text file is a directory" {
		t.Fatalf("readLimitedTextFile() error = %v, want directory error", err)
	}
}

func TestReadLimitedTextFile_ReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	file := &textInputFileStub{
		Reader:   strings.NewReader("note"),
		info:     textFileInfoStub{size: 4},
		closeErr: closeErr,
	}

	_, err := readLimitedTextFileWith(
		"note.txt",
		"text file",
		4,
		func(string) (textInputFile, error) { return file, nil },
	)
	if !errors.Is(err, closeErr) {
		t.Fatalf("readLimitedTextFileWith() error = %v, want close error", err)
	}
}

type textInputFileStub struct {
	io.Reader
	info     os.FileInfo
	statErr  error
	closeErr error
	closed   bool
}

func (f *textInputFileStub) Stat() (os.FileInfo, error) {
	return f.info, f.statErr
}

func (f *textInputFileStub) Close() error {
	f.closed = true
	return f.closeErr
}

type textFileInfoStub struct {
	size int64
}

func (f textFileInfoStub) Name() string       { return "note.txt" }
func (f textFileInfoStub) Size() int64        { return f.size }
func (f textFileInfoStub) Mode() os.FileMode  { return 0 }
func (f textFileInfoStub) ModTime() time.Time { return time.Time{} }
func (f textFileInfoStub) IsDir() bool        { return false }
func (f textFileInfoStub) Sys() any           { return nil }
