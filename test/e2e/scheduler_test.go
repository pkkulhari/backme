package e2e

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/pkkulhari/backme/internal/config"
	"github.com/pkkulhari/backme/internal/scheduler"
)

// TestSchedulerStart tests that the scheduler starts and executes backups correctly
func (s *E2ETestSuite) TestSchedulerStart() {
	// Create test files
	expectedFiles := s.createTestFiles()

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start scheduler with test backup function
	err := s.scheduler.Start(ctx,
		func(ctx context.Context, cfg any) error {
			// We're not testing database backups in this test
			return nil
		},
		func(ctx context.Context, cfg any) error {
			// Direct directory backup
			return s.backup.BackupDirectory(ctx, s.testDir, false, false, nil)
		},
	)
	s.Require().NoError(err)
	defer s.scheduler.Stop()

	// Wait for at least one backup
	time.Sleep(2 * time.Second)

	// Verify backup was created in S3
	s.verifyBackupInS3(filepath.Base(s.testDir), expectedFiles)
}

// TestSchedulerStop tests that the scheduler can be stopped correctly
func (s *E2ETestSuite) TestSchedulerStop() {
	var backupCount int32 = 0

	// Set up context
	ctx := context.Background()

	// Start scheduler with counter
	err := s.scheduler.Start(ctx,
		func(ctx context.Context, cfg any) error {
			// We're not testing database backups in this test
			return nil
		},
		func(ctx context.Context, cfg any) error {
			// Increment backup count
			atomic.AddInt32(&backupCount, 1)
			return nil
		},
	)
	s.Require().NoError(err)

	// Let it run a short while
	time.Sleep(500 * time.Millisecond)

	// Stop the scheduler
	s.scheduler.Stop()

	// Record the current backup count
	countAfterStop := atomic.LoadInt32(&backupCount)

	// Wait a bit more to ensure no more backups happen
	time.Sleep(1 * time.Second)

	// Verify backup count didn't increase after stopping
	s.Equal(countAfterStop, atomic.LoadInt32(&backupCount), "Backup count should not increase after scheduler is stopped")
}

// TestSchedulerWithCustomConfig tests the scheduler with a custom configuration
func (s *E2ETestSuite) TestSchedulerWithCustomConfig() {
	// Create a test directory
	customDir, err := os.MkdirTemp("", "backme-custom-*")
	s.Require().NoError(err)
	defer os.RemoveAll(customDir)

	// Create a test file in the custom directory
	testFile := filepath.Join(customDir, "custom.txt")
	err = os.WriteFile(testFile, []byte("custom content"), 0644)
	s.Require().NoError(err)

	// Configure scheduler with custom config
	customCfg := &config.Config{
		AWS: s.cfg.AWS,
		Schedules: config.Schedules{
			Directories: []config.DirectorySchedule{
				{
					Name:       "custom-dir",
					Expression: "@every 1s", // Run every second instead of every minute
					SourcePath: customDir,
					AWS:        nil, // Use default AWS config
				},
			},
		},
	}

	// Create a new scheduler with the custom config
	customScheduler := scheduler.New(customCfg)

	// Set up context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start scheduler
	var backupExecuted bool
	err = customScheduler.Start(ctx,
		func(ctx context.Context, cfg any) error {
			return nil
		},
		func(ctx context.Context, cfg any) error {
			backupExecuted = true
			dirCfg, ok := cfg.(struct {
				config.DirectorySchedule
				AWS *config.AWSConfig
			})
			s.Require().True(ok, "Expected directory schedule config")

			// Verify the config
			s.Equal("custom-dir", dirCfg.Name)
			s.Equal(customDir, dirCfg.SourcePath)

			// Execute backup
			return s.backup.BackupDirectory(ctx, dirCfg.SourcePath, false, false, dirCfg.AWS)
		},
	)
	s.Require().NoError(err)
	defer customScheduler.Stop()

	// Wait for backup to be executed
	time.Sleep(2 * time.Second)

	// Verify backup was executed
	s.True(backupExecuted, "Backup should have been executed")
}
