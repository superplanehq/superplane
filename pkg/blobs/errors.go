package blobs

import "errors"

var (
	ErrBlobNotFound             = errors.New("blob not found")
	ErrPresignedURLNotSupported = errors.New("presigned URLs are not supported by this backend")
	ErrInvalidScope             = errors.New("invalid blob scope")
	ErrInvalidBlobPath          = errors.New("invalid blob path")
)
