// SPDX-License-Identifier: MIT

package common

import "io"

// CountingReader wraps a reader and counts bytes read.
type CountingReader struct {
	R     io.Reader
	Count int64
}

// NewCountingReader creates a new CountingReader.
func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{R: r}
}

// Read implements io.Reader.
func (cr *CountingReader) Read(p []byte) (int, error) {
	n, err := cr.R.Read(p)
	cr.Count += int64(n)
	return n, err
}
