// Package jobs owns a serial, cancellable job queue independent of the UI shell.
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"kompeg/internal/core"
)

const (
	queueLimit   = 32
	historyLimit = 200
	logByteLimit = 64 * 1024
	logLineLimit = 120
)

type record struct {
	job          core.Job
	plan         core.ExecutionPlan
	cancel       context.CancelFunc
	lastProgress time.Time
	lastLog      time.Time
	logBytes     int
}

type Manager struct {
	runner           core.Runner
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.Mutex
	records          map[string]*record
	order            []string
	queue            chan string
	events           chan core.Job
	done             chan struct{}
	eventDone        chan struct{}
	closed           bool
	closeOnce        sync.Once
	persist          func([]core.Job) error
	persistenceError string
	savedStates      map[string]string
}

// Restore only restores display records. Persisted commands are never executed.
func (m *Manager) Restore(history []core.Job, save func([]core.Job) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, job := range history {
		if terminal(job.State) && m.records[job.ID] == nil {
			m.records[job.ID] = &record{job: snapshot(job)}
			m.order = append(m.order, job.ID)
		}
	}
	m.persist = save
	m.savedStates = make(map[string]string)
	for _, job := range history {
		m.savedStates[job.ID] = job.State
	}
}

func (m *Manager) PersistenceError() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.persistenceError
}

func (m *Manager) EnqueueProject(plan core.ExecutionPlan, project core.Project) (core.Job, error) {
	return m.enqueue(plan, &project)
}

func New(runner core.Runner, emit func(core.Job)) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{runner: runner, ctx: ctx, cancel: cancel, records: make(map[string]*record),
		queue: make(chan string, queueLimit), events: make(chan core.Job, 64),
		done: make(chan struct{}), eventDone: make(chan struct{})}
	go func() {
		defer close(m.eventDone)
		for job := range m.events {
			if emit != nil {
				emit(job)
			}
		}
	}()
	go m.work()
	return m
}

func (m *Manager) Enqueue(plan core.ExecutionPlan) (core.Job, error) {
	return m.enqueue(plan, nil)
}

func (m *Manager) enqueue(plan core.ExecutionPlan, project *core.Project) (core.Job, error) {
	id := plan.JobID
	if id == "" {
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return core.Job{}, fmt.Errorf("generate job ID: %w", err)
		}
		id = hex.EncodeToString(bytes[:])
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return core.Job{}, fmt.Errorf("job manager is closed")
	}
	if _, exists := m.records[id]; exists {
		return core.Job{}, fmt.Errorf("duplicate job ID %s", id)
	}
	if len(m.queue) == cap(m.queue) {
		return core.Job{}, fmt.Errorf("queue is full (%d pending jobs)", queueLimit)
	}
	m.pruneHistory()
	if len(m.records) >= historyLimit {
		return core.Job{}, fmt.Errorf("job history is full")
	}
	plan.JobID = id
	plan.Args = append([]string(nil), plan.Args...)
	if plan.Trim != nil {
		copy := *plan.Trim
		plan.Trim = &copy
	}
	job := core.Job{ID: id, ProjectID: plan.ProjectID, Input: plan.Input, Output: plan.Output,
		State: core.JobQueued, QueuedAt: time.Now(), Logs: []string{},
		Progress: core.Progress{TotalTimeSeconds: plan.DurationSeconds}, Trim: plan.Trim, Project: project}
	job = snapshot(job)
	m.records[id] = &record{job: job, plan: plan}
	m.order = append(m.order, id)
	m.publishLocked(job)
	m.queue <- id
	return snapshot(job), nil
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, exists := m.records[id]
	if !exists {
		return fmt.Errorf("unknown job %s", id)
	}
	if rec.job.State == core.JobCancelling || rec.job.State == core.JobCancelled {
		return nil
	}
	if terminal(rec.job.State) {
		return fmt.Errorf("job %s has already finished", id)
	}
	if rec.job.State == core.JobQueued {
		rec.job.State = core.JobCancelled
		rec.job.CompletedAt = time.Now()
	} else {
		rec.job.State = core.JobCancelling
		if rec.cancel != nil {
			rec.cancel()
		}
	}
	m.publishLocked(rec.job)
	return nil
}

func (m *Manager) List() []core.Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	jobs := make([]core.Job, 0, len(m.order))
	for _, id := range m.order {
		jobs = append(jobs, snapshot(m.records[id].job))
	}
	return jobs
}

func (m *Manager) Get(id string) (core.Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, exists := m.records[id]
	if !exists {
		return core.Job{}, false
	}
	return snapshot(rec.job), true
}

func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		m.mu.Lock()
		m.closed = true
		for _, rec := range m.records {
			if !terminal(rec.job.State) {
				rec.job.State = core.JobCancelled
				rec.job.CompletedAt = time.Now()
				m.publishLocked(rec.job)
			}
		}
		m.mu.Unlock()
		m.cancel()
		<-m.done
		close(m.events)
		<-m.eventDone
	})
}

