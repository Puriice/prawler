package worker

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/purrice/prawler/internal/planner"
)

var (
	ErrNoAvaliableWorker     = errors.New("No worker avaliable.")
	ErrWorkerNotExist        = errors.New("The specify worker is not found.")
	ErrMaximumWorkerCapacity = errors.New("Worker had reached maximum capacity.")
)

type workerID int
type jobID int

type Job func()

type event struct {
	JobID jobID
	Job   Job
}

type WorkerManager struct {
	ctx     context.Context
	workers map[workerID]chan event
	planner *planner.Planner[jobID, workerID]

	mu sync.Mutex

	currentJobId jobID
}

func NewManager(ctx context.Context, workerCount int) *WorkerManager {
	manager := &WorkerManager{
		ctx:          ctx,
		workers:      make(map[workerID]chan event, workerCount),
		planner:      planner.NewPlanner[jobID, workerID](),
		currentJobId: jobID(1),
	}

	for i := range workerCount {
		manager.workers[workerID(i)] = make(chan event, 1000)
	}

	return manager
}

func (m *WorkerManager) confirmJobDone(workId jobID) {
	m.planner.Done(workId)
}

func (m *WorkerManager) SpawnWorker() {
	for workerId, queue := range m.workers {
		go func(workerId workerID, queue chan event) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[WORKER ID: %d] panic: %v\n", workerId, r)
				}
			}()

			for {
				select {
				case <-m.ctx.Done():
					return
				case e := <-queue:
					e.Job()
					m.confirmJobDone(e.JobID)
				}
			}
		}(workerId, queue)

		m.planner.AddResource(workerId)
	}
}

func (m *WorkerManager) Assign(job Job) error {
	m.mu.Lock()
	jobID := m.currentJobId
	m.currentJobId++
	m.mu.Unlock()

	workerID, ok := m.planner.Plan(jobID)

	if !ok {
		return ErrNoAvaliableWorker
	}

	return m.enqueue(jobID, workerID, job)
}

func (m *WorkerManager) AssignTo(workerID workerID, job Job) error {
	m.mu.Lock()
	jobID := m.currentJobId
	m.currentJobId++

	_, exist := m.workers[workerID]
	m.mu.Unlock()

	if !exist {
		return ErrWorkerNotExist
	}

	m.planner.Assign(jobID, workerID)

	return m.enqueue(jobID, workerID, job)
}

func (m *WorkerManager) enqueue(jobID jobID, workerID workerID, job Job) error {
	e := event{
		JobID: jobID,
		Job:   job,
	}

	queue, ok := m.workers[workerID]

	if !ok {
		return ErrWorkerNotExist
	}

	select {
	case queue <- e:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}
