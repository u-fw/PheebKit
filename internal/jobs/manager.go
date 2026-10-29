// Package jobs owns a serial, cancellable job queue independent of the UI shell.
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"pheebkit/internal/core"
)

const (
	queueLimit          = 32
	historyLimit        = 200
	logByteLimit        = 64 * 1024
	logLineLimit        = 120
	initialPersistRetry = 500 * time.Millisecond
	maximumPersistRetry = 30 * time.Second
)

type record struct {
	job          core.Job
	plan         core.ExecutionPlan
	cancel       context.CancelFunc
	lastProgress time.Time
	lastLog      time.Time
	logBytes     int
	lastEvent    uint64
}

type Manager struct {
	runner           core.Runner
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.Mutex
	records          map[string]*record
	order            []string
	pending          []string
	wake             chan struct{}
	paused           bool
	queueRevision    uint64
	events           chan core.Job
	done             chan struct{}
	eventDone        chan struct{}
	closed           bool
	closeOnce        sync.Once
	persistMu        sync.Mutex
	persist          func([]core.Job) error
	persistenceError string
	savedStates      map[string]string
	persistWake      chan struct{}
	persistStop      chan struct{}
	persistDone      chan struct{}
	eventSequence    uint64
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
		wake: make(chan struct{}, 1), events: make(chan core.Job, 64),
		done: make(chan struct{}), eventDone: make(chan struct{}),
		persistWake: make(chan struct{}, 1), persistStop: make(chan struct{}),
		persistDone: make(chan struct{})}
	go m.persistenceWorker()
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
	var projects []core.Project
	if project != nil {
		projects = []core.Project{*project}
	}
	queued, err := m.EnqueueBatch([]core.ExecutionPlan{plan}, projects)
	if err != nil {
		return core.Job{}, err
	}
	return queued[0], nil
}

// EnqueueBatch reserves the entire batch before publishing any jobs. A rejected
// request cannot leave an unexpected partially queued batch behind.
func (m *Manager) EnqueueBatch(plans []core.ExecutionPlan, projects []core.Project) ([]core.Job, error) {
	if len(plans) == 0 || len(plans) > queueLimit || (projects != nil && len(projects) != len(plans)) {
		return nil, fmt.Errorf("batch requires 1–%d plans and matching projects", queueLimit)
	}
	plans = append([]core.ExecutionPlan(nil), plans...)
	for i := range plans {
		if plans[i].JobID == "" {
			var bytes [16]byte
			if _, err := rand.Read(bytes[:]); err != nil {
				return nil, fmt.Errorf("generate job ID: %w", err)
			}
			plans[i].JobID = hex.EncodeToString(bytes[:])
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, fmt.Errorf("job manager is closed")
	}
	if len(m.pending)+len(plans) > queueLimit {
		return nil, fmt.Errorf("queue is full (%d pending jobs maximum)", queueLimit)
	}
	ids := make(map[string]bool, len(plans))
	outputs := make(map[string]bool)
	for _, rec := range m.records {
		if !terminal(rec.job.State) && rec.plan.Output.URI != "" {
			key, err := outputKey(rec.plan.Output.URI)
			if err != nil {
				return nil, err
			}
			outputs[key] = true
		}
	}
	for _, plan := range plans {
		if ids[plan.JobID] || m.records[plan.JobID] != nil {
			return nil, fmt.Errorf("duplicate job ID %s", plan.JobID)
		}
		ids[plan.JobID] = true
		if plan.Output.URI == "" {
			continue
		}
		key, err := outputKey(plan.Output.URI)
		if err != nil {
			return nil, err
		}
		if outputs[key] {
			return nil, fmt.Errorf("output path is already reserved by an active job: %s", plan.Output.URI)
		}
		outputs[key] = true
		if _, err := os.Stat(plan.Output.URI); err == nil {
			return nil, fmt.Errorf("output already exists; choose a new output path: %s", plan.Output.URI)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("check output: %w", err)
		}
	}
	active := 0
	for _, rec := range m.records {
		if !terminal(rec.job.State) {
			active++
		}
	}
	if active+len(plans) > historyLimit {
		return nil, fmt.Errorf("job history is full")
	}
	for len(m.records)+len(plans) > historyLimit {
		m.pruneHistory()
	}
	queued := make([]core.Job, len(plans))
	for i, plan := range plans {
		plan.Args = append([]string(nil), plan.Args...)
		if plan.SubtitleAsset != nil {
			copy := *plan.SubtitleAsset
			plan.SubtitleAsset = &copy
		}
		if plan.ExpectedOutput != nil {
			expected := *plan.ExpectedOutput
			expected.AudioSourceChannels = append([]int(nil), expected.AudioSourceChannels...)
			plan.ExpectedOutput = &expected
		}
		if plan.Trim != nil {
			copy := *plan.Trim
			plan.Trim = &copy
		}
		var project *core.Project
		if projects != nil {
			project = &projects[i]
		}
		job := snapshot(core.Job{ID: plan.JobID, ProjectID: plan.ProjectID, Input: plan.Input, Output: plan.Output, SourceFingerprint: plan.SourceFingerprint,
			State: core.JobQueued, QueuedAt: time.Now(), Logs: []string{},
			Progress: core.Progress{TotalTimeSeconds: plan.DurationSeconds}, Trim: plan.Trim, Project: project})
		m.records[job.ID] = &record{job: job, plan: plan}
		m.order = append(m.order, job.ID)
		m.pending = append(m.pending, job.ID)
		queued[i] = snapshot(job)
	}
	m.queueRevision++
	m.schedulePersistLocked()
	for i, job := range queued {
		queued[i] = m.emitLocked(job)
	}
	m.wakeLocked()
	return queued, nil
}

// Existing parent symlinks and Windows casing share the same reservation key.
func outputKey(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve output: %w", err)
	}
	if parent, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
		abs = filepath.Join(parent, filepath.Base(abs))
	}
	if runtime.GOOS == "windows" {
		abs = strings.ToLower(abs)
	}
	return filepath.Clean(abs), nil
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
		for i, pendingID := range m.pending {
			if pendingID == id {
				m.pending = append(m.pending[:i], m.pending[i+1:]...)
				m.queueRevision++
				break
			}
		}
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
		rec := m.records[id]
		job := snapshot(rec.job)
		job.EventSequence = rec.lastEvent
		jobs = append(jobs, job)
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
	job := snapshot(rec.job)
	job.EventSequence = rec.lastEvent
	return job, true
}

