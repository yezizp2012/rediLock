package rediLock

import (
	"sync"
	"testing"
	"time"
)

func TestMutex(t *testing.T) {
	rs := NewRediSync(newRedisPool("redis://127.0.0.1:6379"))
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mutex := rs.NewMutex("test_mutex", SetExpire(time.Second))
			mutex.Lock()
			defer mutex.UnLock()

			t.Logf("[%d]test_mutex locked.", idx)
			time.Sleep(time.Millisecond * 100)
		}(i)
	}
	wg.Wait()
}
