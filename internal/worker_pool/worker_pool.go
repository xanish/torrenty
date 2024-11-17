package workerpool

import (
	"sync"
)

type Worker[T, U any] interface {
	DoWork(jobs <-chan T, results chan<- U)
}

type WorkerPool[T, U any] struct {
	workers []Worker[T, U]
	jobs    <-chan T
	results chan<- U
}

func New[T, U any](workers []Worker[T, U], jobs <-chan T, results chan<- U) WorkerPool[T, U] {
	return WorkerPool[T, U]{
		workers: workers,
		jobs:    jobs,
		results: results,
	}
}

func (wp WorkerPool[T, U]) Start() {
	wg := sync.WaitGroup{}
	wg.Add(len(wp.workers))

	for _, worker := range wp.workers {
		go func(w Worker[T, U]) {
			defer wg.Done()
			w.DoWork(wp.jobs, wp.results)
		}(worker)
	}

	wg.Wait()
	close(wp.results)
}