func (m *Manager) Close() error {
	first := false
	m.closeOnce.Do(func() {
		first = true
		m.mu.Lock()
		m.closed = true
		m.pending = nil
		m.queueRevision++
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
		close(m.persistStop)
		<-m.persistDone
	})
	// Once stopped, a caller may explicitly retry a failed final write without
	// restarting any job or racing the persistence worker.
	if !first && m.PersistenceError() != "" {
		return m.persistCurrent()
	}
	if message := m.PersistenceError(); message != "" {
		return errors.New(message)
	}
	return nil
}

func (m *Manager) work() {
	defer close(m.done)
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.wake:
			for {
				m.mu.Lock()
				if m.closed || m.paused || len(m.pending) == 0 {
					m.mu.Unlock()
					break
				}
				id := m.pending[0]
				m.pending = m.pending[1:]
				m.queueRevision++
				rec := m.records[id]
				rec.job.State = core.JobPreparing
				rec.job.StartedAt = time.Now()
				m.publishLocked(rec.job)
				m.mu.Unlock()
				m.run(id)
			}
		}
	}
}

func (m *Manager) run(id string) {
	m.mu.Lock()
	rec := m.records[id]
	// A cancel may arrive after the scheduler claims a job, before the runner
	// receives its cancellation context. Complete that cancellation here.
	if rec != nil && rec.job.State == core.JobCancelling && !m.closed {
		rec.job.State = core.JobCancelled
		rec.job.CompletedAt = time.Now()
		m.publishLocked(rec.job)
		m.mu.Unlock()
		return
	}
	if rec == nil || rec.job.State != core.JobPreparing || m.closed {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(m.ctx)
	rec.cancel = cancel
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
		m.schedulePersistLocked()
	}
	m.emitLocked(job)
}

func (m *Manager) schedulePersistLocked() {
	if m.persist == nil {
		return
	}
	select {
	case m.persistWake <- struct{}{}:
	default:
	}
}

func (m *Manager) persistenceWorker() {
	defer close(m.persistDone)
	var retryTimer *time.Timer
	var retry <-chan time.Time
	retryDelay := initialPersistRetry
	stopTimer := func() {
		if retryTimer != nil {
			retryTimer.Stop()
			retryTimer = nil
			retry = nil
		}
	}
	scheduleRetry := func() {
		retryTimer = time.NewTimer(retryDelay)
		retry = retryTimer.C
		retryDelay = min(retryDelay*2, maximumPersistRetry)
	}
	for {
		select {
		case <-m.persistStop:
			stopTimer()
			_ = m.persistCurrent()
			return
		case <-m.persistWake:
			if retry != nil {
				continue
			}
			if err := m.persistCurrent(); err != nil {
				scheduleRetry()
			} else {
				retryDelay = initialPersistRetry
			}
		case <-retry:
			retryTimer = nil
			retry = nil
			if err := m.persistCurrent(); err != nil {
				scheduleRetry()
			} else {
				retryDelay = initialPersistRetry
			}
		}
	}
}

func (m *Manager) persistCurrent() error {
	m.persistMu.Lock()
	defer m.persistMu.Unlock()
	m.mu.Lock()
	save := m.persist
	if save == nil {
		m.mu.Unlock()
		return nil
	}
	all := make([]core.Job, 0, len(m.order))
	for _, id := range m.order {
		job := snapshot(m.records[id].job)
		job.EventSequence = 0
		all = append(all, job)
	}
	m.mu.Unlock()

	err := save(all)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.persistenceError = err.Error()
		return err
	}
	m.persistenceError = ""
	for _, saved := range all {
		rec := m.records[saved.ID]
		if rec != nil && rec.job.State == saved.State {
			m.savedStates[saved.ID] = saved.State
		}
	}
	return nil
}

func (m *Manager) emitLocked(job core.Job) core.Job {
	m.eventSequence++
	job.EventSequence = m.eventSequence
	if rec := m.records[job.ID]; rec != nil {
		rec.lastEvent = job.EventSequence
	}
	event := snapshot(job)
	// Event delivery cannot block execution. Drop an old snapshot if a UI is slow;
	// List/Get always retain the authoritative state and bounded log history.
	select {
	case m.events <- event:
	default:
		select {
		case <-m.events:
		default:
		}
		m.events <- event
	}
	return event
}

func (m *Manager) pruneHistory() {
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
		p.Settings = core.CloneSettings(p.Settings)
		job.Project = &p
	}
	job.Logs = append([]string{}, job.Logs...)
	if job.Trim != nil {
		trim := *job.Trim
		job.Trim = &trim
	}
	if job.Result != nil {
		result := *job.Result
		if result.Verification != nil {
			verification := *result.Verification
			verification.Warnings = append([]string{}, verification.Warnings...)
			result.Verification = &verification
		}
		job.Result = &result
	}
	return job
}

func terminal(state string) bool {
	return state == core.JobCompleted || state == core.JobFailed || state == core.JobCancelled || state == core.JobInterrupted
}
