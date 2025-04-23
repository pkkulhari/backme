package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkkulhari/backme/internal/backup"
	"github.com/pkkulhari/backme/internal/config"
	"github.com/pkkulhari/backme/internal/s3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     *config.Config
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Set default config file if not specified
	if cfgFile == "" {
		defaultConfigPath := "/etc/backme/config.yaml"
		if _, err := os.Stat(defaultConfigPath); err == nil {
			cfgFile = defaultConfigPath
		}
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to execute command")
	}
}

var rootCmd = &cobra.Command{
	Use:   "backme",
	Short: "A CLI tool for backing up databases and directories to S3",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database backup commands",
}

var dbBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup a database to S3",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName, err := cmd.Flags().GetString("db-name")
		if err != nil {
			return err
		}

		s3Client, err := s3.New(cfg, nil)
		if err != nil {
			return err
		}

		backupSvc := backup.New(cfg, s3Client)
		dbConfig := &config.DatabaseConfig{
			Name: dbName,
		}
		return backupSvc.BackupDatabase(context.Background(), dbConfig, nil)
	},
}

var dirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Directory backup commands",
}

var dirBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup a directory to S3",
	RunE: func(cmd *cobra.Command, args []string) error {
		source, err := cmd.Flags().GetString("source")
		if err != nil {
			return err
		}

		s3Client, err := s3.New(cfg, nil)
		if err != nil {
			return err
		}

		backupSvc := backup.New(cfg, s3Client)
		sync, _ := cmd.Flags().GetBool("sync")
		delete, _ := cmd.Flags().GetBool("delete")

		return backupSvc.BackupDirectory(context.Background(), source, sync, delete, nil)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/backme/config.yaml)")

	// Add command flags
	dbBackupCmd.Flags().String("db-name", "", "name of the database to backup")
	_ = dbBackupCmd.MarkFlagRequired("db-name")

	dirBackupCmd.Flags().String("source", "", "source directory path")
	dirBackupCmd.Flags().Bool("sync", false, "sync with S3 (only upload new or modified files)")
	dirBackupCmd.Flags().Bool("delete", false, "delete files from S3 that don't exist locally (only works with --sync)")
	_ = dirBackupCmd.MarkFlagRequired("source")

	// Add commands to root
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbBackupCmd)

	rootCmd.AddCommand(dirCmd)
	dirCmd.AddCommand(dirBackupCmd)
}

func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir := "/etc/backme"
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("BACKUP_ME")
	viper.AutomaticEnv()

	cfg = config.New()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
