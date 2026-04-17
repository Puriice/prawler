package worker

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/purrice/prawler/internal/frontier/planner"
)

var (
	ErrNoAvaliableWorker     = errors.New("No worker avaliable.")
	ErrWorkerNotExist        = errors.New("The specify worker is not found.")
	ErrMaximumWorkerCapacity = errors.New("Worker had reached maximum capacity.")
)

type WorkerID int
type WorkID int

type Work func()

type Event struct {
	WorkID WorkID
	Work   Work
}

type WorkerManager struct {
	ctx     context.Context
	workers map[WorkerID]chan Event
	planner *planner.Planner[WorkID, WorkerID]

	mu sync.Mutex

	currentWorkId WorkID
}

func NewManager(ctx context.Context, workerCount int) *WorkerManager {
	manager := &WorkerManager{
		ctx:           ctx,
		workers:       make(map[WorkerID]chan Event, workerCount),
		planner:       planner.NewPlanner[WorkID, WorkerID](),
		currentWorkId: WorkID(1),
	}

	for i := range workerCount {
		manager.workers[WorkerID(i)] = make(chan Event, 1000)
	}

	return manager
}

func (m *WorkerManager) confirmWork(workId WorkID) {
	m.planner.Done(workId)
}

func (m *WorkerManager) SpawnWorker() {
	for workerId, event := range m.workers {
		go func(workerId WorkerID, event chan Event) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[WORKER ID: %d] panic: %v\n", workerId, r)
				}
			}()

			select {
			case <-m.ctx.Done():
			case e := <-event:
				e.Work()
				m.confirmWork(e.WorkID)
			}
		}(workerId, event)

		m.planner.AddResource(workerId)
	}
}

func (m *WorkerManager) Assign(work Work) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	workID := m.currentWorkId
	m.currentWorkId++

	event := Event{
		WorkID: workID,
		Work:   work,
	}

	workerID, ok := m.planner.Plan(workID)

	if !ok {
		return ErrNoAvaliableWorker
	}

	e, ok := m.workers[workerID]

	if !ok {
		return ErrWorkerNotExist
	}

	select {
	case e <- event:
	default:
		// queue full → drop or log
		log.Println("⚠️ Work dropped:", event.WorkID, event.Work)
		return ErrMaximumWorkerCapacity
	}

	return nil
}
