package objstore

import (
	"io"
	"os"
	"path"
)

type ObjectReader interface {
	Open(key string) (io.ReadCloser, error)
}

type LocalFSObjectReader struct {
	basePath string
}

func NewLocalFSObjectReader(basePath string) *LocalFSObjectReader {
	return &LocalFSObjectReader{basePath: basePath}
}

func (r *LocalFSObjectReader) Open(key string) (io.ReadCloser, error) {
	return os.Open(path.Join(r.basePath, key))
}
