# OpsManage

轻量级服务器运维管理面板，基于 Go + Vue 3 构建。

类似宝塔面板 / 1Panel，提供 Web 界面管理服务器常见运维任务：网站、数据库、Docker 容器、文件、定时任务、安全规则、日志等。

## 功能特性

| 模块 | 功能 |
|------|------|
| 仪表盘 | 系统信息、CPU/内存/磁盘/负载监控 |
| 网站管理 | CRUD + Nginx 配置自动生成 + SSL/WAF |
| 数据库管理 | MySQL / PostgreSQL / Redis 实例管理 |
| 容器管理 | Docker 容器启停重启、日志查看、镜像拉取 |
| 文件管理器 | 在线浏览、编辑、上传、下载、重命名、复制、删除 |
| 计划任务 | Cron 定时执行 + 手动触发 |
| 安全规则 | IP / URL / UA 黑白名单 |
| 日志查看 | 应用日志（可筛选）+ 系统日志 |
| 系统设置 | 通用键值配置 |

## 技术栈

**后端：** Go · Gin · GORM · SQLite · JWT · gorilla/websocket · robfig/cron

**前端：** Vue 3 · TypeScript · Vite · Element Plus · Pinia · Axios

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+

### 克隆并运行

```bash
git clone https://github.com/<your-username>/OpsManage.git
cd OpsManage
```

**后端：**

```bash
cd backend
cp config.yaml.example config.yaml   # 按需修改配置
go build -o opsmanage .
./opsmanage
```

服务默认监听 `http://0.0.0.0:9090`

**前端（开发模式）：**

```bash
cd frontend
npm install
npm run dev
```

开发服务器默认 `http://localhost:5173`，API 请求代理到 `localhost:9090`。

**生产部署：**

```bash
cd frontend
npm run build          # 构建产物输出到 backend/static/
cd ../backend
./opsmanage            # Go 服务同时提供 API 和静态文件
```

### 默认账号

- 用户名：`admin`
- 密码：`admin123`

> 首次登录后请立即修改密码。

## 项目结构

```
OpsManage/
├── backend/
│   ├── main.go                    # 入口
│   ├── config.yaml.example        # 配置模板
│   └── internal/
│       ├── config/                # 配置加载 + 数据库初始化
│       ├── handler/               # API 处理器
│       ├── middleware/             # JWT / CORS 中间件
│       ├── model/                 # GORM 数据模型
│       ├── router/                # 路由定义
│       └── scheduler/             # Cron 定时任务调度器
├── frontend/
│   ├── src/
│   │   ├── api/                   # API 调用封装
│   │   ├── components/            # 公共组件
│   │   ├── stores/                # Pinia 状态管理
│   │   └── views/                 # 页面视图
│   └── vite.config.ts
├── LICENSE
└── README.md
```

## 配置说明

编辑 `backend/config.yaml`：

```yaml
server:
  host: "0.0.0.0"      # 监听地址
  port: 9090            # 监听端口
  mode: debug           # debug / release

database:
  path: "./data/opsmanage.db"   # SQLite 数据库路径

jwt:
  secret: "your-secret-key"    # JWT 密钥（请修改）
  expire_hours: 168             # Token 有效期（小时）

panel:
  title: "OpsManage"            # 面板标题
  version: "1.0.0"
```

## 开源协议

[MIT License](LICENSE)
