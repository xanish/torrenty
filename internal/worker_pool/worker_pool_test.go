package workerpool

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockWorker struct {
	mock.Mock
}

func (m *mockWorker) DoWork(jobs <-chan int, results chan<- string) error {
	m.Called(jobs, results)
	for job := range jobs {
		results <- "processed " + strconv.Itoa(job)
	}

	return nil
}

func TestWorkerPool(t *testing.T) {
	t.Run("new worker pool with valid parameters", func(t *testing.T) {
		mockWorkers := []Worker[int, string]{&mockWorker{}}
		jobs := make(<-chan int)
		results := make(chan<- string)

		wp := New(mockWorkers, jobs, results)

		assert.Equal(t, mockWorkers, wp.workers)
		assert.Equal(t, jobs, wp.jobs)
		assert.Equal(t, results, wp.results)
	})

	t.Run("start worker pool with a single worker", func(t *testing.T) {
		jobs := make(chan int)
		results := make(chan string, 2)
		worker := new(mockWorker)
		worker.On("DoWork", mock.AnythingOfType("<-chan int"), mock.AnythingOfType("chan<- string")).Return().Once()

		wp := New([]Worker[int, string]{worker}, jobs, results)
		go wp.Start()

		jobs <- 1
		jobs <- 2
		close(jobs)

		worker.AssertExpectations(t)
		assert.Equal(t, "processed 1", <-results)
		assert.Equal(t, "processed 2", <-results)
	})

	t.Run("start worker pool with multiple workers", func(t *testing.T) {
		jobs := make(chan int)
		results := make(chan string, 3)

		worker1 := new(mockWorker)
		worker2 := new(mockWorker)
		worker1.On("DoWork", mock.AnythingOfType("<-chan int"), mock.AnythingOfType("chan<- string")).Return().Once()
		worker2.On("DoWork", mock.AnythingOfType("<-chan int"), mock.AnythingOfType("chan<- string")).Return().Once()

		wp := New([]Worker[int, string]{worker1, worker2}, jobs, results)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			wp.Start()
			wg.Done()
		}()

		jobs <- 1
		jobs <- 2
		jobs <- 3
		close(jobs)
		wg.Wait()

		worker1.AssertExpectations(t)
		worker2.AssertExpectations(t)

		assert.Equal(t, "processed 1", <-results)
		assert.Equal(t, "processed 2", <-results)
		assert.Equal(t, "processed 3", <-results)
	})

	t.Run("start worker pool with no jobs", func(t *testing.T) {
		jobs := make(chan int)
		results := make(chan string)
		worker := new(mockWorker)
		worker.On("DoWork", mock.AnythingOfType("<-chan int"), mock.AnythingOfType("chan<- string")).Return().Once()

		wp := New([]Worker[int, string]{worker}, jobs, results)

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			wp.Start()
			wg.Done()
		}()

		close(jobs)
		wg.Wait()

		select {
		case result, ok := <-results:
			assert.False(t, ok, "Results channel should be closed")
			assert.Empty(t, result, "No result should be sent")
		default:
			assert.Fail(t, "Results channel should be closed after processing")
		}
	})
}
