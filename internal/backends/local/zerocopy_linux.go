// SPDX-License-Identifier: MIT

//go:build linux

package local

import (
	"context"
	"io"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// SupportsSendfile returns true if sendfile is supported.
func (z *ZeroCopy) SupportsSendfile() bool {
	return true
}

// SupportsSplice returns true if splice is supported.
func (z *ZeroCopy) SupportsSplice() bool {
	return true
}

// SupportsCopyFileRange returns true if copy_file_range is supported.
func (z *ZeroCopy) SupportsCopyFileRange() bool {
	// copy_file_range was added in Linux 4.5
	return true
}

// zeroCopyTransfer attempts zero-copy transfer between two files.
func (z *ZeroCopy) zeroCopyTransfer(ctx context.Context, src, dst *os.File) (int64, error) {
	srcInfo, err := src.Stat()
	if err != nil {
		return 0, err
	}
	size := srcInfo.Size()

	// Try copy_file_range first (Linux 4.5+)
	if z.SupportsCopyFileRange() {
		written, err := z.copyFileRange(ctx, src, dst, size)
		if err == nil {
			return written, nil
		}
		// Fall through to sendfile on failure
	}

	// Try sendfile
	return z.sendFile(ctx, src, dst, size)
}

// copyFileRange uses copy_file_range(2) for zero-copy file-to-file transfer.
func (z *ZeroCopy) copyFileRange(ctx context.Context, src, dst *os.File, size int64) (int64, error) {
	srcFd := int(src.Fd())
	dstFd := int(dst.Fd())

	var written int64
	var srcOff, dstOff int64

	for written < size {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		n, err := unix.CopyFileRange(srcFd, &srcOff, dstFd, &dstOff, int(size-written), 0)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.EXDEV || err == syscall.ENOSYS || err == syscall.EOPNOTSUPP {
				// Cross-device copy or not supported, fall back
				return written, err
			}
			return written, err
		}
		if n == 0 {
			break
		}
		written += int64(n)
	}

	return written, nil
}

// sendFile uses sendfile(2) for zero-copy transfer.
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

		n, err := unix.Sendfile(dstFd, srcFd, &offset, int(toTransfer))
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.ENOSYS || err == syscall.EINVAL {
				// sendfile not supported between these fds
				return written, err
			}
			return written, err
		}
		if n == 0 {
			break
		}
		written += int64(n)
	}

	return written, nil
}

// copyReaderPlatform attempts platform-specific copy from reader to file.
func (z *ZeroCopy) copyReaderPlatform(ctx context.Context, r io.Reader, dst *os.File) (int64, error) {
	// If the reader is a file, use zero-copy transfer
	if src, ok := r.(*os.File); ok {
		return z.zeroCopyTransfer(ctx, src, dst)
	}

	// If reader supports splice via pipes
	if pipeReader, ok := r.(interface{ Fd() uintptr }); ok {
		return z.spliceFromPipe(ctx, pipeReader.Fd(), dst)
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

	// If writer supports splice via pipes
	if pipeWriter, ok := w.(interface{ Fd() uintptr }); ok {
		return z.spliceToPipe(ctx, src, pipeWriter.Fd())
	}

	// No platform optimization available
	return 0, io.ErrNoProgress
}

// spliceFromPipe uses splice to transfer from a pipe to a file.
func (z *ZeroCopy) spliceFromPipe(ctx context.Context, pipeFd uintptr, dst *os.File) (int64, error) {
	dstFd := int(dst.Fd())
	srcFd := int(pipeFd)

	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		n, err := unix.Splice(srcFd, nil, dstFd, nil, 1<<30, unix.SPLICE_F_MOVE|unix.SPLICE_F_MORE)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.EAGAIN {
				continue
			}
			return written, err
		}
		if n == 0 {
			return written, nil
		}
		written += int64(n)
	}
}

// spliceToPipe uses splice to transfer from a file to a pipe.
func (z *ZeroCopy) spliceToPipe(ctx context.Context, src *os.File, pipeFd uintptr) (int64, error) {
	srcFd := int(src.Fd())
	dstFd := int(pipeFd)

	srcInfo, err := src.Stat()
	if err != nil {
		return 0, err
	}
	size := srcInfo.Size()

	var written int64
	var offset int64

	for written < size {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		n, err := unix.Splice(srcFd, &offset, dstFd, nil, int(size-written), unix.SPLICE_F_MOVE|unix.SPLICE_F_MORE)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.EAGAIN {
				continue
			}
			return written, err
		}
		if n == 0 {
			return written, nil
		}
		written += int64(n)
	}

	return written, nil
}
