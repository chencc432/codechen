package main

import (
	"fmt"
	"sync"
)

type Count struct {
	count int
	mu    sync.Mutex
}

func (c *Count) increase() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

func (c *Count) getCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

func main() {
	count := &Count{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count.increase()
		}()
	}
	wg.Wait()
	fmt.Println(count.getCount())
}
