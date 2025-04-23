package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkkulhari/backme/internal/backup"
	"github.com/pkkulhari/backme/internal/config"
	"github.com/pkkulhari/backme/internal/s3"
	"github.com/pkkulhari/backme/internal/scheduler"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the backme worker process",
	Long:  `Start the backme worker process that runs in the background and executes scheduled backups`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a background context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s3Client, err := s3.New(cfg, nil)
		if err != nil {
			return fmt.Errorf("failed to initialize S3 client: %w", err)
		}
		backupSvc := backup.New(cfg, s3Client)

		sched := scheduler.New(cfg)

		log.Info().Msg("Starting backme worker process")
		if err := sched.Start(ctx,
			func(ctx context.Context, cfg any) error {
				// Handle database backups
				dbConfig, ok := cfg.(struct {
					config.DatabaseConfig
					AWS *config.AWSConfig
				})
				if !ok {
					return fmt.Errorf("invalid database configuration in scheduler")
				}
				return backupSvc.BackupDatabase(ctx, &dbConfig.DatabaseConfig, dbConfig.AWS)
			},
			func(ctx context.Context, cfg any) error {
				// Handle directory backups
				dirConfig, ok := cfg.(struct {
					config.DirectorySchedule
					AWS *config.AWSConfig
				})
				if !ok {
					return fmt.Errorf("invalid directory configuration in scheduler")
				}
				return backupSvc.BackupDirectory(ctx, dirConfig.SourcePath, dirConfig.Sync, dirConfig.Delete, dirConfig.AWS)
			},
		); err != nil {
			return fmt.Errorf("failed to start scheduler: %w", err)
		}

		// Setup signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info().Msg("Stopping backme worker process")
		sched.Stop()
		return nil
	},
}

func init() {
	workerCmd.Flags().String("pidfile", "/var/run/backme.pid", "Path to PID file")
	rootCmd.AddCommand(workerCmd)
}
