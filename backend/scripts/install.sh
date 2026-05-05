#!/bin/bash
# OpsManage 安装脚本（支持版本选择）
# 用法: install.sh <service> [version] [options]
# 服务: nginx, mysql, postgresql, redis

set -e

SERVICE=$1
VERSION=$2
ROOT_PASS=$3

# 进度输出函数
progress() {
    echo "PROGRESS:$1:$2"
}

log_info() {
    echo "INFO:$1"
}

log_error() {
    echo "ERROR:$1"
}

log_success() {
    echo "SUCCESS:$1"
}

# 检查是否为 root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 权限运行"
        exit 1
    fi
}

# 清理 MySQL 旧仓库问题
clean_mysql_repo() {
    log_info "清理旧的仓库配置..."
    rm -f /etc/apt/sources.list.d/mysql*.list 2>/dev/null || true
    rm -f /etc/apt/trusted.gpg.d/mysql* 2>/dev/null || true
    if command -v apt-key &> /dev/null; then
        apt-key del "A4A9406876FCBD3C456770C88C718D3B5072E1F5" 2>/dev/null || true
    fi
}

# 更新软件源
update_sources() {
    log_info "更新软件源..."
    progress 10 "正在更新软件源..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq 2>&1 | while read -r line; do
        echo "DETAIL:$line"
    done
    log_info "软件源更新完成"
}

# ========== Nginx 安装 ==========
install_nginx() {
    log_info "开始安装 Nginx..."
    
    if command -v nginx &> /dev/null; then
        log_success "Nginx 已安装: $(nginx -v 2>&1)"
        progress 100 "Nginx 已安装"
        exit 0
    fi
    
    progress 5 "准备安装环境..."
    clean_mysql_repo
    update_sources
    
    # 添加 Nginx 官方仓库（获取最新版本）
    progress 25 "配置 Nginx 仓库..."
    log_info "添加 Nginx 官方仓库..."
    
    apt-get install -y -qq curl gnupg2 ca-certificates lsb-release 2>&1 | tail -1
    
    # 导入 Nginx 签名密钥
    curl -fsSL https://nginx.org/keys/nginx_signing.key | gpg --dearmor -o /usr/share/keyrings/nginx-archive-keyring.gpg 2>/dev/null || true
    
    # 添加仓库
    echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] http://nginx.org/packages/ubuntu $(lsb_release -cs) nginx" > /etc/apt/sources.list.d/nginx.list
    
    apt-get update -qq 2>&1 | tail -1
    
    # 安装指定版本或最新版本
    progress 40 "正在安装 Nginx..."
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        log_info "安装 Nginx $VERSION..."
        apt-get install -y -qq nginx=$VERSION* 2>&1 | while read -r line; do
            echo "DETAIL:$line"
        done
    else
        log_info "安装最新版 Nginx..."
        apt-get install -y -qq nginx 2>&1 | while read -r line; do
            echo "DETAIL:$line"
        done
    fi
    
    progress 70 "正在配置 Nginx 服务..."
    systemctl enable nginx 2>/dev/null || true
    systemctl start nginx 2>/dev/null || true
    
    progress 90 "正在验证安装..."
    sleep 1
    
    if systemctl is-active --quiet nginx; then
        NGINX_VER=$(nginx -v 2>&1 | grep -oP '[\d.]+')
        log_success "Nginx 安装成功！版本: $NGINX_VER"
        progress 100 "安装完成"
    else
        log_error "Nginx 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== MySQL 安装 ==========
