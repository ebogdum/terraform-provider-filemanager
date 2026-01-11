// SPDX-License-Identifier: MIT

//go:build !unix

package local

import (
	"context"
	"io"
	"os"
)

// SupportsSendfile returns true if sendfile is supported.
func (z *ZeroCopy) SupportsSendfile() bool {
	return false
}

// SupportsSplice returns true if splice is supported.
func (z *ZeroCopy) SupportsSplice() bool {
	return false
}

// SupportsCopyFileRange returns true if copy_file_range is supported.
func (z *ZeroCopy) SupportsCopyFileRange() bool {
	return false
}

// zeroCopyTransfer attempts zero-copy transfer between two files.
// On non-Unix systems, this falls back to buffered copy.
func (z *ZeroCopy) zeroCopyTransfer(ctx context.Context, src, dst *os.File) (int64, error) {
	// No zero-copy support on this platform
	return 0, io.ErrNoProgress
}

// copyReaderPlatform attempts platform-specific copy from reader to file.
func (z *ZeroCopy) copyReaderPlatform(ctx context.Context, r io.Reader, dst *os.File) (int64, error) {
	// No platform optimization available
	return 0, io.ErrNoProgress
}

// transferToWriterPlatform attempts platform-specific transfer from file to writer.
func (z *ZeroCopy) transferToWriterPlatform(ctx context.Context, src *os.File, w io.Writer) (int64, error) {
	// No platform optimization available
	return 0, io.ErrNoProgress
}
