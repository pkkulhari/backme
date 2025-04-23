package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkkulhari/backme/internal/config"
	"github.com/pkkulhari/backme/internal/s3"
	"github.com/rs/zerolog/log"
)

type Service struct {
	cfg      *config.Config
	s3Client *s3.Client
}

func New(cfg *config.Config, s3Client *s3.Client) *Service {
	return &Service{
		cfg:      cfg,
		s3Client: s3Client,
	}
}

func (s *Service) getS3ClientForConfig(awsCfg *config.AWSConfig) (*s3.Client, error) {
	if awsCfg == nil {
		return s.s3Client, nil
	}

	newCfg := &config.Config{
		AWS: config.AWSConfig{
			Region:          s.cfg.AWS.Region,
			AccessKeyID:     s.cfg.AWS.AccessKeyID,
			SecretAccessKey: s.cfg.AWS.SecretAccessKey,
			Bucket:          s.cfg.AWS.Bucket,
			DatabasePrefix:  s.cfg.AWS.DatabasePrefix,
			DirectoryPrefix: s.cfg.AWS.DirectoryPrefix,
		},
	}

	// Override only the properties that are set in awsCfg
	if awsCfg.Region != "" {
		newCfg.AWS.Region = awsCfg.Region
	}
	if awsCfg.AccessKeyID != "" {
		newCfg.AWS.AccessKeyID = awsCfg.AccessKeyID
	}
	if awsCfg.SecretAccessKey != "" {
		newCfg.AWS.SecretAccessKey = awsCfg.SecretAccessKey
	}
	if awsCfg.Bucket != "" {
		newCfg.AWS.Bucket = awsCfg.Bucket
	}
	if awsCfg.DatabasePrefix != "" {
		newCfg.AWS.DatabasePrefix = awsCfg.DatabasePrefix
	}
	if awsCfg.DirectoryPrefix != "" {
		newCfg.AWS.DirectoryPrefix = awsCfg.DirectoryPrefix
	}

	return s3.New(newCfg, nil)
}

func (s *Service) getDatabaseConfigForConfig(dbCfg *config.DatabaseConfig) *config.DatabaseConfig {
	if dbCfg == nil {
		return &s.cfg.Database
	}

	newCfg := config.DatabaseConfig{
		Host:     s.cfg.Database.Host,
		Port:     s.cfg.Database.Port,
		User:     s.cfg.Database.User,
		Password: s.cfg.Database.Password,
		Name:     s.cfg.Database.Name,
	}

	// Override only the properties that are set in dbCfg
	if dbCfg.Host != "" {
		newCfg.Host = dbCfg.Host
	}
	if dbCfg.Port != 0 {
		newCfg.Port = dbCfg.Port
	}
	if dbCfg.User != "" {
		newCfg.User = dbCfg.User
	}
	if dbCfg.Password != "" {
		newCfg.Password = dbCfg.Password
	}
	if dbCfg.Name != "" {
		newCfg.Name = dbCfg.Name
	}

	return &newCfg
}

func (s *Service) BackupDatabase(ctx context.Context, dbCfg *config.DatabaseConfig, awsCfg *config.AWSConfig) error {
	if dbCfg == nil {
		return fmt.Errorf("database configuration is required")
	}

	log.Info().Msgf("Starting backup of database %s", dbCfg.Name)

	s3Client, err := s.getS3ClientForConfig(awsCfg)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}
	dbConfig := s.getDatabaseConfigForConfig(dbCfg)

	// Create a temporary file for the dump
	tmpFile, err := os.CreateTemp("", "pg_dump_*.sql")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Prepare pg_dump command
	cmd := exec.Command("pg_dump",
		"-h", dbConfig.Host,
		"-p", fmt.Sprintf("%d", dbConfig.Port),
		"-U", dbConfig.User,
		"-F", "p",
		"-f", tmpFile.Name(),
		dbConfig.Name,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute pg_dump: %w", err)
	}

	// Upload to S3
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open dump file: %w", err)
	}
	defer file.Close()

	// Use AWS config from schedule if provided, otherwise use default
	prefix := s.cfg.AWS.DatabasePrefix
	if awsCfg != nil {
		prefix = awsCfg.DatabasePrefix
	}

	key := s3.GetObjectKey(prefix, fmt.Sprintf("%s_%s.sql", dbConfig.Name, time.Now().Format("2006-01-02_15-04-05")))
	if err := s3Client.Upload(ctx, key, file); err != nil {
		return fmt.Errorf("failed to upload dump to S3: %w", err)
	}

	log.Info().Msgf("Successfully backed up database %s to S3", dbConfig.Name)
	return nil
}

func (s *Service) BackupDirectory(ctx context.Context, sourcePath string, sync bool, shouldDelete bool, awsCfg *config.AWSConfig) error {
	log.Info().Msgf("Starting backup of directory %s", sourcePath)

	// Get appropriate S3 client
	s3Client, err := s.getS3ClientForConfig(awsCfg)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", sourcePath)
	}

	// Get list of S3 files if sync is enabled
	var s3FileMap map[string]struct{}
	if sync {
		// Use AWS config from schedule if provided, otherwise use default
		prefix := ""
		if awsCfg != nil && awsCfg.DirectoryPrefix != "" {
			prefix = awsCfg.DirectoryPrefix + "/"
		} else if s.cfg.AWS.DirectoryPrefix != "" {
			prefix = s.cfg.AWS.DirectoryPrefix + "/"
		}

		s3Files, err := s3Client.ListObjects(ctx, prefix)
		if err != nil {
			return fmt.Errorf("failed to list objects in S3: %w", err)
		}

		// Create a map of S3 files for quick lookup
		s3FileMap = make(map[string]struct{})
		for _, file := range s3Files {
			s3FileMap[file] = struct{}{}
		}
	}

	// Use AWS config from schedule if provided, otherwise use default
	prefix := s.cfg.AWS.DirectoryPrefix
	if awsCfg != nil {
		prefix = awsCfg.DirectoryPrefix
	}

	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Calculate relative path for S3 key
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		key := s3.GetObjectKey(prefix, relPath)
		shouldUpload := true

		if sync {
			// Check if file exists in S3
			if _, exists := s3FileMap[key]; exists {
				// File exists, check if it's modified
				metadata, err := s3Client.GetObjectMetadata(ctx, key)
				if err != nil {
					return fmt.Errorf("failed to get S3 object metadata for %s: %w", relPath, err)
				}

				if !info.ModTime().After(*metadata.LastModified) {
					shouldUpload = false
				} else {
					log.Debug().Msgf("Modified file detected: %s", relPath)
				}
			} else {
				log.Debug().Msgf("New file detected: %s", relPath)
			}

			delete(s3FileMap, key)
		}

		if shouldUpload {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if err := s3Client.Upload(ctx, key, file); err != nil {
				return fmt.Errorf("failed to upload file %s to S3: %w", path, err)
			}

			log.Debug().Msgf("Uploaded file: %s", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to backup directory: %w", err)
	}

	// Delete files from S3 that don't exist locally
	if sync && shouldDelete && len(s3FileMap) > 0 {
		for key := range s3FileMap {
			if err := s3Client.DeleteObject(ctx, key); err != nil {
				return fmt.Errorf("failed to delete object %s from S3: %w", key, err)
			}
			log.Debug().Msgf("Deleted file from S3: %s", key)
		}
	}

	log.Info().Msgf("Successfully backed up directory %s to S3", sourcePath)
	return nil
}
