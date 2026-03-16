package storage

import "io"

type Storage interface {
	Write(path string, reader io.Reader) error
	Read(path string) (io.Reader, error)
}
