package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gopkg.in/yaml.v3"
	"opsmanage/internal/model"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Panel    PanelConfig    `yaml:"panel"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

type PanelConfig struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

var DB *gorm.DB
var AppConfig *Config

func Load() *Config {
	cfg := &Config{}
	cfgPath := "config.yaml"
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		cfg = defaultConfig()
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			cfg = defaultConfig()
		}
	}
	AppConfig = cfg
	return cfg
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9090,
			Mode: "debug",
		},
		Database: DatabaseConfig{
			Path: "./data/opsmanage.db",
		},
		JWT: JWTConfig{
			Secret:      "opsmanage-jwt-secret-change-in-production",
			ExpireHours: 168,
		},
		Panel: PanelConfig{
			Title:   "OpsManage",
			Version: "1.0.0",
		},
	}
}

func InitDB(cfg *Config) error {
	os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755)
	db, err := gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Website{},
		&model.Database{},
		&model.Container{},
		&model.Task{},
		&model.SecurityRule{},
		&model.LogEntry{},
		&model.Setting{},
	); err != nil {
		return fmt.Errorf("迁移失败: %w", err)
	}

	DB = db

	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		db.Create(&model.User{
			Username: "admin",
			Password: string(hash),
			Nickname: "管理员",
			Role:     "admin",
		})
	}

	return nil
}

func InitStatic() {
	os.MkdirAll("./static", 0755)
}
