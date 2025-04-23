package e2e

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkkulhari/backme/internal/config"
)

func (s *E2ETestSuite) TestBackupFlow() {
	// Test case 1: Single file backup
	s.Run("SingleFileBackup", func() {
		// Create a single test file
		content := "test content"
		filename := "single-test.txt"
		path := filepath.Join(s.testDir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		s.Require().NoError(err)

		// Perform backup
		ctx := context.Background()
		err = s.backup.BackupDirectory(ctx, s.testDir, false, false, nil)
		s.Require().NoError(err)

		// Verify backup
		s.verifyBackupInS3(filepath.Base(s.testDir), []string{filename})
	})

	// Test case 2: Directory backup
	s.Run("DirectoryBackup", func() {
		// Create test directory structure
		files := s.createTestFiles()

		// Perform backup
		ctx := context.Background()
		err := s.backup.BackupDirectory(ctx, s.testDir, false, false, nil)
		s.Require().NoError(err)

		// Verify backup
		s.verifyBackupInS3(filepath.Base(s.testDir), files)
	})

	// Test case 3: Incremental backup
	s.Run("IncrementalBackup", func() {
		// Initial backup
		files := s.createTestFiles()
		ctx := context.Background()
		err := s.backup.BackupDirectory(ctx, s.testDir, true, false, nil)
		s.Require().NoError(err)

		// Add new file
		newFile := "incremental.txt"
		path := filepath.Join(s.testDir, newFile)
		err = os.WriteFile(path, []byte("incremental content"), 0644)
		s.Require().NoError(err)
		files = append(files, newFile)

		// Perform incremental backup
		time.Sleep(1 * time.Second) // Ensure different timestamp
		err = s.backup.BackupDirectory(ctx, s.testDir, true, false, nil)
		s.Require().NoError(err)

		// Verify backup contains all files
		s.verifyBackupInS3(filepath.Base(s.testDir), files)
	})

	// Test case 4: Error handling
	s.Run("ErrorHandling", func() {
		ctx := context.Background()

		// Test non-existent source
		err := s.backup.BackupDirectory(ctx, "/nonexistent/path", false, false, nil)
		s.Require().Error(err)
		s.Contains(err.Error(), "source path does not exist")

		// Test invalid AWS config
		invalidCfg := &config.AWSConfig{
			Region:          "invalid-region",
			AccessKeyID:     "invalid-key",
			SecretAccessKey: "invalid-secret",
			Bucket:          "invalid-bucket-name-@#$%",
		}
		err = s.backup.BackupDirectory(ctx, s.testDir, false, false, invalidCfg)
		s.Require().Error(err)
	})
}

func (s *E2ETestSuite) TestScheduledBackup() {
	// Test scheduled backup execution
	s.Run("ScheduledBackup", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create test files
		files := s.createTestFiles()

		// Start scheduler with test backup function
		err := s.scheduler.Start(ctx,
			func(ctx context.Context, cfg any) error {
				return s.backup.BackupDatabase(ctx, nil, nil)
			},
			func(ctx context.Context, cfg any) error {
				return s.backup.BackupDirectory(ctx, s.testDir, false, false, nil)
			},
		)
		s.Require().NoError(err)
		defer s.scheduler.Stop()

		// Wait for at least one backup
		time.Sleep(2 * time.Second)

		// Verify backup was created
		s.verifyBackupInS3(filepath.Base(s.testDir), files)
	})
}

// TestDirectBackupExecution tests direct backup execution without scheduling
func (s *E2ETestSuite) TestDirectBackupExecution() {
	// Create test files
	expectedFiles := s.createTestFiles()

	// Directly execute backup
	ctx := context.Background()
	err := s.backup.BackupDirectory(ctx, s.testDir, false, false, nil)
	s.Require().NoError(err)

	// Verify backup was created in S3
	s.verifyBackupInS3(filepath.Base(s.testDir), expectedFiles)
}

// TestIncrementalBackup tests that incremental backups work correctly
func (s *E2ETestSuite) TestIncrementalBackup() {
	// Create initial test files
	files := s.createTestFiles()

	// Perform initial backup with sync mode
	ctx := context.Background()
	err := s.backup.BackupDirectory(ctx, s.testDir, true, false, nil)
	s.Require().NoError(err)

	// Add a new file
	newFile := "incremental.txt"
	path := filepath.Join(s.testDir, newFile)
	err = os.WriteFile(path, []byte("incremental content"), 0644)
	s.Require().NoError(err)
	files = append(files, newFile)

	// Perform incremental backup
	time.Sleep(1 * time.Second) // Ensure different timestamp
	err = s.backup.BackupDirectory(ctx, s.testDir, true, false, nil)
	s.Require().NoError(err)

	// Verify backup contains all files
	s.verifyBackupInS3(filepath.Base(s.testDir), files)
}
