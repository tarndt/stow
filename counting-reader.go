package stow

import (
	"io"
)

type CountingReader struct {
	rdr   io.Reader
	count int
}

var _ io.Reader = (*CountingReader)(nil)

func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{rdr: r}
}

func (cr *CountingReader) Read(buf []byte) (count int, err error) {
	count, err = cr.rdr.Read(buf)
	cr.count += count
	return count, err
}

func (cr *CountingReader) Bytes() int64 {
	return int64(cr.count)
}