install_mysql() {
    log_info "开始安装 MySQL..."
    
    if command -v mysql &> /dev/null; then
        log_success "MySQL 已安装: $(mysql --version 2>&1)"
        progress 100 "MySQL 已安装"
        exit 0
    fi
    
    progress 5 "清理旧的仓库配置..."
    clean_mysql_repo
    
    # 添加 MySQL APT 仓库
    progress 15 "配置 MySQL 仓库..."
    log_info "添加 MySQL APT 仓库..."
    
    # 下载 MySQL APT 配置包
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        log_info "准备安装 MySQL $VERSION..."
        # 根据版本选择不同的仓库配置
        case "$VERSION" in
            8.0*)
                MYSQL_APT="mysql-apt-config_0.8.29-1_all.deb"
                ;;
            8.4*)
                MYSQL_APT="mysql-apt-config_0.8.29-1_all.deb"
                ;;
            5.7*)
                MYSQL_APT="mysql-apt-config_0.8.29-1_all.deb"
                ;;
            *)
                MYSQL_APT="mysql-apt-config_0.8.29-1_all.deb"
                ;;
        esac
    else
        MYSQL_APT="mysql-apt-config_0.8.29-1_all.deb"
    fi
    
    # 下载并安装 APT 配置
    cd /tmp
    wget -q "https://dev.mysql.com/get/$MYSQL_APT" -O mysql-apt-config.deb 2>/dev/null || true
    if [ -f mysql-apt-config.deb ]; then
        DEBIAN_FRONTEND=noninteractive dpkg -i mysql-apt-config.deb 2>/dev/null || true
        rm -f mysql-apt-config.deb
    fi
    
    update_sources
    
    # 安装 MySQL
    progress 35 "正在安装 MySQL Server..."
    log_info "安装 MySQL（这可能需要几分钟）..."
    
    export DEBIAN_FRONTEND=noninteractive
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        log_info "安装 MySQL $VERSION..."
        apt-get install -y -qq mysql-server=$VERSION* mysql-client=$VERSION* 2>&1 | while read -r line; do
            echo "DETAIL:$line"
        done
    else
        apt-get install -y -qq mysql-server 2>&1 | while read -r line; do
            echo "DETAIL:$line"
        done
    fi
    
    progress 65 "正在启动 MySQL 服务..."
    systemctl enable mysql 2>/dev/null || true
    systemctl start mysql 2>/dev/null || true
    sleep 2
    
    progress 80 "正在配置安全设置..."
    if [ -n "$ROOT_PASS" ]; then
        log_info "设置 root 密码..."
        mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${ROOT_PASS}'; FLUSH PRIVILEGES;" 2>/dev/null || true
    fi
    
    # 安全配置
    mysql -e "DELETE FROM mysql.user WHERE User='';" 2>/dev/null || true
    mysql -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');" 2>/dev/null || true
    mysql -e "DROP DATABASE IF EXISTS test;" 2>/dev/null || true
    mysql -e "DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';" 2>/dev/null || true
    mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true
    
    progress 95 "正在验证安装..."
    sleep 1
    
    if systemctl is-active --quiet mysql; then
        MYSQL_VER=$(mysql --version 2>&1 | grep -oP 'Ver [\d.]+' | cut -d' ' -f2)
        log_success "MySQL 安装成功！版本: $MYSQL_VER"
        progress 100 "安装完成"
    else
        log_error "MySQL 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== PostgreSQL 安装 ==========
