#!/bin/bash
# OpsManage 安装脚本
# 用法: install.sh <service> [options]
# 服务: nginx, mysql, postgresql, redis

set -e

SERVICE=$1
ROOT_PASS=$2

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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
    log_info "清理旧的 MySQL 仓库配置..."
    rm -f /etc/apt/sources.list.d/mysql*.list 2>/dev/null || true
    rm -f /etc/apt/trusted.gpg.d/mysql* 2>/dev/null || true
    
    # 删除过期的 GPG 密钥
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
    
    # 检查是否已安装
    if command -v nginx &> /dev/null; then
        log_success "Nginx 已安装: $(nginx -v 2>&1)"
        progress 100 "Nginx 已安装"
        exit 0
    fi
    
    # 更新软件源
    progress 5 "准备安装环境..."
    clean_mysql_repo
    update_sources
    
    # 安装 Nginx
    progress 40 "正在安装 Nginx..."
    log_info "安装 Nginx 软件包..."
    
    apt-get install -y -qq nginx 2>&1 | while read -r line; do
        echo "DETAIL:$line"
    done
    
    # 配置服务
    progress 70 "正在配置 Nginx 服务..."
    log_info "启用并启动 Nginx..."
    
    systemctl enable nginx 2>/dev/null || true
    systemctl start nginx 2>/dev/null || true
    
    # 验证安装
    progress 90 "正在验证安装..."
    sleep 1
    
    if systemctl is-active --quiet nginx; then
        VERSION=$(nginx -v 2>&1 | grep -oP '[\d.]+')
        log_success "Nginx 安装成功！版本: $VERSION"
        progress 100 "安装完成"
    else
        log_error "Nginx 安装完成但未运行"
        log_info "尝试手动启动: systemctl start nginx"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== MySQL 安装 ==========
install_mysql() {
    log_info "开始安装 MySQL..."
    
    # 检查是否已安装
    if command -v mysql &> /dev/null; then
        log_success "MySQL 已安装: $(mysql --version 2>&1)"
        progress 100 "MySQL 已安装"
        exit 0
    fi
    
    # 清理旧仓库
    progress 5 "清理旧的仓库配置..."
    clean_mysql_repo
    update_sources
    
    # 安装 MySQL
    progress 30 "正在安装 MySQL Server..."
    log_info "安装 MySQL 软件包（这可能需要几分钟）..."
    
    export DEBIAN_FRONTEND=noninteractive
    apt-get install -y -qq mysql-server 2>&1 | while read -r line; do
        echo "DETAIL:$line"
    done
    
    # 启动服务
    progress 60 "正在启动 MySQL 服务..."
    log_info "启用并启动 MySQL..."
    
    systemctl enable mysql 2>/dev/null || true
    systemctl start mysql 2>/dev/null || true
    sleep 2
    
    # 配置安全设置
    progress 75 "正在配置安全设置..."
    log_info "配置 MySQL 安全设置..."
    
    # 设置 root 密码（如果提供）
    if [ -n "$ROOT_PASS" ]; then
        log_info "设置 root 密码..."
        mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${ROOT_PASS}'; FLUSH PRIVILEGES;" 2>/dev/null || true
    fi
    
    # 运行安全脚本
    mysql -e "DELETE FROM mysql.user WHERE User='';" 2>/dev/null || true
    mysql -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');" 2>/dev/null || true
    mysql -e "DROP DATABASE IF EXISTS test;" 2>/dev/null || true
    mysql -e "DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';" 2>/dev/null || true
    mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true
    
    # 验证安装
    progress 95 "正在验证安装..."
    sleep 1
    
    if systemctl is-active --quiet mysql; then
        VERSION=$(mysql --version 2>&1 | grep -oP 'Ver [\d.]+' | cut -d' ' -f2)
        log_success "MySQL 安装成功！版本: $VERSION"
        progress 100 "安装完成"
    else
        log_error "MySQL 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== PostgreSQL 安装 ==========
install_postgresql() {
    log_info "开始安装 PostgreSQL..."
    
    # 检查是否已安装
    if command -v psql &> /dev/null; then
        log_success "PostgreSQL 已安装: $(psql --version 2>&1)"
        progress 100 "PostgreSQL 已安装"
        exit 0
    fi
    
    progress 5 "准备安装环境..."
    clean_mysql_repo
    update_sources
    
    # 安装 PostgreSQL
    progress 35 "正在安装 PostgreSQL..."
    log_info "安装 PostgreSQL 软件包..."
    
    apt-get install -y -qq postgresql postgresql-contrib 2>&1 | while read -r line; do
        echo "DETAIL:$line"
    done
    
    # 配置服务
    progress 70 "正在配置 PostgreSQL..."
    log_info "启用并启动 PostgreSQL..."
    
    systemctl enable postgresql 2>/dev/null || true
    systemctl start postgresql 2>/dev/null || true
    sleep 2
    
    # 验证安装
    progress 95 "正在验证安装..."
    
    if systemctl is-active --quiet postgresql; then
        VERSION=$(psql --version 2>&1 | grep -oP '[\d.]+' | head -1)
        log_success "PostgreSQL 安装成功！版本: $VERSION"
        progress 100 "安装完成"
    else
        log_error "PostgreSQL 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== Redis 安装 ==========
install_redis() {
    log_info "开始安装 Redis..."
    
    # 检查是否已安装
    if command -v redis-server &> /dev/null; then
        log_success "Redis 已安装: $(redis-server --version 2>&1)"
        progress 100 "Redis 已安装"
        exit 0
    fi
    
    progress 5 "准备安装环境..."
    clean_mysql_repo
    update_sources
    
    # 安装 Redis
    progress 40 "正在安装 Redis..."
    log_info "安装 Redis 软件包..."
    
    apt-get install -y -qq redis-server 2>&1 | while read -r line; do
        echo "DETAIL:$line"
    done
    
    # 配置服务
    progress 75 "正在配置 Redis..."
    log_info "启用并启动 Redis..."
    
    systemctl enable redis-server 2>/dev/null || true
    systemctl start redis-server 2>/dev/null || true
    sleep 1
    
    # 验证安装
    progress 95 "正在验证安装..."
    
    if systemctl is-active --quiet redis-server; then
        VERSION=$(redis-server --version 2>&1 | grep -oP 'v=[\d.]+' | cut -d= -f2)
        log_success "Redis 安装成功！版本: $VERSION"
        progress 100 "安装完成"
    else
        log_error "Redis 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# 主程序
check_root

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
