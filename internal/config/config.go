package config

type Config struct {
	Database  DatabaseConfig `mapstructure:"database"`
	AWS       AWSConfig      `mapstructure:"aws"`
	Schedules Schedules      `mapstructure:"schedules"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

type AWSConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	DatabasePrefix  string `mapstructure:"database_prefix"`
	DirectoryPrefix string `mapstructure:"directory_prefix"`
}

type Schedules struct {
	Databases   []DatabaseSchedule  `mapstructure:"databases"`
	Directories []DirectorySchedule `mapstructure:"directories"`
}

type DatabaseSchedule struct {
	Name       string         `mapstructure:"name"`
	Expression string         `mapstructure:"expression"`
	Database   DatabaseConfig `mapstructure:"database"`
	AWS        *AWSConfig     `mapstructure:"aws,omitempty"`
}

type DirectorySchedule struct {
	Name       string     `mapstructure:"name"`
	Expression string     `mapstructure:"expression"`
	SourcePath string     `mapstructure:"source_path"`
	Sync       bool       `mapstructure:"sync"`
	Delete     bool       `mapstructure:"delete"`
	AWS        *AWSConfig `mapstructure:"aws,omitempty"`
}

func New() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host: "localhost",
			Port: 5432,
		},
		AWS: AWSConfig{},
		Schedules: Schedules{
			Databases:   []DatabaseSchedule{},
			Directories: []DirectorySchedule{},
		},
	}
}
