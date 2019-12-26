// Package countingreader implements an io.Reader that counts the number of
// bytes read.
package countingreader

import (
	"io"
)

// Reader wraps an io.Reader, counting the total number of bytes read. It will
// wrap around after reading 16 exbibytes, which is assumed to be sufficient.
type Reader struct {
	reader    io.Reader
	ReadBytes uint64
}

func New(r io.Reader) *Reader {
	return &Reader{
		reader: r,
	}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.ReadBytes += uint64(n)
	return n, err
}
