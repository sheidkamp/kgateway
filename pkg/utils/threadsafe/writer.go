package threadsafe

import (
	"io"
	"sync"
)

// threadSafeWriter wraps an io.Writer with mutex protection for concurrent writes
type ThreadSafeWriter struct {
	W  io.Writer
	mu sync.Mutex
}

func (tsw *ThreadSafeWriter) Write(p []byte) (n int, err error) {
	tsw.mu.Lock()
	defer tsw.mu.Unlock()
	return tsw.W.Write(p)
}
