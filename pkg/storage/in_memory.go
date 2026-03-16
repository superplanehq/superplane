package storage

import (
	"bytes"
	"errors"
	"io"
)

type InMemoryStorage struct {
	files map[string][]byte
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		files: make(map[string][]byte),
	}
}

func (s *InMemoryStorage) Write(path string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	s.files[path] = data
	return nil
}

func (s *InMemoryStorage) Read(path string) (io.Reader, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, errors.New("file not found")
	}

	return bytes.NewReader(data), nil
}
