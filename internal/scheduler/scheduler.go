package scheduler

import (
	"fmt"
	"keep-streamlit-alive/internal/config"
	"keep-streamlit-alive/internal/executor"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler manages the cron jobs for waking up Streamlit apps
type Scheduler struct {
	cron     *cron.Cron
	config   *config.Config
	executor *executor.PythonExecutor
	jobID    cron.EntryID
}

type printfLogger struct{}

func (l printfLogger) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// NewScheduler creates a new scheduler instance
func NewScheduler(cfg *config.Config, exec *executor.PythonExecutor) *Scheduler {
	// Create cron with seconds precision and logging
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLogger(cron.VerbosePrintfLogger(printfLogger{})),
	)

	return &Scheduler{
		cron:     c,
		config:   cfg,
		executor: exec,
	}
}

// Start begins the cron scheduler
func (s *Scheduler) Start() error {
	fmt.Printf("Starting scheduler with schedule: %s\n", s.config.Schedule)

	// Add the wake-up job
	jobID, err := s.cron.AddFunc(s.config.Schedule, s.wakeUpJob)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.jobID = jobID

	// Start the cron scheduler
	s.cron.Start()

	fmt.Println("Scheduler started successfully!")
	s.printNextRuns()

	return nil
}

// stops the cron scheduler
func (s *Scheduler) Stop() {
	fmt.Println("Stopping scheduler...")
	s.cron.Stop()
	fmt.Println("Scheduler stopped.")
}

// wakeUpJob is the function that gets executed by the cron job
func (s *Scheduler) wakeUpJob() {
	fmt.Printf("\n=== Wake-up job triggered at %s ===\n", time.Now().Format("2006-01-02 15:04:05"))

	if err := s.executor.ExecuteWakeUpScript(s.config.Apps); err != nil {
		fmt.Printf("ERROR: Wake-up job failed: %v\n", err)
	} else {
		fmt.Println("Wake-up job completed successfully!")
	}

	fmt.Printf("=== Next run scheduled for: %s ===\n\n", s.getNextRun().Format("2006-01-02 15:04:05"))
}

// RunOnce executes the wake-up job immediately (for testing)
func (s *Scheduler) RunOnce() error {
	fmt.Println("Running wake-up job immediately...")
	s.wakeUpJob()
	return nil
}

// UpdateSchedule updates the cron schedule
func (s *Scheduler) UpdateSchedule(newSchedule string) error {
	// Remove existing job
	if s.jobID != 0 {
		s.cron.Remove(s.jobID)
	}

	// Add new job with updated schedule
	jobID, err := s.cron.AddFunc(newSchedule, s.wakeUpJob)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	s.jobID = jobID
	s.config.Schedule = newSchedule

	fmt.Printf("Schedule updated to: %s\n", newSchedule)
	s.printNextRuns()

	return nil
}

// getNextRun returns the next scheduled run time
func (s *Scheduler) getNextRun() time.Time {
	entries := s.cron.Entries()
	if len(entries) > 0 {
		return entries[0].Next
	}
	return time.Time{}
}

// printNextRuns prints information about upcoming scheduled runs
func (s *Scheduler) printNextRuns() {
	entries := s.cron.Entries()
	if len(entries) == 0 {
		fmt.Println("No scheduled jobs found.")
		return
	}

	fmt.Printf("Next scheduled runs:\n")
	for i, entry := range entries {
		if i >= 3 { // Show only next 3 runs
			break
		}
		fmt.Printf("  %d. %s\n", i+1, entry.Next.Format("2006-01-02 15:04:05 MST"))
	}
}

// GetStatus returns the current status of the scheduler
func (s *Scheduler) GetStatus() map[string]interface{} {
	entries := s.cron.Entries()

	status := map[string]interface{}{
		"running":    len(entries) > 0,
		"schedule":   s.config.Schedule,
		"apps_count": len(s.config.Apps),
		"next_run":   "",
		"job_count":  len(entries),
	}

	if len(entries) > 0 {
		status["next_run"] = entries[0].Next.Format("2006-01-02 15:04:05 MST")
	}

	return status
}
