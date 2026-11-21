package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotatingLoggerCompressesAndCapsBackups(t *testing.T) {
	dir := t.TempDir()
	writer := newRotatingLogger(filepath.Join(dir, "webnote.log"), 1, 7)
	payload := bytes.Repeat([]byte("x"), 700*1024)
	for i := 0; i < 20; i++ {
		if _, err := writer.Write(payload); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		backups, err := filepath.Glob(filepath.Join(dir, "webnote-*.log.gz"))
		if err != nil {
			t.Fatal(err)
		}
		allValid := len(backups) > 0 && len(backups) <= 7
		for _, path := range backups {
			file, err := os.Open(path)
			if err != nil {
				if os.IsNotExist(err) {
					allValid = false
					break
				}
				t.Fatal(err)
			}
			reader, err := gzip.NewReader(file)
			if err == nil {
				_, err = io.Copy(io.Discard, reader)
				reader.Close()
			}
			file.Close()
			if err != nil {
				allValid = false // compression runs asynchronously after rotation.
				break
			}
		}
		if allValid {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("rotated gzip backups did not settle into valid files: %v", backups)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
