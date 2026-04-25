# Ubuntu22.04.5超级详细部署指南

本文档为 GoPlatform **单服务标准版**部署指南。  
平台入口仅 `ad-platform-main`，不再包含独立任务服务或独立上报服务。

## 1. 架构与端口

- 主服务：`ad-platform-main`（`8960`）
- 数据库：PostgreSQL / MySQL8.0（按 `database.driver` 选择）
- 缓存：Redis（`6379`）
- 消息：Pulsar（`6650`）
- 反向代理：Nginx（`80/443`）
- 监控：Prometheus（`9090`）

## 2. 服务器准备

MySQL 本机部署场景：

```bash
sudo apt update
sudo apt install -y git curl wget vim nginx redis-server mysql-server
```

PostgreSQL 本机部署场景：

```bash
sudo apt update
sudo apt install -y git curl wget vim nginx redis-server postgresql postgresql-contrib
```

按需安装 Go、Node.js、Rust（如需编译 rust-security）。  
若数据库为远程托管（RDS 等），可不安装本机数据库服务，但需正确配置 `DB_HOST` / `DB_PORT`。

## 3. 拉取代码与依赖

```bash
cd /data
sudo git clone git@github.com:Helong0511/GoPlatform.git
cd /data/GoPlatform

go mod download
go mod tidy
```

## 4. 配置文件

编辑 `config/config.yaml`，重点检查：

- `server.port`（默认 `8960`）
- `database.driver`（`postgres` 或 `mysql`）与 `database.*`
- `redis.*`
- `pulsar.*`
- `jwt.secret` 与 `security.aes_secret`（建议通过环境变量覆盖）

如使用环境变量，确保设置：

- `DB_DRIVER`（`postgres` 或 `mysql`）
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `JWT_SECRET`
- `AES_SECRET_KEY`
- `REDIS_PASSWORD`（如有）
- `ENVIRONMENT`（建议生产环境为 `production`）
- `AUTH_ALLOW_PUBLIC_REGISTER`（`true/false`，生产建议 `false`）

公开注册开关规则：

- 若显式设置 `AUTH_ALLOW_PUBLIC_REGISTER`，以该值为准
- 若未设置，则按 `environment` 默认值：`development/dev/local` 默认开启，其它默认关闭

## 5. 初始化数据库（按驱动选择）

当 `database.driver=postgres` 时：

```bash
psql -U postgres -d ad_platform -f /data/GoPlatform/sql/init_db_PostgreSQL.sql
```

当 `database.driver=mysql` 时：

```bash
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS ad_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -uroot -p ad_platform < /data/GoPlatform/sql/init_db_mysql.sql
```

## 6. 编译

```bash
cd /data/GoPlatform
go build -o ad-platform-main ./cmd/main.go
chmod +x ad-platform-main
```

## 7. systemd 服务

创建 `/etc/systemd/system/ad-platform-main.service`：

### 7.1 MySQL 生产版（推荐模板）

```ini
[Unit]
Description=AD Platform Main Service
After=network.target redis-server.service mysql.service mariadb.service
Requires=redis-server.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/data/GoPlatform
ExecStart=/data/GoPlatform/ad-platform-main
Restart=always
RestartSec=5
StartLimitInterval=0
StartLimitBurst=10

Environment="ENVIRONMENT=production"
Environment="AUTH_ALLOW_PUBLIC_REGISTER=false"
Environment="DB_DRIVER=mysql"
Environment="DB_HOST=127.0.0.1"
Environment="DB_PORT=3306"
Environment="DB_USER=remote"
Environment="DB_PASSWORD=__REPLACE_ME__"
Environment="DB_NAME=ad_platform"
Environment="JWT_SECRET=__REPLACE_ME__"
Environment="AES_SECRET_KEY=__REPLACE_ME__"

Environment="GOMAXPROCS=1"
Environment="GOGC=80"
Environment="GOTRACEBACK=crash"
Environment="GODEBUG=madvdontneed=1"
Environment="MALLOC_CONF=dirty_decay_ms:30000,muzzy_decay_ms:30000"

LimitNOFILE=524288
LimitNPROC=32768
LimitMEMLOCK=infinity
LimitAS=infinity
Nice=0
CPUSchedulingPolicy=rr
CPUSchedulingPriority=90
IOWeight=70
OOMScoreAdjust=-500

StandardOutput=journal
StandardError=journal
SyslogIdentifier=ad-platform-main
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
```

### 7.2 PostgreSQL 生产版（推荐模板）

