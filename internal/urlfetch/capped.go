package urlfetch

import (
	"fmt"
	"io"
)

// CappedReader wraps r and returns an error after Max bytes have been read (inclusive).
type CappedReader struct {
	R   io.Reader
	Max int64
	N   int64
}

func (c *CappedReader) Read(p []byte) (int, error) {
	if c.Max <= 0 {
		return 0, fmt.Errorf("urlfetch: Max must be positive")
	}
	if c.N >= c.Max {
		return 0, fmt.Errorf("response exceeds maximum size (%d bytes)", c.Max)
	}
	room := c.Max - c.N
	if int64(len(p)) > room {
		p = p[:room]
	}
	n, err := c.R.Read(p)
	c.N += int64(n)
	if c.N > c.Max {
		return n, fmt.Errorf("response exceeds maximum size (%d bytes)", c.Max)
	}
	return n, err
}
