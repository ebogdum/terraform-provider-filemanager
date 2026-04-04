// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"io"
	"sync/atomic"
)

// CountingReader wraps a reader and counts bytes read.
type CountingReader struct {
	R     io.Reader
	count atomic.Int64
}

// NewCountingReader creates a new CountingReader.
func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{R: r}
}

// Read implements io.Reader.
func (cr *CountingReader) Read(p []byte) (int, error) {
	n, err := cr.R.Read(p)
	cr.count.Add(int64(n))
	return n, err
}

// Count returns the total number of bytes read.
func (cr *CountingReader) Count() int64 {
	return cr.count.Load()
}

// MaxReadSize is the default maximum size for ReadAll operations (256 MB).
const MaxReadSize = 256 * 1024 * 1024

// ReadAllLimited reads all data from r but returns an error if the data exceeds maxBytes.
// If maxBytes is 0, MaxReadSize is used.
func ReadAllLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	if 0 == maxBytes {
		maxBytes = MaxReadSize
	}
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if nil != err {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("data exceeds maximum size of %d bytes", maxBytes)
	}
	return data, nil
}
