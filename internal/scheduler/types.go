package scheduler

import (
	"fmt"

	"github.com/pkkulhari/backme/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type ScheduleManager struct {
	cfg *config.Config
}

func NewScheduleManager(cfg *config.Config) *ScheduleManager {
	return &ScheduleManager{cfg: cfg}
}

func (sm *ScheduleManager) AddDatabaseSchedule(schedule config.DatabaseSchedule) error {
	// Check if a schedule with this name already exists
	for i, s := range sm.cfg.Schedules.Databases {
		if s.Name == schedule.Name {
			// Replace existing schedule
			sm.cfg.Schedules.Databases[i] = schedule
			log.Info().Str("name", schedule.Name).Msg("Updated existing database backup schedule")
			viper.Set("schedules.databases", sm.cfg.Schedules.Databases)
			return sm.saveConfig()
		}
	}

	// Add new schedule
	sm.cfg.Schedules.Databases = append(sm.cfg.Schedules.Databases, schedule)
	log.Info().Str("name", schedule.Name).Msg("Added new database backup schedule")
	viper.Set("schedules.databases", sm.cfg.Schedules.Databases)
	return sm.saveConfig()
}

func (sm *ScheduleManager) AddDirectorySchedule(schedule config.DirectorySchedule) error {
	// Check if a schedule with this name already exists
	for i, s := range sm.cfg.Schedules.Directories {
		if s.Name == schedule.Name {
			// Replace existing schedule
			sm.cfg.Schedules.Directories[i] = schedule
			log.Info().Str("name", schedule.Name).Msg("Updated existing directory backup schedule")
			viper.Set("schedules.directories", sm.cfg.Schedules.Directories)
			return sm.saveConfig()
		}
	}

	// Add new schedule
	sm.cfg.Schedules.Directories = append(sm.cfg.Schedules.Directories, schedule)
	log.Info().Str("name", schedule.Name).Msg("Added new directory backup schedule")
	viper.Set("schedules.directories", sm.cfg.Schedules.Directories)
	return sm.saveConfig()
}

func (sm *ScheduleManager) RemoveSchedule(name string, scheduleType string) error {
	var found bool

	switch scheduleType {
	case "database":
		schedules := sm.cfg.Schedules.Databases[:0]
		for _, s := range sm.cfg.Schedules.Databases {
			if s.Name != name {
				schedules = append(schedules, s)
			} else {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("schedule '%s' not found", name)
		}
		sm.cfg.Schedules.Databases = schedules
		viper.Set("schedules.databases", schedules)
		log.Info().Str("name", name).Msg("Removed database backup schedule")

	case "directory":
		schedules := sm.cfg.Schedules.Directories[:0]
		for _, s := range sm.cfg.Schedules.Directories {
			if s.Name != name {
				schedules = append(schedules, s)
			} else {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("schedule '%s' not found", name)
		}
		sm.cfg.Schedules.Directories = schedules
		viper.Set("schedules.directories", schedules)
		log.Info().Str("name", name).Msg("Removed directory backup schedule")

	default:
		return fmt.Errorf("invalid schedule type: %s", scheduleType)
	}

	return sm.saveConfig()
}

// ListDatabaseSchedules returns all database backup schedules
func (sm *ScheduleManager) ListDatabaseSchedules() []config.DatabaseSchedule {
	return sm.cfg.Schedules.Databases
}

// ListDirectorySchedules returns all directory backup schedules
func (sm *ScheduleManager) ListDirectorySchedules() []config.DirectorySchedule {
	return sm.cfg.Schedules.Directories
}

// GetDatabaseSchedule returns a database backup schedule by name
func (sm *ScheduleManager) GetDatabaseSchedule(name string) (config.DatabaseSchedule, bool) {
	for _, s := range sm.cfg.Schedules.Databases {
		if s.Name == name {
			return s, true
		}
	}
	return config.DatabaseSchedule{}, false
}

// GetDirectorySchedule returns a directory backup schedule by name
func (sm *ScheduleManager) GetDirectorySchedule(name string) (config.DirectorySchedule, bool) {
	for _, s := range sm.cfg.Schedules.Directories {
		if s.Name == name {
			return s, true
		}
	}
	return config.DirectorySchedule{}, false
}

func (sm *ScheduleManager) saveConfig() error {
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}
