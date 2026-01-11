// SPDX-License-Identifier: MIT

package local

import (
	"context"
	"io"
	"os"
	"sync"
)

// ZeroCopy provides zero-copy file operations.
// Platform-specific implementations are in zerocopy_*.go files.
type ZeroCopy struct {
	// bufferPool is a pool of buffers for fallback copy operations
	bufferPool *sync.Pool
}

// NewZeroCopy creates a new ZeroCopy instance.
func NewZeroCopy() *ZeroCopy {
	return &ZeroCopy{
		bufferPool: &sync.Pool{
			New: func() any {
				// 32KB buffers for efficient copying
				buf := make([]byte, 32*1024)
				return &buf
			},
		},
	}
}

// getBuffer gets a buffer from the pool.
func (z *ZeroCopy) getBuffer() *[]byte {
	return z.bufferPool.Get().(*[]byte)
}

// putBuffer returns a buffer to the pool.
func (z *ZeroCopy) putBuffer(buf *[]byte) {
	z.bufferPool.Put(buf)
}

// CopyReader copies data from a reader to a file using optimal methods.
func (z *ZeroCopy) CopyReader(ctx context.Context, r io.Reader, dst *os.File) (int64, error) {
	// Try platform-specific optimizations first
	if written, err := z.copyReaderPlatform(ctx, r, dst); err == nil {
		return written, nil
	}

	// Fall back to buffered copy
	return z.bufferedCopy(ctx, r, dst)
}

// bufferedCopy performs a standard buffered copy using the buffer pool.
func (z *ZeroCopy) bufferedCopy(ctx context.Context, r io.Reader, w io.Writer) (int64, error) {
	buf := z.getBuffer()
	defer z.putBuffer(buf)

	var written int64
	for {
		// Check context
		if err := ctx.Err(); err != nil {
			return written, err
		}

		nr, err := r.Read(*buf)
		if nr > 0 {
			nw, ew := w.Write((*buf)[:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = io.ErrShortWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
	}
}

// TransferToWriter transfers data from a file to any writer.
func (z *ZeroCopy) TransferToWriter(ctx context.Context, src *os.File, w io.Writer) (int64, error) {
	// Try platform-specific optimizations first
	if written, err := z.transferToWriterPlatform(ctx, src, w); err == nil {
		return written, nil
	}

	// Fall back to buffered copy
	return z.bufferedCopy(ctx, src, w)
}

// TransferToFile transfers data from one file to another using optimal methods.
func (z *ZeroCopy) TransferToFile(ctx context.Context, src, dst *os.File) (int64, error) {
	// Try platform-specific zero-copy first
	if written, err := z.zeroCopyTransfer(ctx, src, dst); err == nil {
		return written, nil
	}

	// Fall back to buffered copy
	return z.bufferedCopy(ctx, src, dst)
}

// CopyFile copies a file using the most efficient method.
func (z *ZeroCopy) CopyFile(ctx context.Context, srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = z.TransferToFile(ctx, src, dst)
	if err != nil {
		os.Remove(dstPath)
		return err
	}

	return dst.Sync()
}
