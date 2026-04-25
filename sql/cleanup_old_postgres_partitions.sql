-- PostgreSQL 旧分区清理函数（ad_reports）
-- 用法：
--   1) 预览将被删除的分区（不执行删除）
--      SELECT * FROM cleanup_old_ad_reports_partitions(180, true);
--   2) 执行删除
--      SELECT * FROM cleanup_old_ad_reports_partitions(180, false);

CREATE OR REPLACE FUNCTION cleanup_old_ad_reports_partitions(
    p_keep_days INTEGER DEFAULT 180,
    p_dry_run BOOLEAN DEFAULT true
)
RETURNS TABLE(partition_name TEXT, partition_start DATE, dropped BOOLEAN)
LANGUAGE plpgsql
AS $$
DECLARE
    r RECORD;
    part_from DATE;
    drop_before DATE := CURRENT_DATE - p_keep_days;
    part_expr TEXT;
BEGIN
    FOR r IN
        SELECT c.relname AS child_name,
               pg_get_expr(c.relpartbound, c.oid, true) AS part_bound
        FROM pg_inherits i
        JOIN pg_class p ON p.oid = i.inhparent
        JOIN pg_class c ON c.oid = i.inhrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE p.relname = 'ad_reports'
          AND n.nspname = 'public'
          AND c.relname LIKE 'ad_reports_y%'
        ORDER BY c.relname
    LOOP
        part_expr := r.part_bound;
        part_from := NULL;

        -- 从分区边界表达式中提取 FROM 日期，例如:
        -- FOR VALUES FROM ('2026-03-01') TO ('2026-03-02')
        BEGIN
            part_from := substring(part_expr FROM $$FROM \('([0-9]{4}-[0-9]{2}-[0-9]{2})'\)$$)::date;
        EXCEPTION WHEN OTHERS THEN
            part_from := NULL;
        END;

        IF part_from IS NOT NULL AND part_from < drop_before THEN
            IF p_dry_run THEN
                partition_name := r.child_name;
                partition_start := part_from;
                dropped := false;
                RETURN NEXT;
            ELSE
                EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', r.child_name);
                partition_name := r.child_name;
                partition_start := part_from;
                dropped := true;
                RETURN NEXT;
            END IF;
        END IF;
    END LOOP;
END;
$$;
