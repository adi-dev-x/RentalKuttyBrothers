package cron

import (
	"fmt"
	"time"

	"myproject/pkg/irrl"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps the cron runner and holds a reference to the service.
type Scheduler struct {
	runner  *cron.Cron
	service irrl.Service
}

// NewScheduler creates a new Scheduler running in IST (UTC+5:30).
func NewScheduler(service irrl.Service) *Scheduler {
	ist := time.FixedZone("IST", 5*3600+30*60)

	// WithLocation pins all schedule expressions to IST.
	runner := cron.New(cron.WithLocation(ist))
	return &Scheduler{runner: runner, service: service}
}

// Start registers jobs and starts the cron runner in the background.
func (s *Scheduler) Start() {
	// "0 0 * * *" = every day at 00:00 in the scheduler's timezone (IST).
	_, err := s.runner.AddFunc("0 0 * * *", func() {
		fmt.Println("[cron] Running nightly current_amount update —", time.Now().In(time.FixedZone("IST", 5*3600+30*60)).Format(time.RFC3339))
		if err := s.service.UpdateCurrentAmounts(); err != nil {
			fmt.Println("[cron] ERROR updating current_amount:", err)
		} else {
			fmt.Println("[cron] current_amount updated successfully")
		}
	})
	if err != nil {
		fmt.Println("[cron] Failed to register job:", err)
		return
	}
	s.runner.Start()
	fmt.Println("[cron] Scheduler started — UpdateCurrentAmounts fires daily at 00:00 IST")
}

// Stop gracefully stops the cron runner.
func (s *Scheduler) Stop() {
	s.runner.Stop()
}
