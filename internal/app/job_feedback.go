package app

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"pheebkit/internal/core"
)

const (
	taskbarHidden          = "hidden"
	taskbarRunning         = "running"
	taskbarPaused          = "paused"
	taskbarCompleted       = "completed"
	taskbarError           = "error"
	notificationTitleLimit = 96
	notificationBodyLimit  = 240
)

type TaskbarProgress struct {
	State string `json:"state"`
	Value int    `json:"value"`
}

type JobNotification struct {
	JobID string
	Title string
	Body  string
}

type feedbackJob struct {
	state    string
	progress float64
	active   bool
}

type feedbackUpdate struct {
	taskbar      TaskbarProgress
	notification *JobNotification
}

// jobFeedbackTracker groups terminal transitions from the active queue run.
// It keeps old history out of a new progress episode and emits one summary
// instead of flooding the user with a notification for every batch item.
type jobFeedbackTracker struct {
	mu              sync.Mutex
	jobs            map[string]feedbackJob
	episodeOpen     bool
	total           int
	completed       int
	failed          int
	active          int
	paused          bool
	lastJobID       string
	lastJobName     string
	lastTerminal    string
	lastFailureID   string
	lastFailureName string
	closed          bool
}

func newJobFeedbackTracker() *jobFeedbackTracker {
	return &jobFeedbackTracker{jobs: make(map[string]feedbackJob)}
}

func (t *jobFeedbackTracker) observe(job core.Job, locale string) feedbackUpdate {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return feedbackUpdate{taskbar: TaskbarProgress{State: taskbarHidden}}
	}

	state := job.State
	previous, known := t.jobs[job.ID]
	nowActive := activeJobState(state)
	if !known && !nowActive {
		return feedbackUpdate{taskbar: t.snapshotLocked()}
	}
	if !nowActive && !previous.active {
		return feedbackUpdate{taskbar: t.snapshotLocked()}
	}
	if !nowActive && !t.episodeOpen {
		return feedbackUpdate{taskbar: t.snapshotLocked()}
	}
	if !t.episodeOpen {
		t.resetEpisodeLocked()
		t.episodeOpen = true
	}
	if !known && nowActive {
		t.total++
		t.active++
	}
	if known && previous.active && nowActive {
		// Repeated progress events update the percentage but not the tally.
	}
	if known && previous.active && !nowActive {
		t.active--
		switch state {
		case core.JobCompleted:
			t.completed++
		case core.JobFailed:
			t.failed++
			t.lastFailureID = job.ID
			t.lastFailureName = feedbackJobName(job)
		case core.JobCancelled, core.JobInterrupted:
			t.total--
		default:
			t.total--
		}
	}

	t.jobs[job.ID] = feedbackJob{
		state:    state,
		progress: feedbackProgress(state, job.Progress.Percent),
		active:   nowActive,
	}
	if nowActive {
		t.lastTerminal = ""
	}
	if !nowActive {
		t.lastJobID = job.ID
		t.lastJobName = feedbackJobName(job)
		t.lastTerminal = state
	}

	update := feedbackUpdate{taskbar: t.snapshotLocked()}
	if t.active == 0 && t.episodeOpen && (state == core.JobCompleted || state == core.JobFailed) {
		t.episodeOpen = false
		update.notification = t.summaryNotificationLocked(locale)
	}
	return update
}

func (t *jobFeedbackTracker) setPaused(paused bool) TaskbarProgress {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.paused = paused
	return t.snapshotLocked()
}

func (t *jobFeedbackTracker) close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	t.jobs = make(map[string]feedbackJob)
	t.resetEpisodeLocked()
}

func (t *jobFeedbackTracker) snapshotLocked() TaskbarProgress {
	if t.active == 0 {
		if t.lastTerminal == core.JobCompleted || t.lastTerminal == core.JobFailed {
			state := taskbarCompleted
			if t.failed > 0 {
				state = taskbarError
			}
			return TaskbarProgress{State: state, Value: 100}
		}
		return TaskbarProgress{State: taskbarHidden}
	}

	progress := float64(t.completed)
	running := false
	for _, job := range t.jobs {
		if !job.active {
			continue
		}
		if job.state == core.JobRunning {
			running = true
		}
		progress += job.progress
	}
	if t.total <= 0 {
		return TaskbarProgress{State: taskbarHidden}
	}
	value := int(math.Floor(progress / float64(t.total) * 100))
	if value < 0 {
		value = 0
	}
	if value > 99 {
		value = 99
	}
	state := taskbarRunning
	if t.paused && !running {
		state = taskbarPaused
	}
	return TaskbarProgress{State: state, Value: value}
}

