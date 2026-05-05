#!/bin/bash
# OpsManage 安装脚本（支持版本选择）
# 用法: install.sh <service> [version] [root_pass]
# 服务: nginx, mysql, postgresql, redis
# 版本: 具体版本号 或 "latest"
# 示例: install.sh nginx 1.24.0
#       install.sh mysql latest mypassword
#       install.sh nginx versions  # 列出可用版本

set -e

SERVICE="${1:-}"
VERSION="${2:-latest}"
ROOT_PASS="${3:-}"

# ========== 输出函数 ==========
progress() {
    echo "PROGRESS:${1}:${2}"
}

log_info() {
    echo "INFO:${1}"
}

log_error() {
    echo "ERROR:${1}"
}

log_success() {
    echo "SUCCESS:${1}"
}

log_detail() {
    echo "DETAIL:${1}"
}

# ========== 工具函数 ==========

# 检查是否为 root
check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        log_error "请使用 root 权限运行此脚本"
        exit 1
    fi
}

# 检查服务是否已安装
is_installed() {
    command -v "$1" &>/dev/null
}

# 检查服务是否运行
is_running() {
    systemctl is-active --quiet "$1" 2>/dev/null
}

# 获取系统代号
get_codename() {
    lsb_release -cs 2>/dev/null || echo "noble"
}

# 安装基础依赖
install_deps() {
    log_info "安装基础依赖..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get install -y -qq curl wget gnupg2 ca-certificates lsb-release apt-transport-https 2>&1 | tail -1
}

# 清理旧的仓库配置
clean_repo() {
    local pattern="$1"
    log_info "清理 ${pattern} 旧仓库配置..."
    rm -f /etc/apt/sources.list.d/${pattern}* 2>/dev/null || true
    rm -f /etc/apt/trusted.gpg.d/${pattern}* 2>/dev/null || true
}

# 更新软件源
update_sources() {
    log_info "更新软件源..."
    progress 10 "正在更新软件源..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq 2>&1 | while IFS= read -r line; do
        # 只输出重要信息
        if echo "$line" | grep -qE "(Err|Fetched|Ign)"; then
            log_detail "$line"
        fi
    done
    log_info "软件源更新完成"
}

