package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/pkkulhari/backme/internal/backup"
	"github.com/pkkulhari/backme/internal/config"
	"github.com/pkkulhari/backme/internal/s3"
	"github.com/pkkulhari/backme/internal/scheduler"
)

// E2ETestSuite defines the test suite for end-to-end testing
type E2ETestSuite struct {
	suite.Suite
	cfg        *config.Config
	s3Client   *s3.Client
	testBucket string
	testDir    string
	backup     *backup.Service
	scheduler  *scheduler.Scheduler
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (s *E2ETestSuite) SetupSuite() {
	var err error
	// Load test configuration
	s.cfg = &config.Config{
		AWS: config.AWSConfig{
			Region:          "us-east-1",
			AccessKeyID:     "test",
			SecretAccessKey: "test",
			Bucket:          "backme-e2e-test-" + time.Now().Format("20060102150405"),
		},
	}
	s.testBucket = s.cfg.AWS.Bucket

	// Create temporary test directory
	testDir, err := os.MkdirTemp("", "backme-e2e-*")
	s.Require().NoError(err)
	s.testDir = testDir

	// Initialize S3 client with localstack endpoint
	s.s3Client, err = s3.New(s.cfg, &s3.Options{
		Endpoint: "http://s3.localhost.localstack.cloud:4566", // LocalStack S3 endpoint
	})
	s.Require().NoError(err)

	// Create test bucket
	ctx := context.Background()
	err = s.s3Client.CreateBucket(ctx)
	s.Require().NoError(err)

	// Initialize backup service
	s.backup = backup.New(s.cfg, s.s3Client)

	// Initialize scheduler service
	s.scheduler = scheduler.New(s.cfg)
}

func (s *E2ETestSuite) TearDownSuite() {
	// Cleanup test directory
	os.RemoveAll(s.testDir)

	// Cleanup test bucket
	if s.s3Client != nil {
		if err := s.s3Client.DeleteBucket(context.Background()); err != nil {
			s.T().Logf("Failed to delete test bucket: %v", err)
		}
	}
}

// Helper function to create test files
func (s *E2ETestSuite) createTestFiles() []string {
	files := []string{
		"test1.txt",
		"test2.txt",
		"subdir/test3.txt",
	}

	for _, f := range files {
		path := filepath.Join(s.testDir, f)
		dir := filepath.Dir(path)
		s.Require().NoError(os.MkdirAll(dir, 0755))
		s.Require().NoError(os.WriteFile(path, []byte("test content"), 0644))
	}

	return files
}

// Helper function to verify backup in S3
func (s *E2ETestSuite) verifyBackupInS3(_ string, expectedFiles []string) {
	ctx := context.Background()

	// List all objects in the bucket
	objects, err := s.s3Client.ListObjects(ctx, "")
	s.Require().NoError(err)

	// Verify all expected files exist in the backup
	for _, expectedFile := range expectedFiles {
		found := false
		for _, obj := range objects {
			if obj == expectedFile {
				found = true
				break
			}
		}
		s.True(found, "Expected file %s not found in backup", expectedFile)
	}
}
