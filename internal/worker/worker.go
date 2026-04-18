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
type workID int

type Work func()

type event struct {
	WorkID workID
	Work   Work
}

type WorkerManager struct {
	ctx     context.Context
	workers map[workerID]chan event
	planner *planner.Planner[workID, workerID]

	mu sync.Mutex

	currentWorkId workID
}

func NewManager(ctx context.Context, workerCount int) *WorkerManager {
	manager := &WorkerManager{
		ctx:           ctx,
		workers:       make(map[workerID]chan event, workerCount),
		planner:       planner.NewPlanner[workID, workerID](),
		currentWorkId: workID(1),
	}

	for i := range workerCount {
		manager.workers[workerID(i)] = make(chan event, 1000)
	}

	return manager
}

func (m *WorkerManager) confirmWork(workId workID) {
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
					e.Work()
					m.confirmWork(e.WorkID)
				}
			}
		}(workerId, queue)

		m.planner.AddResource(workerId)
	}
}

func (m *WorkerManager) Assign(work Work) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	workID := m.currentWorkId
	m.currentWorkId++

	workerID, ok := m.planner.Plan(workID)

	if !ok {
		return ErrNoAvaliableWorker
	}

	e := event{
		WorkID: workID,
		Work:   work,
	}

	queue, ok := m.workers[workerID]

	if !ok {
		return ErrWorkerNotExist
	}

	select {
	case queue <- e:
	default:
		// queue full → drop or log
		log.Println("⚠️ Work dropped:", e.WorkID, e.Work)
		return ErrMaximumWorkerCapacity
	}

	return nil
}

func (m *WorkerManager) AssignTo(workerID workerID, work Work) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	workID := m.currentWorkId
	m.currentWorkId++

	e := event{
		WorkID: workID,
		Work:   work,
	}

	queue, ok := m.workers[workerID]

	if !ok {
		return ErrWorkerNotExist
	}

	m.planner.Assign(workID, workerID)

	select {
	case queue <- e:
	default:
		// queue full → drop or log
		log.Println("⚠️ Work dropped:", e.WorkID, e.Work)
		return ErrMaximumWorkerCapacity
	}

	return nil
}