```ini
[Unit]
Description=AD Platform Main Service
After=network.target redis-server.service postgresql.service
Requires=redis-server.service postgresql.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/data/GoPlatform
ExecStart=/data/GoPlatform/ad-platform-main
Restart=always
RestartSec=5
StartLimitInterval=0
StartLimitBurst=10

Environment="ENVIRONMENT=production"
Environment="AUTH_ALLOW_PUBLIC_REGISTER=false"
Environment="DB_DRIVER=postgres"
Environment="DB_HOST=127.0.0.1"
Environment="DB_PORT=5432"
Environment="DB_USER=postgres"
Environment="DB_PASSWORD=__REPLACE_ME__"
Environment="DB_NAME=ad_platform"
Environment="JWT_SECRET=__REPLACE_ME__"
Environment="AES_SECRET_KEY=__REPLACE_ME__"

Environment="GOMAXPROCS=1"
Environment="GOGC=80"
Environment="GOTRACEBACK=crash"
Environment="GODEBUG=madvdontneed=1"
Environment="MALLOC_CONF=dirty_decay_ms:30000,muzzy_decay_ms:30000"

LimitNOFILE=524288
LimitNPROC=32768
LimitMEMLOCK=infinity
LimitAS=infinity
Nice=0
CPUSchedulingPolicy=rr
CPUSchedulingPriority=90
IOWeight=70
OOMScoreAdjust=-500

StandardOutput=journal
StandardError=journal
SyslogIdentifier=ad-platform-main
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
```

启动并设置开机自启（两种驱动通用）：

```bash
sudo systemctl daemon-reload
sudo systemctl start ad-platform-main
sudo systemctl enable ad-platform-main
sudo systemctl status ad-platform-main
```

## 8. Nginx 配置

使用仓库内单服务版本配置：

```bash
sudo cp /data/GoPlatform/scripts/nginx.conf /etc/nginx/nginx.conf
sudo nginx -t
sudo systemctl reload nginx
```

## 9. 防火墙（按需）

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8960/tcp
sudo ufw allow 5432/tcp
sudo ufw allow 6379/tcp
sudo ufw allow 6650/tcp
sudo ufw allow 9090/tcp
```

## 10. 验证

```bash
curl http://127.0.0.1:8960/health
sudo ss -tulpn | grep -E '8960|5432|6379|6650|80|443|9090'
```

密钥提交前扫描（建议每次发布前执行）：

```powershell
pwsh ./scripts/check-secrets.ps1
```

权限初始化说明：

- `sql/init_db_PostgreSQL.sql` 与 `sql/init_db_mysql.sql` 已包含 `report:create` 权限初始化，无需额外执行旧补丁脚本。

上线后 5 分钟快速核验（建议每次发布执行）：

```bash
# 1) 服务状态
sudo systemctl status ad-platform-main --no-pager

# 2) 近200行日志是否有数据库连接错误
sudo journalctl -u ad-platform-main -n 200 --no-pager

# 3) 健康检查
curl -f http://127.0.0.1:8960/health

# 4) 端口检查
sudo ss -tulpn | grep -E '8960|6379|6650|80|443|9090|5432|3306'
```

驱动与脚本一致性检查（避免混配）：

- MySQL：`DB_DRIVER=mysql` + `DB_PORT=3306` + `sql/init_db_mysql.sql`
- PostgreSQL：`DB_DRIVER=postgres` + `DB_PORT=5432` + `sql/init_db_PostgreSQL.sql`

## 11. 常用运维

```bash
# 服务管理
sudo systemctl start ad-platform-main
sudo systemctl stop ad-platform-main
sudo systemctl restart ad-platform-main
sudo systemctl status ad-platform-main

# 日志
sudo journalctl -u ad-platform-main -n 100 --no-pager
sudo journalctl -u ad-platform-main -f
```

## 12. 更新流程（标准）

```bash
cd /data/GoPlatform
sudo cp ad-platform-main ad-platform-main.backup
sudo -u <deploy-user> git pull

sudo systemctl stop ad-platform-main
go mod tidy
go build -o ad-platform-main ./cmd/main.go
sudo systemctl start ad-platform-main

curl http://127.0.0.1:8960/health
```

## 13. 回滚

```bash
sudo systemctl stop ad-platform-main
sudo cp /data/GoPlatform/ad-platform-main.backup /data/GoPlatform/ad-platform-main
sudo chmod +x /data/GoPlatform/ad-platform-main
sudo systemctl start ad-platform-main
```
