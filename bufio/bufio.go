package bufio

import (
	"bufio"
	"github.com/pkg/errors"
)

func PopReader(b *bufio.Reader, n int) ([]byte, error) {
	buf, err := b.Peek(n)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	_, err = b.Discard(n)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buf, nil
}
