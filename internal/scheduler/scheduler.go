package scheduler

import (
	"context"

	"github.com/pkkulhari/backme/internal/config"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

type BackupFunc func(ctx context.Context, cfg any) error

type Scheduler struct {
	cfg  *config.Config
	cron *cron.Cron
}

func New(cfg *config.Config) *Scheduler {
	return &Scheduler{
		cfg:  cfg,
		cron: cron.New(),
	}
}

func (s *Scheduler) Start(ctx context.Context, dbBackupFunc, dirBackupFunc BackupFunc) error {
	// Schedule database backups
	for _, schedule := range s.cfg.Schedules.Databases {
		dbSchedule := schedule // Create a copy to avoid closure issues
		cronExpr := s.getCronExpression(dbSchedule.Expression)
		if cronExpr == "" {
			log.Error().Str("name", dbSchedule.Name).Msg("Invalid schedule configuration, skipping")
			continue
		}

		_, err := s.cron.AddFunc(cronExpr, func() {
			if err := dbBackupFunc(ctx, struct {
				config.DatabaseConfig
				AWS *config.AWSConfig
			}{
				DatabaseConfig: dbSchedule.Database,
				AWS:            dbSchedule.AWS,
			}); err != nil {
				log.Error().Err(err).
					Str("name", dbSchedule.Name).
					Str("database", dbSchedule.Database.Name).
					Msg("Failed to execute scheduled database backup")
			}
		})
		if err != nil {
			log.Error().Err(err).
				Str("name", dbSchedule.Name).
				Str("database", dbSchedule.Database.Name).
				Msg("Failed to schedule database backup")
			continue
		}

		log.Info().
			Str("name", dbSchedule.Name).
			Str("database", dbSchedule.Database.Name).
			Str("schedule", cronExpr).
			Msg("Scheduled database backup")
	}

	// Schedule directory backups
	for _, schedule := range s.cfg.Schedules.Directories {
		dirSchedule := schedule // Create a copy to avoid closure issues
		cronExpr := s.getCronExpression(dirSchedule.Expression)
		if cronExpr == "" {
			log.Error().Str("name", dirSchedule.Name).Msg("Invalid schedule configuration, skipping")
			continue
		}

		_, err := s.cron.AddFunc(cronExpr, func() {
			if err := dirBackupFunc(ctx, struct {
				config.DirectorySchedule
				AWS *config.AWSConfig
			}{
				DirectorySchedule: dirSchedule,
				AWS:               dirSchedule.AWS,
			}); err != nil {
				log.Error().Err(err).
					Str("name", dirSchedule.Name).
					Str("source", dirSchedule.SourcePath).
					Msg("Failed to execute scheduled directory backup")
			}
		})
		if err != nil {
			log.Error().Err(err).
				Str("name", dirSchedule.Name).
				Str("source", dirSchedule.SourcePath).
				Msg("Failed to schedule directory backup")
			continue
		}

		log.Info().
			Str("name", dirSchedule.Name).
			Str("source", dirSchedule.SourcePath).
			Str("schedule", cronExpr).
			Msg("Scheduled directory backup")
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

func (s *Scheduler) getCronExpression(expr string) string {
	switch expr {
	case "daily":
		return "0 0 * * *" // Run at midnight every day
	case "twice_daily":
		return "0 */12 * * *" // Run every 12 hours
	case "thrice_daily":
		return "0 */8 * * *" // Run every 8 hours
	default:
		if _, err := cron.ParseStandard(expr); err == nil {
			return expr
		}
		return ""
	}
}
