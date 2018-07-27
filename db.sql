-- 日志记录表
CREATE table log_operation(
id serial NOT NULL PRIMARY KEY,
operation_id VARCHAR NOT NULL DEFAULT '', -- 操作对象ID
operation_table VARCHAR NOT NULL DEFAULT '', -- 操作表
operation_type SMALLINT NOT NULL DEFAULT 0, -- 操作类型 1:查询 2:新增 3:编辑 4:删除
operation_ip VARCHAR NOT NULL DEFAULT '', -- ip
comment VARCHAR NOT NULL DEFAULT '', -- 描述
request_info jsonb NOT NULL DEFAULT '{}', -- 请求信息
column_info jsonb NOT NULL DEFAULT '[]', -- 列变更信息
user_id  INT NOT NULL DEFAULT 0,       -- 用户id
user_role VARCHAR NOT NULL DEFAULT '', -- 用户角色
add_time  TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP -- 添加时间
);
-- 测试业务表
CREATE table notice(
id serial NOT NULL PRIMARY KEY,
notice_name VARCHAR(100) NOT NULL DEFAULT '', -- 公告名
notice_content TEXT NOT NULL DEFAULT '', -- 公告内容
notice_type SMALLINT NOT NULL DEFAULT 1, -- 公告类型 1:用户公告 2:代理公告
status SMALLINT NOT NULL DEFAULT 0, --  公告状态 0:关闭 1:开启
add_time  TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP -- 添加时间
);
-- 给测试业务表添加comment
COMMENT ON TABLE notice IS '公告';
COMMENT ON COLUMN notice.notice_name IS '公告名';
COMMENT ON COLUMN notice.notice_content IS '公告内容';
COMMENT ON COLUMN notice.notice_type IS '公告类型';
COMMENT ON COLUMN notice.status IS '公告状态';
COMMENT ON COLUMN notice.add_time IS '添加时间';