package config

import (
	"fmt"
	"log"
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
	Host        string   `yaml:"host"`
	Port        int      `yaml:"port"`
	Mode        string   `yaml:"mode"`
	CORSOrigins []string `yaml:"cors_origins"`
	TLSCert     string   `yaml:"tls_cert"`
	TLSKey      string   `yaml:"tls_key"`
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

	// Reject default/insecure JWT secrets
	defaultSecrets := []string{
		"opsmanage-jwt-secret-change-in-production",
		"change-me-to-a-random-string",
		"",
	}
	for _, s := range defaultSecrets {
		if cfg.JWT.Secret == s {
			log.Fatalf("❌ JWT secret 不安全，请修改 config.yaml 中的 jwt.secret（当前为默认值）")
		}
	}

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
			Secret:      "",
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
		&model.SSHAccount{},
		&model.LogEntry{},
		&model.Setting{},
		&model.DockerRegistry{},
		&model.ComposeProject{},
		&model.ComposeTemplate{},
	); err != nil {
		return fmt.Errorf("迁移失败: %w", err)
	}

	DB = db

	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成默认密码失败: %w", err)
		}
		db.Create(&model.User{
			Username: "admin",
			Password: string(hash),
			Nickname: "管理员",
			Role:     "admin",
		})
		log.Println("⚠️  已创建默认管理员账号 admin / admin123，请首次登录后立即修改密码")
	}

	return nil
}

func InitStatic() {
	os.MkdirAll("./static", 0755)
}
