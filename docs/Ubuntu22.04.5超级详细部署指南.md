# Ubuntu 22.04.5 全新服务器部署指南（GoPlatform）

本文档面向 **一台全新的 Ubuntu 22.04.5 云主机/物理机**，从零完成 GoPlatform 生产部署。

| 项目 | 说明 |
|------|------|
| 部署形态 | **二进制 + systemd + Nginx**（无 Docker / K8s） |
| 主程序 | `ad-platform-main`（默认端口 `8960`） |
| 管理前端 | `admin/dist` 静态资源，由 Nginx 托管 |
| 下游上报 | `POST /api/report` + `X-API-Key`（`report:write`） |
| 已移除接口 | `/api/getWebTask`、`/upWebTask`（勿再对接） |

**访问方式（三选一）**：

| 场景 | 文档入口 | 浏览器地址示例 |
|------|----------|----------------|
| **有域名** | [§14 Nginx、DNS 与 HTTPS](#14-nginxdns-与-https) | `http://admin.example.com/` |
| **无域名（仅 IP）** | [附录 A](#附录-a无域名仅用-ip-部署与测试)（推荐内网/测试） | `http://203.0.113.10/` |
| **本地开发机** | [附录 B](#附录-b本地开发机windows--linux-测试) | `http://localhost:3000` |

仓库 Nginx 示例默认 `server_name admin.yh-ad.com`；**没有域名时改成 `_` 即可**，不必买域名、不必配 HTTPS。

---

## 0. 场景选择（先看本节）

### 0.1 我只有 Ubuntu 服务器 + 公网 IP，没有域名

按正文 **§2～§16** 做完标准部署后，额外做三件事：

1. Nginx：`server_name _;`（见 [附录 A.2](#a2-nginx-改为-ip-访问)）
2. `config.yaml` → `cors.allow_origins` 加上 `http://<你的公网IP>`
3. 安全组只放行 **80**（不要对公网开 8960）

访问：`http://<公网IP>/` → 管理后台；API 为同域 `/api/...`。

### 0.2 我有域名

按 [§14.1 DNS](#141-dns) 解析后，把 Nginx `server_name` 改成你的域名，并按需配置 [§14.3 HTTPS](#143-https-lets-encrypt-示例)。

### 0.3 我只是在 Windows / Linux 本机开发调试

**不需要** Ubuntu 服务器、**不需要** Nginx。见 [附录 B](#附录-b本地开发机windows--linux-测试)：`go run` + `admin` 里 `npm run dev`。

---

## 目录

1. [部署架构与端口](#1-部署架构与端口)
2. [部署前准备清单](#2-部署前准备清单)
3. [新机系统初始化](#3-新机系统初始化)
4. [安装运行时与中间件](#4-安装运行时与中间件)
5. [获取代码与目录规划](#5-获取代码与目录规划)
6. [生产配置（.env + config.yaml）](#6-生产配置env--configyaml)
7. [MySQL 8.0 安装与建库（推荐路径）](#7-mysql-80-安装与建库推荐路径)
8. [PostgreSQL 安装与建库（可选）](#8-postgresql-安装与建库可选)
9. [Redis 安装与加固](#9-redis-安装与加固)
10. [执行数据库初始化脚本](#10-执行数据库初始化脚本)
11. [编译后端](#11-编译后端)
12. [编译管理前端](#12-编译管理前端)
13. [systemd 托管主服务](#13-systemd-托管主服务)
14. [Nginx、DNS 与 HTTPS](#14-nginxdns-与-https)
15. [防火墙与安全组](#15-防火墙与安全组)
16. [上线验证](#16-上线验证)
17. [首次管理员账号](#17-首次管理员账号)
18. [日常更新与回滚](#18-日常更新与回滚)
19. [备份与日志](#19-备份与日志)
20. [规格建议与检查清单](#20-规格建议与检查清单)
- [0. 场景选择（先看本节）](#0-场景选择先看本节)
- [附录 A：无域名（仅用 IP）部署与测试](#附录-a无域名仅用-ip部署与测试)
- [附录 B：本地开发机（Windows / Linux）测试](#附录-b本地开发机windows--linux-测试)

---

## 1. 部署架构与端口

```
                    Internet
                        |
                   [ Nginx :80/443 ]
                    /              \
         admin 静态页          /api/ 反代
    (admin/dist)                  |
                          [ ad-platform-main :8960 ]
                           /         |         \
                    [ MySQL/PostgreSQL ]   [ Redis ]   [ Metrics :9090 仅本机]
```

| 组件 | 默认端口 | 公网是否暴露 |
|------|----------|--------------|
| Nginx | 80 / 443 | 是 |
| 主服务 | 8960 | **否**（仅本机 + Nginx 反代） |
| MySQL | 3306 | **否** |
| PostgreSQL | 5432 | **否** |
| Redis | 6379 | **否** |
| Prometheus 指标 | 9090 | **否** |
| Pulsar | 6650 | 仅在使用消息队列且 `pulsar.url` 非空时需要 |

**Pulsar**：`config.yaml` 中 `pulsar.url` **留空** 时主程序不初始化消息队列（推荐大多数场景）。非空且生产环境连不通时，**进程会启动失败**。

---

## 2. 部署前准备清单

- [ ] 已申请服务器（建议 Ubuntu **22.04.x** LTS，amd64）
- [ ] 可通过 SSH 登录（密钥登录优先）
- [ ] 域名已解析到服务器公网 IP（如 `admin.example.com` → A 记录）
- [ ] 已准备强随机：`JWT_SECRET`（≥32 字符）、`AES_SECRET_KEY`、`DB`/`REDIS` 密码
- [ ] 选定数据库：**MySQL 8.0** 或 **PostgreSQL**（二选一，与 `DB_DRIVER` 一致）
- [ ] 云厂商安全组：仅放行 **22（SSH）、80、443**；数据库/Redis/8960 **不对公网开放**

---

## 3. 新机系统初始化

以 root 或具备 sudo 的账号登录后执行。

### 3.1 更新系统与基础工具

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl wget vim unzip ca-certificates gnupg lsb-release ufw
```

### 3.2 时区与主机名（可选）

```bash
sudo timedatectl set-timezone Asia/Shanghai
sudo hostnamectl set-hostname ad-platform-prod
```

### 3.3 工作目录

```bash
sudo mkdir -p /data
sudo chown "$USER:$USER" /data
cd /data
```

### 3.4 Swap（2G 内存及以下机器强烈建议）

```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
free -h
```

---

## 4. 安装运行时与中间件

### 4.1 Go（编译后端，建议 1.22+）

```bash
GO_VERSION=1.22.10
wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

### 4.2 Node.js 20（编译管理前端）

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
node -v
npm -v
```

### 4.3 Nginx

```bash
sudo apt install -y nginx
sudo systemctl enable nginx
```

### 4.4 Redis

```bash
sudo apt install -y redis-server
sudo systemctl enable redis-server
```

### 4.5 数据库（二选一）

- **MySQL 8.0**：见 [第 7 节](#7-mysql-80-安装与建库推荐路径)
- **PostgreSQL**：见 [第 8 节](#8-postgresql-安装与建库可选)

### 4.6 Rust（可选）

仅当需要编译 `rust-security` 动态库时安装；多数部署使用预编译 `.so` 或暂不启用 Rust 模块可跳过。

---

## 5. 获取代码与目录规划

### 5.1 克隆仓库

```bash
cd /data
git clone git@github.com:Helong0511/GoPlatform.git
# 或 HTTPS：
# git clone https://github.com/Helong0511/GoPlatform.git

cd /data/GoPlatform
go mod download
```

### 5.2 目录说明

| 路径 | 用途 |
|------|------|
| `/data/GoPlatform` | 项目根、配置、二进制工作目录 |
| `/data/GoPlatform/.env` | 敏感配置（**勿提交 Git**） |
| `/data/GoPlatform/config/config.yaml` | 非敏感/结构化配置 |
| `/data/GoPlatform/admin/dist` | 管理前端构建产物 |
| `/data/GoPlatform/ad-platform-main` | 主服务二进制 |
| `/data/backups/` | 建议自建数据库备份目录 |

---

## 6. 生产配置（.env + config.yaml）

> **无域名**：`cors.allow_origins` 必须包含 `http://<公网IP>`（见 [附录 A.3](#a3-configyaml--env-无域名时要改什么)）。  
> **本地开发**：用 `development` + `localhost:3000`，见 [附录 B.2](#b2-配置本地)。

### 6.1 创建 `.env`

```bash
cd /data/GoPlatform
cp .env.example .env
chmod 600 .env
vim .env
```

**MySQL 生产示例**（请替换 `__REPLACE_ME__`）：

```bash
ENVIRONMENT=production
AUTH_ALLOW_PUBLIC_REGISTER=false

DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=ad_platform
DB_PASSWORD=__REPLACE_ME__
DB_NAME=ad_platform

JWT_SECRET=__REPLACE_ME_至少32字符随机__
AES_SECRET_KEY=__REPLACE_ME_Base64或明文强密钥__

REDIS_PASSWORD=__REPLACE_ME__

# Nginx 反代在本机时，用于正确识别 ClientIP（IP 黑名单依赖此配置）
TRUSTED_PROXIES=127.0.0.1

# 指标接口 Bearer（生产建议设置）
METRICS_BEARER_TOKEN=__REPLACE_ME__
```

**PostgreSQL 生产示例**：将 `DB_DRIVER=postgres`、`DB_PORT=5432`、`DB_USER=postgres`，并保证 `config.yaml` 中 `database.sslmode` 为 `require` 或更严。

> 启动时自动加载 `.env`；**已存在的系统环境变量优先**（不覆盖）。  
> `production` / `staging` 等严格环境会拒绝 `__REPLACE_ME__` 占位符，密钥长度不足也会启动失败。

### 6.2 修改 `config/config.yaml`

```bash
vim /data/GoPlatform/config/config.yaml
```

生产必改项：

```yaml
environment: production

server:
  host: 127.0.0.1          # 仅本机监听，由 Nginx 对外
  port: 8960
  trusted_proxies:
    - "127.0.0.1"

database:
  driver: mysql              # 或 postgres
  host: localhost
  port: 3306                 # postgres 改为 5432
  sslmode: disable           # mysql；postgres 生产用 require

auth:
  allow_public_register: false

pulsar:
  url: ""                    # 不用消息队列则留空

security:
  rust_lib_path: ./rust-security/target/release/librust_security.so  # Linux 路径按实际调整
  hsts_max_age_seconds: 31536000   # 全站 HTTPS 后启用
  ip_blacklist:
    enabled: true
    ban_duration: 24h
    strike_window: 10m
    strikes_before_ban: 5
    static_ips: []
    static_whitelist_ips: []       # 办公出口 IP 等

metrics:
  host: 127.0.0.1
  port: 9090
  bearer_token: ""                 # 建议用环境变量 METRICS_BEARER_TOKEN 覆盖

cors:
  allow_origins:
    - "https://admin.example.com"  # 改成你的管理域名
```

### 6.3 配置校验（可选，本机试跑）

```bash
cd /data/GoPlatform
go build -o ad-platform-main ./cmd/main.go
./ad-platform-main
# 看到 configuration validated 且无报错后 Ctrl+C 停止
```

---

## 7. MySQL 8.0 安装与建库（推荐路径）

```bash
sudo apt install -y mysql-server
sudo systemctl enable mysql
sudo systemctl start mysql
```

### 7.1 创建库与用户

```bash
sudo mysql
```

```sql
CREATE DATABASE IF NOT EXISTS ad_platform
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE USER IF NOT EXISTS 'ad_platform'@'127.0.0.1' IDENTIFIED BY '__强密码__';
CREATE USER IF NOT EXISTS 'ad_platform'@'localhost' IDENTIFIED BY '__强密码__';
GRANT ALL PRIVILEGES ON ad_platform.* TO 'ad_platform'@'127.0.0.1';
GRANT ALL PRIVILEGES ON ad_platform.* TO 'ad_platform'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

`.env` 中 `DB_USER` / `DB_PASSWORD` 与上一致。

### 7.2 限制远程 root（安全）

确保 MySQL **不监听公网**，或安全组未放行 3306。可检查：

```bash
sudo ss -tulpn | grep 3306
```

---

## 8. PostgreSQL 安装与建库（可选）

```bash
sudo apt install -y postgresql postgresql-contrib
sudo systemctl enable postgresql
```

```bash
sudo -u postgres psql -c "CREATE DATABASE ad_platform;"
sudo -u postgres psql -c "CREATE USER ad_platform WITH ENCRYPTED PASSWORD '__强密码__';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE ad_platform TO ad_platform;"
```

`.env` 使用 `DB_DRIVER=postgres`、`DB_PORT=5432`，`config.yaml` 中 `database.sslmode: require`。

---

## 9. Redis 安装与加固

编辑配置：

```bash
sudo vim /etc/redis/redis.conf
```

建议修改：

```conf
bind 127.0.0.1 ::1
requirepass __你的Redis密码__
```

```bash
sudo systemctl restart redis-server
redis-cli -a '__你的Redis密码__' ping
# 应返回 PONG
```

`.env` 中 `REDIS_PASSWORD` 与 `requirepass` 一致。

---

## 10. 执行数据库初始化脚本

**驱动与脚本必须一致**，且脚本会 **DROP 重建表**（仅适合首次部署或明确要清库的环境）。

### MySQL

```bash
mysql -u ad_platform -p ad_platform < /data/GoPlatform/sql/init_db_mysql.sql
```

### PostgreSQL

```bash
psql -U ad_platform -d ad_platform -f /data/GoPlatform/sql/init_db_PostgreSQL.sql
```

脚本已包含：

- 角色/权限（含 `report:create`）
- **IP 审计表**：`ip_blacklist_audit`、`ip_whitelist_audit`（运行时封禁仍以内存 + Redis 为主）

仅补审计表时可用：

```bash
# MySQL
mysql -u ad_platform -p ad_platform < /data/GoPlatform/sql/ip_blacklist_mysql.sql
mysql -u ad_platform -p ad_platform < /data/GoPlatform/sql/ip_whitelist_mysql.sql
```

---

## 11. 编译后端

```bash
cd /data/GoPlatform
go mod tidy
go build -ldflags="-s -w" -o ad-platform-main ./cmd/main.go
chmod +x ad-platform-main
./ad-platform-main &
curl -s http://127.0.0.1:8960/health
kill %1
```

---

## 12. 编译管理前端

```bash
cd /data/GoPlatform/admin
npm install
npm run build
ls -la dist/
```

- 生产 API 默认走相对路径 `/api`（由 Nginx 反代），一般**无需**设置 `REACT_APP_API_URL`。
- 若前后端不同域，构建前可：`export REACT_APP_API_URL=https://api.example.com/api`。

---

## 13. systemd 托管主服务

推荐：**密钥放 `.env`**，systemd 只指定工作目录（进程启动时自动加载 `.env`）。

```bash
sudo cp /data/GoPlatform/scripts/ad-platform-main.service /etc/systemd/system/ad-platform-main.service
sudo vim /etc/systemd/system/ad-platform-main.service
```

**MySQL 最小可用单元**（按需合并进你的 service 文件）：

```ini
[Unit]
Description=AD Platform Main Service
After=network.target redis-server.service mysql.service
Wants=mysql.service redis-server.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/data/GoPlatform
ExecStart=/data/GoPlatform/ad-platform-main
Restart=always
RestartSec=5
Environment="GOTRACEBACK=crash"

StandardOutput=journal
StandardError=journal
SyslogIdentifier=ad-platform-main

[Install]
WantedBy=multi-user.target
```

> 使用 `.env` 后，**不必**在 `[Service]` 里重复写 `JWT_SECRET` 等（除非你想用 systemd 覆盖）。  
> PostgreSQL 将 `mysql.service` 改为 `postgresql.service`。

启用服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable ad-platform-main
sudo systemctl start ad-platform-main
sudo systemctl status ad-platform-main --no-pager
sudo journalctl -u ad-platform-main -n 100 --no-pager
```

---

## 14. Nginx、DNS 与 HTTPS

### 14.0 无域名时（跳过 DNS / HTTPS）

没有域名时：

- **跳过** 本节 14.1（DNS）和 14.3（HTTPS）
- 按 [14.2](#142-应用-nginx-配置) 部署 Nginx，但把 `server_name` 设为 `_`（详见 [附录 A.2](#a2-nginx-改为-ip-访问)）
- 浏览器用 `http://<公网IP>/` 访问

有域名则继续下面 14.1。

### 14.1 DNS（仅有域名时需要）

在域名服务商添加记录：

| 类型 | 主机记录 | 值 |
|------|----------|-----|
| A | `admin` | 服务器公网 IP |

例如：`admin.example.com` → `203.0.113.10`。

### 14.2 应用 Nginx 配置

```bash
sudo cp /data/GoPlatform/scripts/nginx.conf /etc/nginx/nginx.conf
sudo nginx -t
sudo systemctl reload nginx
```

**有域名**时改 `server_name`：

```bash
sudo sed -i 's/admin.yh-ad.com/admin.example.com/g' /etc/nginx/nginx.conf
sudo nginx -t && sudo systemctl reload nginx
```

**无域名、仅用 IP** 时改为接受任意 Host（并建议作为默认站点）：

```bash
sudo sed -i 's/admin.yh-ad.com/_/' /etc/nginx/nginx.conf
# 可选：把 listen 80; 改为 listen 80 default_server;
sudo nginx -t && sudo systemctl reload nginx
```

完整说明与 `server` 块示例见 [附录 A.2](#a2-nginx-改为-ip-访问)。

配置要点（仓库已包含）：

- `/` → `admin/dist` 静态资源
- `/api/` → 反代 `127.0.0.1:8960`
- `/metrics` → `127.0.0.1:9090`（仅允许本机）

### 14.3 HTTPS（Let's Encrypt 示例）

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d admin.example.com
```

证书生效后，在 `config.yaml` 启用 `security.hsts_max_age_seconds: 31536000`，重启主服务。

---

## 15. 防火墙与安全组

**云安全组**：只开 22、80、443。

**本机 UFW（可选）**：

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

**不要**对公网放行：`8960`、`3306`、`5432`、`6379`、`9090`。

---

## 16. 上线验证

将下面命令里的 `<访问地址>` 换成你的实际入口：

| 部署方式 | `<访问地址>` 示例 |
|----------|-------------------|
| 有域名 + HTTPS | `https://admin.example.com` |
| 有域名 HTTP | `http://admin.example.com` |
| **无域名** | `http://<公网IP>` |

### 16.1 服务器上执行（通用）

```bash
# 1) 服务状态
sudo systemctl status ad-platform-main --no-pager

# 2) 日志无连续报错
sudo journalctl -u ad-platform-main -n 200 --no-pager

# 3) 主程序（仅本机，应成功）
curl -f http://127.0.0.1:8960/health
curl -f http://127.0.0.1:8960/health/ready

# 4) 经 Nginx（本机）
curl -f http://127.0.0.1/health
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://127.0.0.1/

# 5) 端口（8960/3306/6379 应仅监听 127.0.0.1；80 对外）
sudo ss -tulpn | grep -E '8960|3306|5432|6379|80|443|9090'
```

### 16.2 在你自己的电脑上（无域名时必做）

把 `<公网IP>` 换成云主机面板里的公网地址：

```bash
curl -f http://<公网IP>/health
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://<公网IP>/
```

浏览器打开 `http://<公网IP>/`，应出现**登录页**。  
登录后请求为 `http://<公网IP>/api/...`（前端 `baseUrl` 默认为 `/api`，与是否有域名无关）。

### 16.3 有域名时

```bash
curl -f http://admin.example.com/health
curl -I http://admin.example.com/
```

浏览器访问 `https://admin.example.com`（或你的 HTTP 地址），应出现登录页。

### 16.4 下游上报抽测

```bash
curl -s -X POST "http://<访问地址>/api/report" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <具备 report:write 的 API Key>" \
  -d '{"action":"click","user_id":"test-u1"}'
```

无域名时 `<访问地址>` 填 `http://<公网IP>`。更多用例见 [附录 A.6](#a6-下游上报测试无域名)。

**管理端**：侧栏 **系统配置**（`/system-config`，读写 `config.yaml`，密钥脱敏，IP 黑名单可热更新）、**IP 访问控制**（`/ip-access`）；均需 `user:read` / `user:update`。接口见 `ubuntu/api-spec.txt`。

---

## 17. 首次管理员账号

初始化脚本**默认不插入**可登录用户（避免弱口令入库）。可选方式：

**方式 A：临时开启注册（仅内网/测试）**

1. `.env` 设 `AUTH_ALLOW_PUBLIC_REGISTER=true`，重启服务  
2. 浏览器注册第一个账号  
3. 在库中把该用户关联 `super_admin` 角色  
4. 关闭公开注册并重启  

**方式 B：SQL 插入（生产常用）**

1. 用 bcrypt 生成密码哈希（Go/Python 均可）  
2. `INSERT INTO users (...)` 并 `INSERT INTO user_roles` 关联 `super_admin`  

注册审核、API Key、IP 黑白名单等见管理后台各菜单。

---

## 18. 日常更新与回滚

### 18.1 标准更新

```bash
cd /data/GoPlatform
sudo systemctl stop ad-platform-main
cp ad-platform-main ad-platform-main.backup.$(date +%Y%m%d)

git pull
go mod tidy
go build -ldflags="-s -w" -o ad-platform-main ./cmd/main.go

cd admin
npm install
npm run build

sudo systemctl start ad-platform-main
curl -f http://127.0.0.1:8960/health
```

### 18.2 回滚

```bash
sudo systemctl stop ad-platform-main
cp ad-platform-main.backup.YYYYMMDD ad-platform-main
chmod +x ad-platform-main
sudo systemctl start ad-platform-main
```

---

## 19. 备份与日志

### 19.1 journald 限制

```ini
# /etc/systemd/journald.conf
[Journal]
SystemMaxUse=1G
MaxRetentionSec=7day
Compress=yes
```

```bash
sudo systemctl restart systemd-journald
```

### 19.2 数据库备份

```bash
chmod +x /data/GoPlatform/scripts/backup_mysql.sh
chmod +x /data/GoPlatform/scripts/backup_postgres.sh
```

MySQL 示例：

```bash
export MYSQL_PWD='__DB密码__'
BACKUP_DIR=/data/backups/mysql DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=ad_platform DB_NAME=ad_platform RETENTION_DAYS=7 \
  /data/GoPlatform/scripts/backup_mysql.sh
```

### 19.3 常用运维命令

```bash
sudo systemctl restart ad-platform-main
sudo journalctl -u ad-platform-main -f
sudo nginx -t && sudo systemctl reload nginx
```

---

## 20. 规格建议与检查清单

### 20.1 规格

| 场景 | 建议 |
|------|------|
| 主服务 + DB/Redis 同机（低流量） | 4C8G+ |
| 主服务单独、DB 外置 | 2C4G 起 |
| 2C2G | 仅主服务，DB/Redis 外置；见下文轻量化参数 |

### 20.2 2C2G 轻量化（`config.yaml` 摘录）

```yaml
database:
  max_open_conns: 8
  max_idle_conns: 4
rate_limit:
  requests_per_second: 20
  burst: 80
redis:
  pool_size: 20
  min_idle_conns: 5
```

### 20.3 上线前检查（Pre-Flight）

- [ ] `DB_DRIVER` / `DB_PORT` / 初始化 SQL 三者一致  
- [ ] `.env` 中无 `__REPLACE_ME__`，`ENVIRONMENT=production`  
- [ ] `AUTH_ALLOW_PUBLIC_REGISTER=false`  
- [ ] `REDIS_PASSWORD`、数据库密码已设置  
- [ ] `server.host=127.0.0.1`，`metrics.host=127.0.0.1`  
- [ ] Nginx `server_name`、CORS `allow_origins` 与真实域名一致  
- [ ] `TRUSTED_PROXIES=127.0.0.1`（Nginx 同机反代时）  
- [ ] `admin/dist` 已构建，Nginx `nginx -t` 通过  
- [ ] 安全组未暴露 3306/6379/8960/9090  
- [ ] 备份脚本手工跑通一次  

### 20.4 上线后 30 分钟

- [ ] `/health`、`/health/ready` 稳定  
- [ ] 管理后台可登录  
- [ ] 无 OOM / 反复重启  
- [ ] `journalctl` 无持续 DB/Redis 连接失败  

### 20.5 回退条件

- 10 分钟内连续重启 ≥3 次  
- 核心接口不可用超过 5 分钟  
- 数据库连接 15 分钟内无法恢复  

---

## 附录

| 文件 | 说明 |
|------|------|
| `ubuntu/deployment-notes.txt` | 运维速记 |
| `ubuntu/api-spec.txt` | 对外 API 与 IP 管理接口 |
| `.env.example` | 环境变量模板 |
| `scripts/nginx.conf` | Nginx 全量配置参考 |
| `scripts/ad-platform-main.service` | systemd 模板 |

**密钥扫描（发布前，开发机 Windows）**：

```powershell
pwsh ./scripts/check-secrets.ps1
```

---

## 附录 A：无域名（仅用 IP）部署与测试

没有域名时，用 **服务器公网 IP** 访问即可：不必配置 DNS、不必申请 HTTPS（内网/测试用 HTTP 即可）。

### A.1 原理说明（为什么要这样配）

```
你的电脑浏览器  ──HTTP:80──►  云服务器 Nginx  ──反代──►  127.0.0.1:8960 主程序
                              │
                              └── 静态文件 admin/dist（登录页）
```

- 生产建议 `server.host: 127.0.0.1`：主程序**不**对公网直接暴露 8960。
- 用户只访问 `http://公网IP/`，由 Nginx 同时提供页面和 `/api`。
- 因此 **`http://公网IP:8960` 从外网打不开是正常现象**，不要改防火墙去放行 8960（除非你很清楚风险）。

### A.2 访问方式对照

| 场景 | 管理后台 | 后端 API | 说明 |
|------|----------|----------|------|
| **推荐（服务器对外）** | `http://<公网IP>/` | `http://<公网IP>/api/...` | 完成 §2～§16 + 本节配置 |
| 仅测 API（SSH 登录服务器后） | — | `http://127.0.0.1:8960/health` | 在服务器本机 curl |
| 开发机联调 | `http://localhost:3000` | Vite → `8960` | [附录 B](#附录-b本地开发机windows--linux-测试) |

### A.3 无域名最小步骤清单（在标准部署之后勾选）

假定你已完成正文 **§2～§13**（代码、`.env`、`config.yaml`、数据库初始化、`go build`、`systemd`、`admin/dist` 已构建）。

- [ ] **A-1** Nginx：`server_name _;`（[A.4](#a4-nginx-改为-ip-访问)）
- [ ] **A-2** `cors.allow_origins` 含 `http://<公网IP>`（[A.5](#a5-configyaml--env-无域名时要改什么)）
- [ ] **A-3** `.env` 中 `TRUSTED_PROXIES=127.0.0.1`（Nginx 与主程序同机）
- [ ] **A-4** 云安全组 / UFW：只开 **22、80**（[A.6](#a6-云服务器安全组--防火墙)）
- [ ] **A-5** `sudo systemctl restart ad-platform-main && sudo systemctl reload nginx`
- [ ] **A-6** 本机电脑浏览器打开 `http://<公网IP>/`（[A.7](#a7-在服务器上测试无域名)）
- [ ] **A-7** 登录、菜单、上报接口抽测（[A.8](#a8-下游上报测试无域名)）

### A.4 Nginx 改为 IP 访问

编辑 `/etc/nginx/nginx.conf`（或仓库 `scripts/nginx.conf`）中 `server` 块，关键两行：

```nginx
server {
    listen 80 default_server;
    server_name _;    # 无域名：匹配任意 Host（含直接用 IP 访问）

    location / {
        root /data/GoPlatform/admin/dist;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8960/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location = /health {
        proxy_pass http://127.0.0.1:8960/health;
    }

    # ... 其余 location 与仓库 scripts/nginx.conf 一致 ...
}
```

一键替换仓库默认域名：

```bash
sudo cp /data/GoPlatform/scripts/nginx.conf /etc/nginx/nginx.conf
sudo sed -i 's/admin.yh-ad.com/_/' /etc/nginx/nginx.conf
sudo nginx -t && sudo systemctl reload nginx
```

浏览器访问：`http://<你的公网IP>/`（例如云面板显示的 `159.138.133.34`，**不要**加端口 8960）。

### A.5 config.yaml / .env 无域名时要改什么

编辑 `/data/GoPlatform/config/config.yaml`：

```yaml
environment: production
# 内网试运行、校验想宽松时可改为 development

server:
  host: 127.0.0.1
  port: 8960

cors:
  allow_origins:
    - "http://<你的公网IP>"      # 必填：把 <你的公网IP> 换成真实 IP
    - "http://127.0.0.1"
  # 仅临时排障可改为 ["*"]，生产不建议长期使用
```

编辑 `/data/GoPlatform/.env`（密钥仍放这里，**不要**写域名）：

```bash
ENVIRONMENT=production
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=ad_platform
DB_PASSWORD=你的强密码
DB_NAME=ad_platform
JWT_SECRET=至少32位随机字符
AES_SECRET_KEY=你的AES密钥
REDIS_PASSWORD=你的Redis密码
AUTH_ALLOW_PUBLIC_REGISTER=false
TRUSTED_PROXIES=127.0.0.1
```

也可在管理后台 **系统配置**（`/system-config`）修改部分项；改 `cors`、`.env` 覆盖项后需重启主程序。

保存并重启：

```bash
sudo systemctl restart ad-platform-main
sudo systemctl reload nginx
sudo journalctl -u ad-platform-main -n 50 --no-pager
```

### A.6 云服务器安全组 / 防火墙

| 端口 | 是否对公网开放 | 说明 |
|------|----------------|------|
| 22 | 是 | SSH |
| 80 | 是 | HTTP（Nginx） |
| 443 | 可选 | 无 HTTPS 时可不开 |
| 8960 | **否** | 主程序仅本机 |
| 3306 / 5432 | **否** | 数据库 |
| 6379 | **否** | Redis |
| 9090 | **否** | 指标 |

云厂商控制台「安全组」与服务器 UFW 保持一致：

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw enable
sudo ufw status
```

### A.7 在服务器上测试（无域名）

在 **SSH 登录服务器** 后执行：

```bash
# 主程序是否正常
curl -s http://127.0.0.1:8960/health | head -c 200
echo

# Nginx 反代是否正常
curl -s http://127.0.0.1/health | head -c 200
echo

# 首页是否 200（返回 HTML）
curl -s -o /dev/null -w "index HTTP %{http_code}\n" http://127.0.0.1/
```

在 **你自己的电脑**（Windows PowerShell / Linux 终端）执行，`<公网IP>` 换成服务器公网地址：

```powershell
curl http://<公网IP>/health
curl -I http://<公网IP>/
```

**浏览器测试（推荐）**：

1. 打开 `http://<公网IP>/` → 应看到管理后台登录页  
2. 登录（需已 [§17 创建管理员](#17-首次管理员账号)）  
3. 打开开发者工具 → Network：登录请求应为 `http://<公网IP>/api/login`，状态 200  
4. 侧栏可进：用户管理、系统配置、IP 访问控制 等  

### A.8 下游上报测试（无域名）

在管理后台为用户生成具备 `report:write` 的 API Key 后：

```bash
curl -s -X POST "http://<公网IP>/api/report" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <你的API密钥>" \
  -d '{"action":"click","user_id":"test-u1"}'
```

成功时返回 JSON，`code` 为 200（具体字段以接口为准）。

### A.9 从开发机访问远程服务器（仍无域名）

与在服务器上对外提供服务相同：在你 PC 浏览器输入 `http://<公网IP>/` 即可。  
**不要**依赖 `http://<公网IP>:8960`，除非你把 `server.host` 改成 `0.0.0.0` 且安全组放行 8960（不推荐）。

### A.10 常见问题

| 现象 | 原因 | 处理 |
|------|------|------|
| 浏览器能开页但登录失败 / 跨域 | `cors` 未含 `http://<公网IP>` | 改 `config.yaml` 后 `systemctl restart ad-platform-main` |
| `http://IP:8960` 外网超时 | 8960 未对公网监听 | 改用 `http://IP/`（80 端口） |
| Nginx 502 Bad Gateway | 主程序未启动或崩溃 | `journalctl -u ad-platform-main -n 100` |
| 启动报 `__REPLACE_ME__` / 密钥错误 | 生产校验 | 填真实 `JWT_SECRET`、`REDIS_PASSWORD` 等 |
| 页面 404 | 未构建前端 | `cd admin && npm install && npm run build` |
| 能登录但接口 401 | Token 过期或未带 Bearer | 重新登录；检查 Nginx 是否截断 Header |

---

## 附录 B：本地开发机（Windows / Linux）测试

在 **自己电脑** 上联调：**不需要** Ubuntu 部署、**不需要** 域名、**不需要** Nginx。  
典型路径：仓库在 `E:\GoPlatform`（Windows）或 `~/GoPlatform`（Linux）。

### B.1 与服务器部署的区别

| 项目 | 本地开发 | Ubuntu 无域名 |
|------|----------|---------------|
| 管理后台 | `npm run dev` → `:3000` | Nginx → `http://IP/` |
| 后端 | `go run` → `:8960` | systemd → 本机 `:8960` + Nginx 反代 |
| 配置环境 | `development` | 多为 `production` |
| 数据库 | 本机 MySQL/PostgreSQL | 服务器本机或同机库 |

### B.2 准备（一次性）

1. **Go** 1.22+、`Node.js` 20+  
2. **MySQL 8** 或 **PostgreSQL**（与 `config.yaml` 里 `database.driver` 一致）  
3. **Redis**（Windows 可用 WSL / Memurai / 远程 Redis）  
4. 克隆或已有代码目录，例如 `E:\GoPlatform`

初始化数据库（按驱动二选一）：

```powershell
# MySQL 示例（在 mysql 客户端里先 CREATE DATABASE ad_platform）
mysql -uroot -p ad_platform < E:\GoPlatform\sql\init_db_mysql.sql
```

```bash
# PostgreSQL 示例
psql -U postgres -d ad_platform -f sql/init_db_PostgreSQL.sql
```

### B.3 配置（本地）

```powershell
cd E:\GoPlatform
copy .env.example .env
notepad .env
```

`.env` 示例（按本机修改）：

```bash
ENVIRONMENT=development
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=你的本地MySQL密码
DB_NAME=ad_platform
JWT_SECRET=12345678901234567890123456789012
AES_SECRET_KEY=1234567890123456
REDIS_PASSWORD=
AUTH_ALLOW_PUBLIC_REGISTER=true
```

`config/config.yaml` 关键项：

```yaml
environment: development

server:
  host: 0.0.0.0
  port: 8960

database:
  driver: mysql
  host: localhost
  port: 3306
  sslmode: disable

cors:
  allow_origins:
    - "http://localhost:3000"
    - "http://127.0.0.1:3000"

security:
  ip_blacklist:
    enabled: true
```

> `development` 下校验较松，便于排错；勿把开发密钥用于生产。

### B.4 启动后端（终端 1）

**Windows PowerShell：**

```powershell
cd E:\GoPlatform
go mod download
go run ./cmd/main.go
```

或编译后运行：

```powershell
go build -o ad-platform-main.exe .\cmd\main.go
.\ad-platform-main.exe
```

**Linux / macOS：**

```bash
cd ~/GoPlatform
go run ./cmd/main.go
```

看到日志含 `configuration validated`、`server is running` 即正常。  
另开终端验证：

```powershell
curl http://127.0.0.1:8960/health
```

### B.5 启动管理前端（终端 2）

```powershell
cd E:\GoPlatform\admin
npm install
npm run dev
```

终端会提示类似：

```text
  VITE v5.x  ready in xxx ms
  ➜  Local:   http://localhost:3000/
```

浏览器打开：**http://localhost:3000**

仓库 `admin/vite.config.js` 已配置开发代理：

- 浏览器请求 `http://localhost:3000/api/...` → 自动转发到 `http://127.0.0.1:8960/api/...`
- **无需**设置 `REACT_APP_API_URL`

### B.6 本地测试清单（按顺序做）

| 序号 | 测试项 | 操作与预期 |
|------|--------|------------|
| 1 | 健康检查 | `curl http://127.0.0.1:8960/health` 返回 200 JSON |
| 2 | 前端页面 | `http://localhost:3000` 出现登录页 |
| 3 | 注册 | 若 `AUTH_ALLOW_PUBLIC_REGISTER=true`，访问注册页创建账号 |
| 4 | 登录 | 登录后进入仪表盘，Network 里 `/api/login` 为 200 |
| 5 | 管理菜单 | 用户 / 广告 / 系统配置 / IP 访问控制 可打开 |
| 6 | 系统配置 | `/system-config` 能加载 YAML，保存后看提示是否需重启 |
| 7 | 下游上报 | 见 [B.7](#b7-本地下游上报测试) |

**首次管理员**：若关闭公开注册，需按 [§17](#17-首次管理员账号) 在库里创建用户并绑定 `super_admin` 角色。

### B.7 本地下游上报测试

```powershell
curl -X POST http://127.0.0.1:8960/api/report `
  -H "Content-Type: application/json" `
  -H "X-API-Key: 你的API密钥" `
  -d "{\"action\":\"click\",\"user_id\":\"u1\"}"
```

Linux / bash：

```bash
curl -s -X POST http://127.0.0.1:8960/api/report \
  -H "Content-Type: application/json" \
  -H "X-API-Key: 你的API密钥" \
  -d '{"action":"click","user_id":"u1"}'
```

### B.8 本地“像生产一样”测法（可选）

不启 Vite，改与服务器相同：构建静态资源 + 本机 Nginx（或只测 API）。

```bash
cd admin
npm run build
# 将 dist 配到 Nginx，或仅用 Postman 调 http://127.0.0.1:8960/api/...
```

### B.9 本地常见问题

| 现象 | 处理 |
|------|------|
| `go run` 报数据库连接失败 | 检查 MySQL/Redis 是否启动、`.env` 密码与端口 |
| 前端能开页，接口 404 | 确认后端在 8960 运行；确认用的是 `npm run dev` 而非直接打开 dist |
| 跨域错误 | `cors.allow_origins` 含 `http://localhost:3000` |
| `npm run dev` 端口不是 3000 | 看终端输出；或改 `vite.config.js` 里 `server.port` |
| Windows 本机无 Redis | 安装 Redis 或 WSL；开发环境连接失败时看日志是否降级为 mock |

### B.10 本地开发 vs 访问远程 IP

- **本地开发**：改代码、断点调试 → 用本节附录 B。  
- **访问已部署的 Ubuntu**：浏览器打开 `http://<公网IP>/` → 用 [附录 A](#附录-a无域名仅用-ip-部署与测试)，不要在本地再起一套去连远程 8960（除非刻意放行且知情）。

---

*文档版本：与当前仓库单服务架构、IP 黑白名单、.env 加载、Pulsar 可选初始化保持一致。*
