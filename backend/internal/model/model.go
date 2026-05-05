package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password  string         `gorm:"size:128;not null" json:"-"`
	Nickname  string         `gorm:"size:64" json:"nickname"`
	Email     string         `gorm:"size:128" json:"email"`
	Role      string         `gorm:"size:32;default:admin" json:"role"`
	LastLogin time.Time      `json:"last_login"`
	IP        string         `gorm:"size:64" json:"ip"`
}

type Website struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Domain      string         `gorm:"size:256;not null" json:"domain"`
	Path        string         `gorm:"size:512;not null" json:"path"`
	Port        int            `gorm:"default:80" json:"port"`
	Status      string         `gorm:"size:16;default:running" json:"status"`
	SSLEnabled  bool           `gorm:"default:false" json:"ssl_enabled"`
	SSLCertPath string         `gorm:"size:512" json:"ssl_cert_path"`
	SSLKeyPath  string         `gorm:"size:512" json:"ssl_key_path"`
	WAFEnabled  bool           `gorm:"default:false" json:"waf_enabled"`
	WAFRules    string         `gorm:"type:text" json:"waf_rules"`
	NginxConf   string         `gorm:"type:text" json:"nginx_conf"`
	Remark      string         `gorm:"size:256" json:"remark"`
}

type Database struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Type      string         `gorm:"size:32;not null" json:"type"`
	Host      string         `gorm:"size:128;default:127.0.0.1" json:"host"`
	Port      int            `json:"port"`
	Username  string         `gorm:"size:64" json:"username"`
	Password  string         `gorm:"size:128" json:"-"`
	Status    string         `gorm:"size:16;default:running" json:"status"`
	Version   string         `gorm:"size:32" json:"version"`
}

type Container struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Image     string         `gorm:"size:256;not null" json:"image"`
	Status    string         `gorm:"size:32;default:stopped" json:"status"`
	Ports     string         `gorm:"size:256" json:"ports"`
	Volumes   string         `gorm:"type:text" json:"volumes"`
	Env       string         `gorm:"type:text" json:"env"`
	NetworkID string         `gorm:"size:128" json:"network_id"`
	ContainerID string       `gorm:"size:128" json:"container_id"`
}

type Task struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Name       string         `gorm:"size:128;not null" json:"name"`
	Command    string         `gorm:"type:text;not null" json:"command"`
	CronExpr   string         `gorm:"size:64" json:"cron_expr"`
	Status     string         `gorm:"size:16;default:enabled" json:"status"`
	LastRun    time.Time      `json:"last_run"`
	LastResult string         `gorm:"size:16" json:"last_result"`
	LastOutput string         `gorm:"type:text" json:"last_output"`
	NextRun    time.Time      `json:"next_run"`
}

type SecurityRule struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Type      string         `gorm:"size:32;not null" json:"type"`
	Content   string         `gorm:"type:text" json:"content"`
	Status    string         `gorm:"size:16;default:enabled" json:"status"`
	Priority  int            `gorm:"default:0" json:"priority"`
	Remark    string         `gorm:"size:256" json:"remark"`
}

type LogEntry struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
	Level     string    `gorm:"size:16;not null" json:"level"`
	Source    string    `gorm:"size:64;not null" json:"source"`
	Message   string    `gorm:"type:text" json:"message"`
	Detail    string    `gorm:"type:text" json:"detail"`
}

type SSHAccount struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Host        string         `gorm:"size:256;not null" json:"host"`
	Port        int            `gorm:"default:22" json:"port"`
	Username    string         `gorm:"size:64;not null" json:"username"`
	Password    string         `gorm:"size:256" json:"-"`
	AuthMethod  string         `gorm:"size:16;default:password" json:"auth_method"` // password / key
	PrivateKey  string         `gorm:"type:text" json:"-"`
	PublicKey   string         `gorm:"type:text" json:"-"`
	Status      string         `gorm:"size:16;default:active" json:"status"`
	Description string         `gorm:"size:512" json:"description"`
}

