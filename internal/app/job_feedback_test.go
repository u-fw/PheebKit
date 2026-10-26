package app

import (
	"strings"
	"testing"

	"pheebkit/internal/core"
)

func feedbackTestJob(id, name, state string, percent float64) core.Job {
	return core.Job{
		ID:    id,
		Input: core.Source{DisplayName: name},
		State: state,
		Progress: core.Progress{
			Percent: percent,
		},
	}
}

func TestJobFeedbackAggregatesOnlyTheCurrentEpisode(t *testing.T) {
	tracker := newJobFeedbackTracker()
	old := tracker.observe(feedbackTestJob("old", "old.mp4", core.JobCompleted, 100), "zh-CN")
	if old.taskbar.State != taskbarHidden || old.notification != nil {
		t.Fatalf("restored history entered feedback: %+v", old)
	}

	first := tracker.observe(feedbackTestJob("first", "one.mp4", core.JobQueued, 0), "zh-CN")
	second := tracker.observe(feedbackTestJob("second", "two.mp4", core.JobQueued, 0), "zh-CN")
	if first.taskbar.State != taskbarRunning || first.taskbar.Value != 0 ||
		second.taskbar.State != taskbarRunning || second.taskbar.Value != 0 {
		t.Fatalf("queued episode progress = %+v then %+v", first.taskbar, second.taskbar)
	}

	running := tracker.observe(feedbackTestJob("first", "one.mp4", core.JobRunning, 50), "zh-CN")
	if running.taskbar.Value != 25 {
		t.Fatalf("two-job aggregate = %d, want 25", running.taskbar.Value)
	}
	completed := tracker.observe(feedbackTestJob("first", "one.mp4", core.JobCompleted, 100), "zh-CN")
	if completed.taskbar.Value != 50 || completed.notification != nil {
		t.Fatalf("mid-batch completion = %+v", completed)
	}
	finished := tracker.observe(feedbackTestJob("second", "two.mp4", core.JobCompleted, 100), "zh-CN")
	if finished.taskbar.State != taskbarCompleted || finished.taskbar.Value != 100 {
		t.Fatalf("terminal taskbar state = %+v", finished.taskbar)
	}
	if finished.notification == nil ||
		finished.notification.JobID != "second" ||
		!strings.Contains(finished.notification.Body, "2") {
		t.Fatalf("batch notification = %+v", finished.notification)
	}
	duplicate := tracker.observe(feedbackTestJob("second", "two.mp4", core.JobCompleted, 100), "zh-CN")
	if duplicate.notification != nil {
		t.Fatal("duplicate terminal event emitted another notification")
	}
}

func TestJobFeedbackPauseCancelAndNewBatch(t *testing.T) {
	tracker := newJobFeedbackTracker()
	tracker.observe(feedbackTestJob("paused-a", "a.mp4", core.JobQueued, 0), "zh-CN")
	tracker.observe(feedbackTestJob("paused-b", "b.mp4", core.JobQueued, 0), "zh-CN")
	paused := tracker.setPaused(true)
	if paused.State != taskbarPaused {
		t.Fatalf("paused state = %q", paused.State)
	}
	tracker.observe(feedbackTestJob("paused-a", "a.mp4", core.JobCancelled, 0), "zh-CN")
	cancelled := tracker.observe(feedbackTestJob("paused-b", "b.mp4", core.JobCancelled, 0), "zh-CN")
	if cancelled.taskbar.State != taskbarHidden || cancelled.notification != nil {
		t.Fatalf("cancelled episode produced feedback: %+v", cancelled)
	}

	tracker.setPaused(false)
	next := tracker.observe(feedbackTestJob("next", "next.mp4", core.JobRunning, 10), "en-US")
	if next.taskbar.Value != 10 || next.taskbar.State != taskbarRunning {
		t.Fatalf("new episode reused old totals: %+v", next.taskbar)
	}
	failed := tracker.observe(feedbackTestJob("next", "next.mp4", core.JobFailed, 10), "en-US")
	if failed.taskbar.State != taskbarError ||
		failed.notification == nil ||
		!strings.Contains(failed.notification.Body, "failed") {
		t.Fatalf("failed episode = %+v", failed)
	}
}

func TestJobFeedbackCloseClearsWithoutNotification(t *testing.T) {
	tracker := newJobFeedbackTracker()
	tracker.observe(feedbackTestJob("running", "running.mp4", core.JobRunning, 20), "zh-CN")
	tracker.close()
	update := tracker.observe(feedbackTestJob("running", "running.mp4", core.JobCompleted, 100), "zh-CN")
	if update.taskbar.State != taskbarHidden || update.notification != nil {
		t.Fatalf("closed tracker emitted feedback: %+v", update)
	}
}

func TestJobNotificationTextIsSafeAndBounded(t *testing.T) {
	notification := SanitizeJobNotification(JobNotification{
		JobID: "job",
		Title: "PheebKit\n\u0000",
		Body:  "clip]]><script>\n\u0007" + strings.Repeat("长", notificationBodyLimit),
	})

	if strings.ContainsAny(notification.Title, "\n\x00") {
		t.Fatalf("title retained control characters: %q", notification.Title)
	}
	if strings.Contains(notification.Body, "]]>") {
		t.Fatalf("body retained a CDATA terminator: %q", notification.Body)
	}
	if got := []rune(notification.Body); len(got) != notificationBodyLimit {
		t.Fatalf("body rune count = %d, want %d", len(got), notificationBodyLimit)
	}
	if !strings.HasSuffix(notification.Body, "…") {
		t.Fatalf("truncated body has no ellipsis: %q", notification.Body)
	}
}
