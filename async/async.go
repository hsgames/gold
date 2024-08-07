package async

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

var (
	ErrServiceChanFull     = errors.New("async: service chan is full")
	ErrServiceAwaitTimeout = errors.New("async: service await timeout")
)

type response struct {
	res any
	err error
}

type Service struct {
	ctx     context.Context
	name    string
	ch      chan func()
	timeout time.Duration
}

func New(ctx context.Context, name string,
	size int, timeout time.Duration) *Service {

	return &Service{
		ctx:     ctx,
		name:    name,
		ch:      make(chan func(), size),
		timeout: timeout,
	}
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) Get() <-chan func() {
	return s.ch
}

func (s *Service) Await(f func() (any, error)) (res any, err error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	ch := make(chan response, 1)

	fn := func() {
		var (
			res1 any
			err1 error
		)

		defer func() {
			if err1 != nil {
				slog.Error(
					fmt.Sprintf("async: [%s] await func", s.name),
					slog.Any("error", err1),
				)

				res1 = nil
			}

			ch <- response{res1, err1}

			close(ch)
		}()

		defer func() {
			if r := recover(); r != nil {
				err1 = fmt.Errorf("async: [%s] await func panic [%v]", s.name, r)
			}
		}()

		res1, err1 = f()

		return
	}

	select {
	case s.ch <- fn:
		select {
		case resp := <-ch:
			if err = resp.err; err != nil {
				return
			}

			res = resp.res
		case <-ctx.Done():
			err = ErrServiceAwaitTimeout
		}
	default:
		err = ErrServiceChanFull
	}

	return
}

func (s *Service) Send(f func() (any, error)) (err error) {
	fn := func() {
		var err1 error

		defer func() {
			if err1 != nil {
				slog.Error(
					fmt.Sprintf("async: [%s] send func", s.name),
					slog.Any("error", err1),
				)
			}
		}()

		defer func() {
			if r := recover(); r != nil {
				err1 = fmt.Errorf("async: [%s] send func panic [%v]", s.name, r)
			}
		}()

		_, err1 = f()

		return
	}

	select {
	case s.ch <- fn:
	default:
		err = ErrServiceChanFull
	}

	return
}