# ========== Nginx 安装 ==========
install_nginx() {
    log_info "开始安装 Nginx..."

    if is_installed nginx; then
        local ver
        ver=$(nginx -v 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        log_success "Nginx 已安装: ${ver}"
        progress 100 "Nginx 已安装"
        exit 0
    fi

    progress 5 "准备安装环境..."
    clean_repo "nginx"
    install_deps

    # 添加 Nginx 官方仓库
    progress 20 "配置 Nginx 仓库..."
    log_info "添加 Nginx 官方仓库..."

    local codename
    codename=$(get_codename)

    # 导入签名密钥
    curl -fsSL https://nginx.org/keys/nginx_signing.key | gpg --dearmor --yes -o /usr/share/keyrings/nginx-archive-keyring.gpg 2>/dev/null || true

    # 检查密钥是否导入成功
    if [ ! -f /usr/share/keyrings/nginx-archive-keyring.gpg ]; then
        log_error "导入 Nginx 签名密钥失败"
        # 尝试备用方式
        wget -qO- https://nginx.org/keys/nginx_signing.key | apt-key add - 2>/dev/null || true
    fi

    # 添加仓库
    echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] http://nginx.org/packages/ubuntu ${codename} nginx" > /etc/apt/sources.list.d/nginx.list

    update_sources

    # 安装 Nginx
    progress 40 "正在安装 Nginx..."
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        log_info "安装 Nginx ${VERSION}..."
        # 查询可用版本
        local avail_ver
        avail_ver=$(apt-cache madison nginx | grep "$VERSION" | head -1 | awk '{print $3}')
        if [ -n "$avail_ver" ]; then
            apt-get install -y -qq "nginx=${avail_ver}" 2>&1 | while IFS= read -r line; do
                log_detail "$line"
            done
        else
            log_error "未找到 Nginx ${VERSION}，安装最新版本"
            apt-get install -y -qq nginx 2>&1 | while IFS= read -r line; do
                log_detail "$line"
            done
        fi
    else
        log_info "安装最新版 Nginx..."
        apt-get install -y -qq nginx 2>&1 | while IFS= read -r line; do
            log_detail "$line"
        done
    fi

    progress 70 "正在配置 Nginx 服务..."
    systemctl enable nginx 2>/dev/null || true

    # 检查端口占用并启动
    if ss -tlnp 2>/dev/null | grep -q ':80 ' || netstat -tlnp 2>/dev/null | grep -q ':80 '; then
        local port_holder
        port_holder=$(ss -tlnp 2>/dev/null | grep ':80 ' | grep -oP 'users:\(\("([^"]+)' | head -1 | cut -d'"' -f2)
        log_info "端口 80 已被 ${port_holder:-其他服务} 占用，修改 Nginx 监听端口为 8080..."
        # 修改所有配置文件中的 80 端口
        find /etc/nginx -name "*.conf" -exec sed -i 's/listen\s\+80;/listen 8080;/g' {} \; 2>/dev/null || true
        find /etc/nginx -name "*.conf" -exec sed -i 's/listen\s\+\[::\]:80;/listen [::]:8080;/g' {} \; 2>/dev/null || true
        log_info "Nginx 将监听 8080 端口"
    fi

    systemctl start nginx 2>/dev/null || true

    progress 90 "正在验证安装..."
    sleep 1

    if is_running nginx; then
        local ver
        ver=$(nginx -v 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        log_success "Nginx 安装成功！版本: ${ver}"
        progress 100 "安装完成"
    else
        log_error "Nginx 安装完成但未运行，尝试手动启动: systemctl start nginx"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== MySQL 安装 ==========
install_mysql() {
    log_info "开始安装 MySQL..."

    if is_installed mysql; then
        local ver
        ver=$(mysql --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        log_success "MySQL 已安装: ${ver}"
        progress 100 "MySQL 已安装"
        exit 0
    fi

    progress 5 "清理旧的仓库配置..."
    clean_repo "mysql"

    progress 15 "配置 MySQL 仓库..."
    log_info "配置 MySQL APT 仓库..."

    # 下载并安装 MySQL APT 配置包
    cd /tmp
    local apt_config_url="https://dev.mysql.com/get/mysql-apt-config_0.8.29-1_all.deb"
    if wget -q "$apt_config_url" -O mysql-apt-config.deb 2>/dev/null; then
        export DEBIAN_FRONTEND=noninteractive
        dpkg -i mysql-apt-config.deb 2>/dev/null || true
        rm -f mysql-apt-config.deb
    else
        log_info "跳过 MySQL APT 配置，使用系统仓库"
    fi

    update_sources

    # 安装 MySQL
    progress 35 "正在安装 MySQL Server..."
    log_info "安装 MySQL（这可能需要几分钟）..."

    export DEBIAN_FRONTEND=noninteractive
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        log_info "尝试安装 MySQL ${VERSION}..."
        # 查询可用版本
        local avail_ver
        avail_ver=$(apt-cache madison mysql-server | grep "$VERSION" | head -1 | awk '{print $3}')
        if [ -n "$avail_ver" ]; then
            apt-get install -y -qq "mysql-server=${avail_ver}" 2>&1 | while IFS= read -r line; do
                log_detail "$line"
            done
        else
            log_error "未找到 MySQL ${VERSION}，安装系统默认版本"
            apt-get install -y -qq mysql-server 2>&1 | while IFS= read -r line; do
                log_detail "$line"
            done
        fi
    else
        apt-get install -y -qq mysql-server 2>&1 | while IFS= read -r line; do
            log_detail "$line"
        done
    fi

    progress 65 "正在启动 MySQL 服务..."
    systemctl enable mysql 2>/dev/null || true
    systemctl start mysql 2>/dev/null || true
    sleep 3

    progress 80 "正在配置安全设置..."
    if [ -n "$ROOT_PASS" ]; then
        log_info "设置 root 密码..."
        mysql -u root -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${ROOT_PASS}'; FLUSH PRIVILEGES;" 2>/dev/null || \
        mysql -u root --skip-password -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${ROOT_PASS}'; FLUSH PRIVILEGES;" 2>/dev/null || \
        log_info "密码设置跳过（可能已设置）"
    fi

    # 安全配置（忽略错误）
    mysql -u root -e "DELETE FROM mysql.user WHERE User='';" 2>/dev/null || true
    mysql -u root -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');" 2>/dev/null || true
    mysql -u root -e "DROP DATABASE IF EXISTS test;" 2>/dev/null || true
    mysql -u root -e "DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';" 2>/dev/null || true
    mysql -u root -e "FLUSH PRIVILEGES;" 2>/dev/null || true

    progress 95 "正在验证安装..."
    sleep 1

    if is_running mysql; then
        local ver
        ver=$(mysql --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        log_success "MySQL 安装成功！版本: ${ver}"
        progress 100 "安装完成"
    else
        log_error "MySQL 安装完成但未运行，尝试手动启动: systemctl start mysql"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== PostgreSQL 安装 ==========
install_postgresql() {
    log_info "开始安装 PostgreSQL..."

    if is_installed psql; then
        local ver
        ver=$(psql --version 2>&1 | grep -oE '[0-9]+\.[0-9]+' | head -1)
        log_success "PostgreSQL 已安装: ${ver}"
        progress 100 "PostgreSQL 已安装"
        exit 0
    fi

    progress 5 "准备安装环境..."
    clean_repo "postgresql"
    install_deps

    progress 15 "配置 PostgreSQL 仓库..."
    log_info "添加 PostgreSQL 官方仓库..."

    local codename
    codename=$(get_codename)

    # 导入签名密钥
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor --yes -o /usr/share/keyrings/postgresql-archive-keyring.gpg 2>/dev/null || true

    # 确定主版本号
    local pg_major="16"
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        pg_major=$(echo "$VERSION" | cut -d. -f1)
        log_info "准备安装 PostgreSQL ${pg_major}..."
    fi

    # 添加仓库
    echo "deb [signed-by=/usr/share/keyrings/postgresql-archive-keyring.gpg] http://apt.postgresql.org/pub/repos/ubuntu ${codename}-pgdg main" > /etc/apt/sources.list.d/pgdg.list

    update_sources

    progress 35 "正在安装 PostgreSQL..."
    log_info "安装 PostgreSQL ${pg_major}..."

    apt-get install -y -qq "postgresql-${pg_major}" "postgresql-contrib-${pg_major}" 2>&1 | while IFS= read -r line; do
        log_detail "$line"
    done

    progress 70 "正在配置 PostgreSQL..."
    # PostgreSQL 服务名格式: postgresql@版本号-main
    local pg_service="postgresql"
    if systemctl list-unit-files | grep -q "postgresql@${pg_major}"; then
        pg_service="postgresql@${pg_major}-main"
    fi

    systemctl enable "${pg_service}" 2>/dev/null || systemctl enable postgresql 2>/dev/null || true
    systemctl start "${pg_service}" 2>/dev/null || systemctl start postgresql 2>/dev/null || true
    sleep 2

    progress 95 "正在验证安装..."

    if is_running postgresql || systemctl is-active --quiet "postgresql@${pg_major}-main" 2>/dev/null; then
        local ver
        ver=$(psql --version 2>&1 | grep -oE '[0-9]+\.[0-9]+' | head -1)
        log_success "PostgreSQL 安装成功！版本: ${ver}"
        progress 100 "安装完成"
    else
        log_error "PostgreSQL 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== Redis 安装 ==========
install_redis() {
    log_info "开始安装 Redis..."

    if is_installed redis-server; then
        local ver
        ver=$(redis-server --version 2>&1 | grep -oE 'v=[0-9]+\.[0-9]+\.[0-9]+' | cut -d= -f2)
        log_success "Redis 已安装: ${ver}"
        progress 100 "Redis 已安装"
        exit 0
    fi

    progress 5 "准备安装环境..."
    clean_repo "redis"
    install_deps

    progress 15 "配置 Redis 仓库..."
    log_info "添加 Redis 官方仓库..."

    local codename
    codename=$(get_codename)

    # 导入签名密钥
    curl -fsSL https://packages.redis.io/gpg | gpg --dearmor --yes -o /usr/share/keyrings/redis-archive-keyring.gpg 2>/dev/null || true

    # 添加仓库
    echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb ${codename} main" > /etc/apt/sources.list.d/redis.list

    update_sources

    progress 40 "正在安装 Redis..."
    if [ -n "$VERSION" ] && [ "$VERSION" != "latest" ]; then
        log_info "尝试安装 Redis ${VERSION}..."
        local avail_ver
        avail_ver=$(apt-cache madison redis-server | grep "$VERSION" | head -1 | awk '{print $3}')
        if [ -n "$avail_ver" ]; then
            apt-get install -y -qq "redis-server=${avail_ver}" 2>&1 | while IFS= read -r line; do
                log_detail "$line"
            done
        else
            log_error "未找到 Redis ${VERSION}，安装系统默认版本"
            apt-get install -y -qq redis-server 2>&1 | while IFS= read -r line; do
                log_detail "$line"
            done
        fi
    else
        apt-get install -y -qq redis-server 2>&1 | while IFS= read -r line; do
            log_detail "$line"
        done
    fi

    progress 75 "正在配置 Redis..."
    systemctl enable redis-server 2>/dev/null || true
    systemctl start redis-server 2>/dev/null || true
    sleep 1

    progress 95 "正在验证安装..."

    if is_running redis-server; then
        local ver
        ver=$(redis-server --version 2>&1 | grep -oE 'v=[0-9]+\.[0-9]+\.[0-9]+' | cut -d= -f2)
        log_success "Redis 安装成功！版本: ${ver}"
        progress 100 "安装完成"
    else
        log_error "Redis 安装完成但未运行"
        progress 100 "安装完成（未启动）"
    fi
}

# ========== 获取可用版本 ==========
get_available_versions() {
    local service="$1"
    case "$service" in
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
            log_error "未知的服务: ${service}"
            log_info "支持的服务: nginx, mysql, postgresql, redis"
            exit 1
            ;;
    esac
}

# ========== 主程序 ==========

# 检查参数
if [ -z "$SERVICE" ]; then
    log_error "缺少服务名称参数"
    echo "用法: $0 <service> [version] [root_pass]"
    echo "服务: nginx, mysql, postgresql, redis"
    echo "示例: $0 nginx 1.24.0"
    echo "      $0 mysql latest mypassword"
    echo "      $0 nginx versions  # 列出可用版本"
    exit 1
fi

check_root

# 处理版本查询
if [ "$VERSION" = "versions" ]; then
    get_available_versions "$SERVICE"
    exit 0
fi

# 安装服务
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
        log_error "未知的服务: ${SERVICE}"
        log_info "支持的服务: nginx, mysql, postgresql, redis"
        exit 1
        ;;
esac

exit 0