type Setting struct {
	ID    uint   `gorm:"primarykey" json:"id"`
	Key   string `gorm:"uniqueIndex;size:128;not null" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

// NginxSite Nginx站点配置
type NginxSite struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Name          string         `gorm:"size:128;not null" json:"name"`
	Domain        string         `gorm:"size:256;not null" json:"domain"`
	Root          string         `gorm:"size:512;not null" json:"root"`
	Port          int            `gorm:"default:80" json:"port"`
	SSL           bool           `gorm:"default:false" json:"ssl"`
	SSLCert       string         `gorm:"size:512" json:"ssl_cert"`
	SSLKey        string         `gorm:"size:512" json:"ssl_key"`
	ProxyPass     string         `gorm:"size:512" json:"proxy_pass"`
	ProxyType     string         `gorm:"size:32;default:static" json:"proxy_type"`  // static / proxy
	PHPVersion    string         `gorm:"size:16" json:"php_version"`
	Gzip          bool           `gorm:"default:true" json:"gzip"`
	CacheEnabled  bool           `gorm:"default:false" json:"cache_enabled"`
	CacheTime     int            `gorm:"default:7" json:"cache_time"`
	RateLimit     int            `gorm:"default:0" json:"rate_limit"`
	Status        string         `gorm:"size:16;default:running" json:"status"`
	ConfigFile    string         `gorm:"size:512" json:"config_file"`
	ConfigContent string         `gorm:"type:text" json:"config_content"`
	Remark        string         `gorm:"size:256" json:"remark"`
}

// DBInstance 数据库服务实例（已安装的数据库服务）
type DBInstance struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Name       string         `gorm:"size:128;not null" json:"name"`
	Type       string         `gorm:"size:32;not null" json:"type"` // mysql / postgresql / redis
	Version    string         `gorm:"size:32" json:"version"`
	InstallWay string         `gorm:"size:16;default:apt" json:"install_way"`
	Host       string         `gorm:"size:128;default:127.0.0.1" json:"host"`
	Port       int            `json:"port"`
	RootPass   string         `gorm:"size:128" json:"-"`
	Status     string         `gorm:"size:16;default:stopped" json:"status"`
	ConfigPath string         `gorm:"size:512" json:"config_path"`
	DataPath   string         `gorm:"size:512" json:"data_path"`
	Remark     string         `gorm:"size:256" json:"remark"`
}

// DBDatabase 数据库中的具体 database/schema
type DBDatabase struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	InstanceID uint           `gorm:"index;not null" json:"instance_id"`
	Name       string         `gorm:"size:128;not null" json:"name"`
	Charset    string         `gorm:"size:32;default:utf8mb4" json:"charset"`
	Collation  string         `gorm:"size:64" json:"collation"`
	Size       int64          `gorm:"default:0" json:"size"`
	Remark     string         `gorm:"size:256" json:"remark"`
}

// DBUser 数据库用户
type DBUser struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	InstanceID uint           `gorm:"index;not null" json:"instance_id"`
	Username   string         `gorm:"size:64;not null" json:"username"`
	Password   string         `gorm:"size:128" json:"-"`
	Host       string         `gorm:"size:128;default:%" json:"host"`
	Privileges string         `gorm:"size:256" json:"privileges"`
	DBName     string         `gorm:"size:128" json:"db_name"`
}

// DBBackup 数据库备份记录
type DBBackup struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	InstanceID uint      `gorm:"index;not null" json:"instance_id"`
	DBName     string    `gorm:"size:128" json:"db_name"`
	FilePath   string    `gorm:"size:512" json:"file_path"`
	Size       int64     `gorm:"default:0" json:"size"`
	Status     string    `gorm:"size:16;default:success" json:"status"`
}

// DockerRegistry 镜像仓库
type DockerRegistry struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	URL       string         `gorm:"size:256;not null" json:"url"`
	Username  string         `gorm:"size:128" json:"username"`
	Password  string         `gorm:"size:256" json:"-"`
	IsDefault bool           `gorm:"default:false" json:"is_default"`
}

// ComposeProject 编排项目
type ComposeProject struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Path      string         `gorm:"size:512;not null" json:"path"`
	Status    string         `gorm:"size:32;default:stopped" json:"status"`
	Services  int            `gorm:"default:0" json:"services"`
}

// ComposeTemplate 编排模板
type ComposeTemplate struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Description string         `gorm:"size:512" json:"description"`
	Content     string         `gorm:"type:text" json:"content"`
}

// Certificate SSL 证书
type Certificate struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"uniqueIndex;size:255" json:"name"`       // 证书名称
	Domain    string         `gorm:"size:255" json:"domain"`                 // 域名
	Type      string         `gorm:"size:20" json:"type"`                    // letsencrypt / custom
	CertPath  string         `gorm:"size:500" json:"cert_path"`              // 证书文件路径
	KeyPath   string         `gorm:"size:500" json:"key_path"`               // 私钥文件路径
	ChainPath string         `gorm:"size:500" json:"chain_path"`             // 证书链路径
	Issuer    string         `gorm:"size:255" json:"issuer"`                 // 颁发者
	NotBefore time.Time      `json:"not_before"`                             // 生效时间
	NotAfter  time.Time      `json:"not_after"`                              // 过期时间
	Subject   string         `gorm:"size:500" json:"subject"`                // 主题
	SANs      string         `gorm:"type:text" json:"sans"`                  // 备用域名
	Status    string         `gorm:"size:20;default:valid" json:"status"`    // valid / expired / about_to_expire
}