func (m *Manager) work() {
	defer close(m.done)
	for {
		select {
		case <-m.ctx.Done():
			return
		case id := <-m.queue:
			m.run(id)
		}
	}
}

func (m *Manager) run(id string) {
	m.mu.Lock()
	rec := m.records[id]
	if rec == nil || rec.job.State != core.JobQueued || m.closed {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(m.ctx)
	rec.cancel = cancel
	rec.job.State = core.JobPreparing
	rec.job.StartedAt = time.Now()
	m.publishLocked(rec.job)
	plan := rec.plan
	rec.job.State = core.JobRunning
	m.publishLocked(rec.job)
	m.mu.Unlock()
	defer cancel()
	result, err := m.runner.Start(ctx, plan, core.Callbacks{
		Progress: func(progress core.Progress) { m.progress(id, progress) },
		Log:      func(line string) { m.log(id, line) },
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	rec.cancel = nil
	rec.job.CompletedAt = time.Now()
	if err == nil {
		// Successful atomic publication is the commit point. A late cancel cannot
		// turn a complete, published output into a misleading cancelled state.
		rec.job.State = core.JobCompleted
		rec.job.Result = &result
		rec.job.Progress.Percent = 100
		rec.job.Progress.Finished = true
		rec.job.Progress.OutputSizeBytes = result.OutputSizeBytes
	} else if rec.job.State == core.JobCancelling || rec.job.State == core.JobCancelled || errors.Is(err, context.Canceled) {
		rec.job.State = core.JobCancelled
	} else {
		rec.job.State = core.JobFailed
		rec.job.Error = err.Error()
	}
	m.publishLocked(rec.job)
}

func (m *Manager) progress(id string, progress core.Progress) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[id]
	if rec == nil || terminal(rec.job.State) || rec.job.State == core.JobCancelling {
		return
	}
	if progress.Percent >= 100 {
		progress.Percent = 99.9
	}
	rec.job.Progress = progress
	if time.Since(rec.lastProgress) >= 150*time.Millisecond || progress.Finished {
		rec.lastProgress = time.Now()
		m.publishLocked(rec.job)
	}
}

func (m *Manager) log(id, line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[id]
	if rec == nil || terminal(rec.job.State) {
		return
	}
	if len(line) > 2048 {
		line = line[:2048] + "…"
	}
	rec.job.Logs = append(rec.job.Logs, line)
	rec.logBytes += len(line)
	for len(rec.job.Logs) > logLineLimit || rec.logBytes > logByteLimit {
		rec.logBytes -= len(rec.job.Logs[0])
		rec.job.Logs = rec.job.Logs[1:]
	}
	if time.Since(rec.lastLog) >= 250*time.Millisecond {
		rec.lastLog = time.Now()
		m.publishLocked(rec.job)
	}
}

func (m *Manager) publishLocked(job core.Job) {
	if m.persist != nil && m.savedStates[job.ID] != job.State {
		all := make([]core.Job, 0, len(m.order))
		for _, id := range m.order {
			all = append(all, snapshot(m.records[id].job))
		}
		if err := m.persist(all); err != nil {
			m.persistenceError = err.Error()
		} else {
			m.persistenceError = ""
			m.savedStates[job.ID] = job.State
		}
	}
	// Event delivery cannot block execution. Drop an old snapshot if a UI is slow;
	// List/Get always retain the authoritative state and bounded log history.
	select {
	case m.events <- snapshot(job):
	default:
		select {
		case <-m.events:
		default:
		}
		m.events <- snapshot(job)
	}
}

func (m *Manager) pruneHistory() {
	if len(m.records) < historyLimit {
		return
	}
	for i, id := range m.order {
		if terminal(m.records[id].job.State) {
			delete(m.records, id)
			delete(m.savedStates, id)
			m.order = append(m.order[:i], m.order[i+1:]...)
			return
		}
	}
}

func snapshot(job core.Job) core.Job {
	if job.Project != nil {
		p := *job.Project
		p.Inputs = append([]core.Source(nil), p.Inputs...)
		p.Settings.RawArguments = append([]string(nil), p.Settings.RawArguments...)
		if p.Settings.Trim != nil {
			trim := *p.Settings.Trim
			p.Settings.Trim = &trim
		}
		job.Project = &p
	}
	job.Logs = append([]string{}, job.Logs...)
	if job.Trim != nil {
		trim := *job.Trim
		job.Trim = &trim
	}
	if job.Result != nil {
		result := *job.Result
		job.Result = &result
	}
	return job
}

func terminal(state string) bool {
	return state == core.JobCompleted || state == core.JobFailed || state == core.JobCancelled || state == core.JobInterrupted
}
