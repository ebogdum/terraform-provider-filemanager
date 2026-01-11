// SPDX-License-Identifier: MIT

//go:build unix && !linux

package local

import (
	"context"
	"io"
	"os"
	"syscall"
)

// SupportsSendfile returns true if sendfile is supported.
func (z *ZeroCopy) SupportsSendfile() bool {
	return true // Darwin supports sendfile
}

// SupportsSplice returns true if splice is supported.
func (z *ZeroCopy) SupportsSplice() bool {
	return false // splice is Linux-only
}

// SupportsCopyFileRange returns true if copy_file_range is supported.
func (z *ZeroCopy) SupportsCopyFileRange() bool {
	return false // copy_file_range is Linux-only
}

// zeroCopyTransfer attempts zero-copy transfer between two files.
func (z *ZeroCopy) zeroCopyTransfer(ctx context.Context, src, dst *os.File) (int64, error) {
	srcInfo, err := src.Stat()
	if err != nil {
		return 0, err
	}
	size := srcInfo.Size()

	// Try sendfile
	return z.sendFile(ctx, src, dst, size)
}

// sendFile uses sendfile(2) for zero-copy transfer on Darwin.
func (z *ZeroCopy) sendFile(ctx context.Context, src, dst *os.File, size int64) (int64, error) {
	srcFd := int(src.Fd())
	dstFd := int(dst.Fd())

	var written int64
	var offset int64

	for written < size {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		// sendfile can transfer up to 2GB at a time
		toTransfer := size - written
		if toTransfer > 1<<30 { // 1GB chunks
			toTransfer = 1 << 30
		}

		// Darwin sendfile has different signature
		// int sendfile(int fd, int s, off_t offset, off_t *len, struct sf_hdtr *hdtr, int flags)
		var length int64 = toTransfer
		_, err := syscall.Sendfile(dstFd, srcFd, &offset, int(length))
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.ENOSYS || err == syscall.EINVAL || err == syscall.EAGAIN {
				// sendfile not supported between these fds, fall back
				return written, err
			}
			return written, err
		}
		written += length
	}

	return written, nil
}

// copyReaderPlatform attempts platform-specific copy from reader to file.
func (z *ZeroCopy) copyReaderPlatform(ctx context.Context, r io.Reader, dst *os.File) (int64, error) {
	// If the reader is a file, use zero-copy transfer
	if src, ok := r.(*os.File); ok {
		return z.zeroCopyTransfer(ctx, src, dst)
	}

	// No platform optimization available
	return 0, io.ErrNoProgress
}

// transferToWriterPlatform attempts platform-specific transfer from file to writer.
func (z *ZeroCopy) transferToWriterPlatform(ctx context.Context, src *os.File, w io.Writer) (int64, error) {
	// If writer is a file, use zero-copy transfer
	if dst, ok := w.(*os.File); ok {
		return z.zeroCopyTransfer(ctx, src, dst)
	}

	// No platform optimization available
	return 0, io.ErrNoProgress
}