func (t *jobFeedbackTracker) summaryNotificationLocked(locale string) *JobNotification {
	if t.completed == 0 && t.failed == 0 {
		return nil
	}
	jobID := t.lastJobID
	jobName := t.lastJobName
	if t.failed > 0 && t.lastFailureID != "" {
		jobID = t.lastFailureID
		jobName = t.lastFailureName
	}
	english := locale == "en-US"
	switch {
	case t.failed > 0 && t.completed > 0:
		if english {
			return &JobNotification{JobID: jobID, Title: "PheebKit · Some jobs need attention", Body: batchSummaryEnglish(t.completed, t.failed)}
		}
		return &JobNotification{JobID: jobID, Title: "PheebKit · 部分任务需要检查", Body: batchSummaryChinese(t.completed, t.failed)}
	case t.failed > 0:
		if english {
			if t.failed == 1 {
				return &JobNotification{JobID: jobID, Title: "PheebKit · Processing failed", Body: jobName + " failed. Open the queue for details."}
			}
			return &JobNotification{JobID: jobID, Title: "PheebKit · Processing failed", Body: batchSummaryEnglish(t.completed, t.failed)}
		}
		if t.failed == 1 {
			return &JobNotification{JobID: jobID, Title: "PheebKit · 处理失败", Body: jobName + " 处理失败。打开队列查看详情。"}
		}
		return &JobNotification{JobID: jobID, Title: "PheebKit · 处理失败", Body: batchSummaryChinese(t.completed, t.failed)}
	case t.completed == 1:
		if english {
			return &JobNotification{JobID: jobID, Title: "PheebKit · Processing complete", Body: jobName + " is ready."}
		}
		return &JobNotification{JobID: jobID, Title: "PheebKit · 处理完成", Body: jobName + " 已完成。"}
	default:
		if english {
			return &JobNotification{JobID: jobID, Title: "PheebKit · Batch complete", Body: batchSummaryEnglish(t.completed, 0)}
		}
		return &JobNotification{JobID: jobID, Title: "PheebKit · 批量处理完成", Body: batchSummaryChinese(t.completed, 0)}
	}
}

func (t *jobFeedbackTracker) resetEpisodeLocked() {
	t.total = 0
	t.completed = 0
	t.failed = 0
	t.active = 0
	t.lastJobID = ""
	t.lastJobName = ""
	t.lastTerminal = ""
	t.lastFailureID = ""
	t.lastFailureName = ""
}

func activeJobState(state string) bool {
	return state == core.JobQueued ||
		state == core.JobPreparing ||
		state == core.JobRunning ||
		state == core.JobCancelling
}

func feedbackProgress(state string, percent float64) float64 {
	if state != core.JobRunning {
		return 0
	}
	if percent < 0 {
		return 0
	}
	if percent >= 100 {
		return 0.999
	}
	return percent / 100
}

func feedbackJobName(job core.Job) string {
	if job.Input.DisplayName != "" {
		return job.Input.DisplayName
	}
	return job.ID
}

// Windows Toast text is embedded inside CDATA in the reviewed Wails v2
// backend. Strip control characters, collapse whitespace, neutralize CDATA
// terminators and cap user-derived filenames before handing them to the OS.
func SanitizeJobNotification(notification JobNotification) JobNotification {
	notification.Title = boundedNotificationText(
		notification.Title,
		notificationTitleLimit,
	)
	notification.Body = boundedNotificationText(
		notification.Body,
		notificationBodyLimit,
	)
	if notification.Title == "" {
		notification.Title = "PheebKit"
	}
	return notification
}

func boundedNotificationText(value string, limit int) string {
	value = strings.Join(strings.FieldsFunc(value, func(value rune) bool {
		return unicode.IsControl(value) || unicode.IsSpace(value)
	}), " ")
	value = strings.ReplaceAll(value, "]]>", "]] >")
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}

func batchSummaryChinese(completed, failed int) string {
	if failed == 0 {
		return pluralChinese(completed, "个任务已完成。点击查看队列。")
	}
	return pluralChinese(completed, "个任务完成，") + pluralChinese(failed, "个失败。点击查看队列。")
}

func batchSummaryEnglish(completed, failed int) string {
	if failed == 0 {
		return pluralEnglish(completed, "job completed.", "jobs completed.") + " Open the queue for details."
	}
	return pluralEnglish(completed, "job completed, ", "jobs completed, ") +
		pluralEnglish(failed, "job failed.", "jobs failed.") + " Open the queue for details."
}

func pluralChinese(value int, suffix string) string {
	return strconv.Itoa(value) + suffix
}

func pluralEnglish(value int, singular, plural string) string {
	if value == 1 {
		return strconv.Itoa(value) + " " + singular
	}
	return strconv.Itoa(value) + " " + plural
}
