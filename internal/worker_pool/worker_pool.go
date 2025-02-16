package workerpool

import (
	"sync"
)

type Worker[T, U any] interface {
	DoWork(jobs chan T, results chan<- U) error
}

type WorkerPool[T, U any] struct {
	workers []Worker[T, U]
	jobs    chan T
	results chan<- U
	wg      *sync.WaitGroup
}

func New[T, U any](workers []Worker[T, U], jobs chan T, results chan<- U) WorkerPool[T, U] {
	return WorkerPool[T, U]{
		workers: workers,
		jobs:    jobs,
		results: results,
	}
}

func (wp WorkerPool[T, U]) DoWork() {
	wp.wg.Add(len(wp.workers))

	for _, worker := range wp.workers {
		go func(w Worker[T, U]) {
			defer wp.wg.Done()
			err := w.DoWork(wp.jobs, wp.results)
			if err != nil {
				return
			}
		}(worker)
	}

	wp.wg.Wait()
	close(wp.results)
}