install_postgresql() {
    log_info "开始安装 PostgreSQL..."
    
    if command -v psql &> /dev/null; then
        log_success "PostgreSQL 已安装: $(psql --version 2>&1)"
        progress 100 "PostgreSQL 已安装"
        exit 0
    fi
    
    progress 5 "准备安装环境..."
    clean_mysql_repo
    
    # 添加 PostgreSQL 官方仓库
    progress 15 "配置 PostgreSQL 仓库..."
    log_info "添加 PostgreSQL 官方仓库..."
    
    # 导入 PostgreSQL 签名密钥
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql-archive-keyring.gpg 2>/dev/null || true
    
    # 添加仓库
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        PG_MAJOR=$(echo "$VERSION" | cut -d. -f1)
        log_info "准备安装 PostgreSQL $VERSION..."
    else
        PG_MAJOR="16"  # 默认最新稳定版
    fi
    
    echo "deb [signed-by=/usr/share/keyrings/postgresql-archive-keyring.gpg] http://apt.postgresql.org/pub/repos/ubuntu $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list
    
    update_sources
    
    progress 35 "正在安装 PostgreSQL..."
    log_info "安装 PostgreSQL $PG_MAJOR..."
    
    apt-get install -y -qq postgresql-$PG_MAJOR postgresql-contrib-$PG_MAJOR 2>&1 | while read -r line; do
        echo "DETAIL:$line"
    done
    
    progress 70 "正在配置 PostgreSQL..."
    systemctl enable postgresql 2>/dev/null || true
    systemctl start postgresql 2>/dev/null || true
    sleep 2
    
    progress 95 "正在验证安装..."
    
    if systemctl is-active --quiet postgresql; then
        PG_VER=$(psql --version 2>&1 | grep -oP '[\d.]+' | head -1)
        log_success "PostgreSQL 安装成功！版本: $PG_VER"
        progress 100 "安装完成"
    else
        log_error "PostgreSQL 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== Redis 安装 ==========
install_redis() {
    log_info "开始安装 Redis..."
    
    if command -v redis-server &> /dev/null; then
        log_success "Redis 已安装: $(redis-server --version 2>&1)"
        progress 100 "Redis 已安装"
        exit 0
    fi
    
    progress 5 "准备安装环境..."
    clean_mysql_repo
    
    # 添加 Redis 官方仓库
    progress 15 "配置 Redis 仓库..."
    log_info "添加 Redis 官方仓库..."
    
    # 导入 Redis 签名密钥
    curl -fsSL https://packages.redis.io/gpg | gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg 2>/dev/null || true
    
    # 添加仓库
    echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" > /etc/apt/sources.list.d/redis.list
    
    update_sources
    
    progress 40 "正在安装 Redis..."
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        log_info "安装 Redis $VERSION..."
        apt-get install -y -qq redis-server=$VERSION* 2>&1 | while read -r line; do
            echo "DETAIL:$line"
        done
    else
        log_info "安装最新版 Redis..."
        apt-get install -y -qq redis-server 2>&1 | while read -r line; do
            echo "DETAIL:$line"
        done
    fi
    
    progress 75 "正在配置 Redis..."
    systemctl enable redis-server 2>/dev/null || true
    systemctl start redis-server 2>/dev/null || true
    sleep 1
    
    progress 95 "正在验证安装..."
    
    if systemctl is-active --quiet redis-server; then
        REDIS_VER=$(redis-server --version 2>&1 | grep -oP 'v=[\d.]+' | cut -d= -f2)
        log_success "Redis 安装成功！版本: $REDIS_VER"
        progress 100 "安装完成"
    else
        log_error "Redis 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== 获取可用版本 ==========
get_available_versions() {
    case "$SERVICE" in
        nginx)
            echo "=== Nginx 可用版本 ==="
            apt-cache madison nginx 2>/dev/null | awk '{print $3}' | head -10
            ;;
        mysql)
            echo "=== MySQL 可用版本 ==="
            apt-cache madison mysql-server 2>/dev/null | awk '{print $3}' | head -10
            ;;
        postgresql)
            echo "=== PostgreSQL 可用版本 ==="
            apt-cache madison postgresql 2>/dev/null | awk '{print $3}' | head -10
            ;;
        redis)
            echo "=== Redis 可用版本 ==="
            apt-cache madison redis-server 2>/dev/null | awk '{print $3}' | head -10
            ;;
        *)
            log_error "未知的服务: $SERVICE"
            ;;
    esac
}

# 主程序
check_root

# 如果第二个参数是 "versions"，则显示可用版本
if [ "$VERSION" = "versions" ]; then
    get_available_versions
    exit 0
fi

case "$SERVICE" in
    nginx)
        install_nginx
        ;;
    mysql)
        install_mysql
        ;;
    postgresql|postgres)
        install_postgresql
        ;;
    redis)
        install_redis
        ;;
    *)
        log_error "未知的服务: $SERVICE"
        log_info "支持的服务: nginx, mysql, postgresql, redis"
        exit 1
        ;;
esac
