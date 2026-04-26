# Ubuntu22.04.5超级详细部署指南

本文档为 GoPlatform **单服务标准版**部署指南。  
平台入口仅 `ad-platform-main`，不再包含独立任务服务或独立上报服务。

> 注意：按当前 `cmd/main.go` 实现，服务启动时会**跳过 Pulsar 初始化**（开发策略保留）。  
> 若你的生产链路依赖 Pulsar，请先在代码中恢复/开启初始化逻辑，再按本指南配置与放通 Pulsar。

## 目录索引

- [1. 架构与端口](#1-架构与端口)
- [2. 服务器准备](#2-服务器准备)
- [3. 拉取代码与依赖](#3-拉取代码与依赖)
- [4. 配置文件](#4-配置文件)
- [5. 初始化数据库（按驱动选择）](#5-初始化数据库按驱动选择)
- [6. 编译](#6-编译)
- [7. systemd 服务](#7-systemd-服务)
- [8. Nginx 配置](#8-nginx-配置)
- [9. 防火墙（按需）](#9-防火墙按需)
- [10. 验证](#10-验证)
- [11. 常用运维](#11-常用运维)
- [12. 更新流程（标准）](#12-更新流程标准)
- [13. 回滚](#13-回滚)
- [14. 日志与数据备份/销毁（生产必做）](#14-日志与数据备份销毁生产必做)
- [15. 服务器规格建议（按部署形态）](#15-服务器规格建议按部署形态)
- [16. 2核2G 专用推荐参数（轻量化）](#16-2核2g-专用推荐参数轻量化)
- [17. 上线前/后超级详细检查清单（建议打印执行）](#17-上线前后超级详细检查清单建议打印执行)

## 1. 架构与端口

- 主服务：`ad-platform-main`（`8960`）
- 数据库：PostgreSQL / MySQL8.0（按 `database.driver` 选择）
- 缓存：Redis（`6379`）
- 消息：Pulsar（`6650`，当前版本默认不在主程序初始化）
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

说明：当前版本即使安装了 Pulsar，主程序也会跳过初始化；仅在你修改代码启用 Pulsar 后，该依赖才是运行必需项。

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

Pulsar 配置提示：

- 当前版本即使配置了 `pulsar.*` 也不会在主程序初始化连接。
- 建议保留配置项，便于后续启用消息链路时直接切换。

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

## 14. 日志与数据备份/销毁（生产必做）

### 14.1 journald 日志保留策略（避免磁盘被日志打满）

编辑 `/etc/systemd/journald.conf`：

```ini
[Journal]
SystemMaxUse=1G
RuntimeMaxUse=256M
MaxRetentionSec=7day
Compress=yes
```

应用配置：

```bash
sudo systemctl restart systemd-journald
sudo journalctl --vacuum-time=7d
```

### 14.2 数据库备份脚本（仓库内可直接用）

- PostgreSQL 备份：`scripts/backup_postgres.sh`
- PostgreSQL 恢复：`scripts/restore_postgres.sh`
- MySQL 备份：`scripts/backup_mysql.sh`
- MySQL 恢复：`scripts/restore_mysql.sh`

首次部署请赋予执行权限：

```bash
cd /data/GoPlatform
chmod +x scripts/backup_postgres.sh scripts/restore_postgres.sh scripts/backup_mysql.sh scripts/restore_mysql.sh
```

PostgreSQL 备份示例：

```bash
export PGPASSWORD='__REPLACE_ME__'
BACKUP_DIR=/data/backups/postgres DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=postgres DB_NAME=ad_platform RETENTION_DAYS=7 \
  /data/GoPlatform/scripts/backup_postgres.sh
```

MySQL 备份示例：

```bash
export MYSQL_PWD='__REPLACE_ME__'
BACKUP_DIR=/data/backups/mysql DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_NAME=ad_platform RETENTION_DAYS=7 \
  /data/GoPlatform/scripts/backup_mysql.sh
```

恢复示例（执行前请确认环境）：

```bash
# PostgreSQL
export PGPASSWORD='__REPLACE_ME__'
DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=postgres DB_NAME=ad_platform \
  /data/GoPlatform/scripts/restore_postgres.sh /data/backups/postgres/ad_platform_full_YYYYMMDD_HHMMSS.sql.gz

# MySQL
export MYSQL_PWD='__REPLACE_ME__'
DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_NAME=ad_platform \
  /data/GoPlatform/scripts/restore_mysql.sh /data/backups/mysql/ad_platform_full_YYYYMMDD_HHMMSS.sql.gz
```

### 14.3 PostgreSQL 上报分区清理（数据销毁）

初始化清理函数：

```bash
psql -U postgres -d ad_platform -f /data/GoPlatform/sql/cleanup_old_postgres_partitions.sql
```

预览将删除的分区（不执行删除）：

```sql
SELECT * FROM cleanup_old_ad_reports_partitions(180, true);
```

执行删除（保留最近 180 天）：

```sql
SELECT * FROM cleanup_old_ad_reports_partitions(180, false);
```

### 14.4 定时任务（cron 示例）

```bash
crontab -e
```

按数据库类型选择以下任务：

```cron
# 每天 02:30 PostgreSQL 全量+结构备份，保留 7 天
30 2 * * * PGPASSWORD=__REPLACE_ME__ BACKUP_DIR=/data/backups/postgres DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=postgres DB_NAME=ad_platform RETENTION_DAYS=7 /data/GoPlatform/scripts/backup_postgres.sh >> /var/log/ad-platform-backup.log 2>&1

# 每天 02:30 MySQL 全量+结构备份，保留 7 天
30 2 * * * MYSQL_PWD=__REPLACE_ME__ BACKUP_DIR=/data/backups/mysql DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_NAME=ad_platform RETENTION_DAYS=7 /data/GoPlatform/scripts/backup_mysql.sh >> /var/log/ad-platform-backup.log 2>&1

# 每天 03:10 预览待清理分区（PostgreSQL）
10 3 * * * psql -U postgres -d ad_platform -c "SELECT * FROM cleanup_old_ad_reports_partitions(180, true);" >> /var/log/ad-platform-partition-cleanup.log 2>&1

# 每周日 03:20 执行分区清理（PostgreSQL）
20 3 * * 0 psql -U postgres -d ad_platform -c "SELECT * FROM cleanup_old_ad_reports_partitions(180, false);" >> /var/log/ad-platform-partition-cleanup.log 2>&1
```

### 14.5 恢复演练建议

- 至少每月一次在测试环境做恢复演练（从备份文件恢复并验证关键接口）。
- 演练后记录：恢复耗时、是否成功、失败原因、改进项。
- 生产变更前，先确认最近 24 小时备份文件存在且可解压读取。

## 15. 服务器规格建议（按部署形态）

### 15.1 仅部署主服务（数据库/中间件外置）

- 最低可用：`2核 2G内存 40G SSD`（低流量）
- 推荐起步：`2核 4G内存 80G SSD`
- 适用场景：内网系统、验证环境、小规模生产

### 15.2 全部同机部署（主服务+数据库+Redis+Pulsar+Nginx+Prometheus）

- 最低可用（仅测试/低流量）：`4核 8G内存 100G SSD`
- 推荐生产起步：`8核 16G内存 200G SSD`
- 有增长预期：`8核 32G内存 300G SSD`

说明：

- `2核2G` 不建议同机部署全家桶，容易出现 OOM、超时、频繁重启。
- 若必须小规格上线，优先外置数据库与消息组件（Pulsar）。
- 强烈建议启用 swap（至少 1G，建议 2G）。

## 16. 2核2G 专用推荐参数（轻量化）

适用于“主服务部署在 2核2G，数据库尽量外置”的场景。

### 16.1 `ad-platform-main.service` 推荐

保留以下关键项：

```ini
Environment="GOMAXPROCS=1"
Environment="GOGC=100"
Environment="GODEBUG=madvdontneed=1"
Restart=always
RestartSec=5
```

### 16.2 `config/config.yaml` 推荐

数据库连接池：

```yaml
database:
  max_open_conns: 8
  max_idle_conns: 4
  conn_max_lifetime: 20m
  conn_max_idle_time: 5m
```

Redis 连接池：

```yaml
redis:
  pool_size: 20
  min_idle_conns: 5
  dial_timeout: 2s
  read_timeout: 1s
  write_timeout: 1s
  idle_timeout: 3m
  max_retries: 2
  min_retry_backoff: 100ms
  max_retry_backoff: 500ms
```

限流建议：

```yaml
rate_limit:
  requests_per_second: 20
  burst: 80
```

### 16.3 资源告警阈值建议

- 可用内存持续 `< 200MB`：立即降限流并排查
- CPU 持续 `> 80%` 超过 10 分钟：降并发或升配
- 日志出现 OOM / timeout：优先降低连接池与 QPS

## 17. 上线前/后超级详细检查清单（建议打印执行）

### 17.1 上线前检查（Pre-Flight）

- [ ] `DB_DRIVER` 与数据库类型一致（mysql/postgres）
- [ ] `DB_PORT` 与驱动一致（3306/5432）
- [ ] 初始化脚本与驱动一致：
  - MySQL：`sql/init_db_mysql.sql`
  - PostgreSQL：`sql/init_db_PostgreSQL.sql`
- [ ] `JWT_SECRET` / `AES_SECRET_KEY` 已替换强随机值
- [ ] `AUTH_ALLOW_PUBLIC_REGISTER=false`
- [ ] `pwsh ./scripts/check-secrets.ps1` 扫描通过
- [ ] 备份脚本可执行且手工执行一次成功
- [ ] 最近 24 小时备份文件存在且可解压

### 17.2 上线执行检查

```bash
sudo systemctl daemon-reload
sudo systemctl restart ad-platform-main
sudo systemctl status ad-platform-main --no-pager
curl -f http://127.0.0.1:8960/health
sudo journalctl -u ad-platform-main -n 200 --no-pager
```

### 17.3 上线后 30 分钟观察

- [ ] `/health` 连续可用
- [ ] 无持续数据库连接错误
- [ ] 无 OOM / 频繁重启
- [ ] 内存、CPU、磁盘在可控范围
- [ ] 业务核心接口抽样通过

### 17.4 回退触发条件（建议）

满足任一条件建议立即回退：

- 服务 10 分钟内连续重启 3 次以上
- 核心接口持续不可用超过 5 分钟
- 数据库连接持续异常且 15 分钟内无法恢复

执行回退：

```bash
sudo systemctl stop ad-platform-main
sudo cp /data/GoPlatform/ad-platform-main.backup /data/GoPlatform/ad-platform-main
sudo chmod +x /data/GoPlatform/ad-platform-main
sudo systemctl start ad-platform-main
```
