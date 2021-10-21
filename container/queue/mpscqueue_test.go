package queue_test

import (
	"github.com/hsgames/gold/container/queue"
	"sync"
	"testing"
)

func TestMPSCQueue(t *testing.T) {
	var wg sync.WaitGroup
	var wg2 sync.WaitGroup
	q := queue.NewMPSCQueue(0, 1024)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var exit bool
		for {
			datas := q.Pop()
			for _, v := range *datas {
				if v == nil {
					exit = true
					break
				}
				t.Log(v.(int))
			}
			if exit {
				break
			}
		}
	}()
	for j := 0; j < 1; j++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			for i := 0; i < 100; i++ {
				q.Push(i)
			}
		}()
	}
	wg2.Wait()
	q.Push(nil)
	wg.Wait()
}
