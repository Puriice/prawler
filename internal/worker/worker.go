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

type Task func()

type Job struct {
	JobID jobID
	Task  Task
}

type WorkerManager struct {
	ctx     context.Context
	workers map[workerID]chan Job
	planner *planner.Planner[jobID, workerID]

	mu sync.Mutex

	currentJobId jobID
	workingJob   map[workerID]jobID
}

func NewManager(ctx context.Context, workerCount int, queueSizePerWorker int) *WorkerManager {
	manager := &WorkerManager{
		ctx:     ctx,
		workers: make(map[workerID]chan Job, workerCount),
		planner: planner.NewPlanner[jobID, workerID](),

		currentJobId: jobID(1),
		workingJob:   make(map[workerID]jobID, workerCount),
	}

	for i := range workerCount {
		manager.workers[workerID(i)] = make(chan Job, queueSizePerWorker)
	}

	return manager
}

func (m *WorkerManager) confirmJobDone(jobId jobID) {
	m.planner.Done(jobId)
}

func (m *WorkerManager) SpawnWorker() {
	for workerId, queue := range m.workers {
		go m.spawnWorker(workerId, queue)

		m.planner.AddResource(workerId)
	}
}

func (m *WorkerManager) spawnWorker(workerId workerID, queue chan Job) {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		m.runWorkerOnce(workerId, queue) // recovers internally
	}
}

func (m *WorkerManager) runWorkerOnce(workerId workerID, queue chan Job) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WORKER ID: %d] panic: %v\n", workerId, r)
			m.mu.Lock()
			jobId := m.workingJob[workerId]
			m.mu.Unlock()
			m.confirmJobDone(jobId)
		}
	}()

	for {
		select {
		case j := <-queue:
			jobID := j.JobID

			m.mu.Lock()
			m.workingJob[workerId] = jobID
			m.mu.Unlock()

			log.Printf("[WORKER ID: %d] working on: %d\n", workerId, jobID)
			j.Task()

			m.confirmJobDone(jobID)
			log.Printf("[WORKER ID: %d] finished working on: %d\n", workerId, jobID)
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *WorkerManager) Assign(task Task) error {
	m.mu.Lock()
	jobID := m.currentJobId
	m.currentJobId++
	m.mu.Unlock()

	workerID, ok := m.planner.Plan(jobID)

	if !ok {
		return ErrNoAvaliableWorker
	}

	return m.enqueue(jobID, workerID, task)
}

func (m *WorkerManager) AssignTo(workerID workerID, task Task) error {
	m.mu.Lock()
	jobID := m.currentJobId
	m.currentJobId++

	_, exist := m.workers[workerID]
	m.mu.Unlock()

	if !exist {
		return ErrWorkerNotExist
	}

	m.planner.Assign(jobID, workerID)

	return m.enqueue(jobID, workerID, task)
}

func (m *WorkerManager) enqueue(jobID jobID, workerID workerID, task Task) error {
	e := Job{
		JobID: jobID,
		Task:  task,
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
