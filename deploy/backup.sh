#!/bin/sh
# MyBlog 数据库备份脚本（服务器上运行）
# 用法：./deploy/backup.sh [备份目录]（默认 /opt/myblog/backups）
# 建议加 crontab 每日备份（凌晨 3 点）：
#   0 3 * * * /opt/myblog/deploy/backup.sh >> /var/log/myblog-backup.log 2>&1
set -eu

cd "$(dirname "$0")/.."

BACKUP_DIR="${1:-/opt/myblog/backups}"
KEEP_DAYS=14
STAMP="$(date +%Y%m%d-%H%M%S)"
FILE="$BACKUP_DIR/myblog-$STAMP.sql.gz"

mkdir -p "$BACKUP_DIR"

# 从 .env 读取 root 密码（脚本只在本机使用）
MYSQL_PW="$(sed -n 's/^MYSQL_ROOT_PASSWORD=//p' .env)"
if [ -z "$MYSQL_PW" ]; then
  echo "错误：.env 中未找到 MYSQL_ROOT_PASSWORD" >&2
  exit 1
fi

docker exec myblog-mysql-1 mysqldump -uroot -p"$MYSQL_PW" \
  --single-transaction --routines --triggers myblog 2>/dev/null | gzip > "$FILE"

echo "已备份：$FILE ($(du -h "$FILE" | cut -f1))"

# 清理过期备份
find "$BACKUP_DIR" -name 'myblog-*.sql.gz' -mtime +$KEEP_DAYS -delete
echo "已清理 ${KEEP_DAYS} 天前的旧备份"
