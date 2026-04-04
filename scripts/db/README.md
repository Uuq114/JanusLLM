# 数据库脚本说明

当前目录只保留一份 PostgreSQL 建表脚本：

- `create_core_tables.sql`

## 执行示例（psql）

```bash
psql "$DATABASE_URL" -f scripts/db/create_core_tables.sql
```

## 约定

- 现阶段先维护单个建表脚本。
- 后续如果网关涉及较大表结构更新，再补充独立迁移脚本。
