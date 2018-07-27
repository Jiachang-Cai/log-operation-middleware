思路借鉴  [浅谈管理系统操作日志设计](https://blog.csdn.net/sd4015700/article/details/47397289)

### 说明
采用中间件的方式 纯属原创  请转载标明来处 谢谢
该方案目前只支持单条数据的操作
### 需求
1.需要去记录后台管理系统的 一些增删改敏感操作 能够查询到 操作详情
以及每个字段的新旧值
2. 不能去影响现有的业务代码
### 实现思路
因为 不能去影响已经编写好的业务代码 那只能去通过中间件的方式去实现 而我又要去拿到 表名和操作对象ID
1. 通过restful 机制 的method 去 区分增删改  如 POST 为增加操作
PUT 为修改操作 DELETE 为删除操作
2. 通过gin 框架的 Param 机制  去或者 操作对象的ID 
3. 通过gin handler 名 去 映射 表名
4. 通过表名和操作对象ID 去获取数据库的comment

| 操作 | 说明 |
| --- | --- |
| INSERT | 在INSERT后执行 |
| UPDATE |  在UPDATE前后都要执行，操作前获取操作前数据，操作后获取操作后数据 |
| DELETE | 在DELETE前执行  |
### 实现环境
- DB:postgresql
- 框架: gin
- 语言: go

### 准备工作
#### 创建一张日志记录表和测试业务表并添加comment

```sql
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
```
#### 编写main.go

```go
package main

import (
	"github.com/gin-gonic/gin"
	"database/sql"
	_ "github.com/lib/pq"
	"fmt"
)

var Db *sql.DB

func init()  {
	// 初始化数据库
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=123456 dbname=test sslmode=disable")
	if err != nil {
		fmt.Println(err)
	}
	Db = db
}

func main() {
	r := gin.Default()
	// handler 和 表名的映射
	handleTableName := map[string]string{
		"AddNotice":  "notice",
		"EditNotice": "notice",
		"DelNotice":  "notice",
	}
	r.POST("/notices/", AddNotice)
	r.PUT("/notices/:pk/", EditNotice)
	r.DELETE("/notices/:pk/", DelNotice)

	r.Use(Operation(handleTableName, Db))
	r.Run()
}

```
#### 编写handles.go

```go
package main

import (
	"github.com/gin-gonic/gin"
	"fmt"
)

func AddNotice(c *gin.Context) {
	var Id int64
	if err := Db.QueryRow("INSERT INTO notice(notice_name) VALUES($1) RETURNING id;", "测试增加").Scan(&Id); err != nil {
		fmt.Println(err)
		c.String(400, "server error")
		return
	}
	// 由于 新增操作 需要插入数据库后才能知道对象ID 在获取对象ID 后 需要传递给中间件
	c.Set("pk", Id)
	c.String(200, "success")

}

func EditNotice(c *gin.Context) {
	pk := c.Param("pk")
	if _, err := Db.Exec("UPDATE notice set notice_name=$1 WHERE id=$2", "测试修改", pk); err != nil {
		fmt.Println(err)
		c.String(400, "server error")
		return
	}
	c.String(200, "success")
}

func DelNotice(c *gin.Context) {
	pk := c.Param("pk")
	if _, err := Db.Exec("DELETE FORM notice WHERE id=$1", pk); err != nil {
		fmt.Println(err)
		c.String(400, "server error")
		return
	}
	c.String(200, "success")
}
```

#### 编写operation.go 中间件

```
package main

import (
	"github.com/gin-gonic/gin"
	"fmt"
	"time"
	"strings"
	"log"
	"encoding/json"
	"net/http"
	"io/ioutil"
	"bytes"
	"database/sql"
)

type operation struct {
	DB *sql.DB
}

// Operation 操作日志中间件
// 1:查询 2:新增 3:编辑 4:删除
func Operation(handlerTableName map[string]string, DB *sql.DB) gin.HandlerFunc {
	opt := &operation{
		DB: DB,
	}
	return func(c *gin.Context) {
		// 获取当前用户ID 可通过jwt 中间件获取
		userId, ok := c.Get("userId")
		if !ok {
			userId = 0
		}
		handlerName := strings.Split(c.HandlerName(), ".")[1]

		switch c.Request.Method {
		case "PUT":
			// Read the Body content
			var bodyBytes []byte
			if c.Request.Body != nil {
				bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
			}
			body := strings.Join(strings.Fields(string(bodyBytes)), "")
			// Restore the io.ReadCloser to its original state
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			// 获取修改对象ID
			if c.Param("pk") != "" {
				// 根据HandlerName 映射表名
				if tableName, ok := handlerTableName[handlerName]; ok {

					dataOld, err := opt.getData(fmt.Sprintf(`select * from %s where id=$1`, tableName), c.Param("pk"))
					if err != nil && len(dataOld) > 0 {
						log.Println(err)
					} else {
						c.Set("operation", map[string]interface{}{"tableName": tableName,
							"pk": c.Param("pk"), "dataOld": dataOld[0]})
					}
				}

			}
			c.Next()
			if c.Writer.Status() != 200 {
				c.Abort()
				return
			}
			// 获取修改对象
			operation, ok := c.Get("operation")
			if !ok {
				c.Abort()
				return
			}
			item := operation.(map[string]interface{})
			tableName := item["tableName"].(string)
			dataNow, err := opt.getData(fmt.Sprintf(`select * from %s where id=$1`, tableName), item["pk"])
			if err != nil {
				log.Println(err)
				c.Abort()
				return
			}
			colComment := make(map[string]interface{})
			var tbComment string
			// 查询表注解
			if err := opt.getTbComment(tableName, &tbComment); err != nil {
				log.Println(err)
				c.Abort()
				return
			}
			// 查询列注解
			if err := opt.getColComment(tableName, colComment); err != nil {
				log.Println(err)
				c.Abort()
				return
			}
			colInfo := make([]map[string]interface{}, 0)
			if len(dataNow) > 0 {
				// 进行对比
				for k, v := range dataNow[0] {
					oldValue := item["dataOld"].(map[string]interface{})[k]
					if v != oldValue {
						entry := make(map[string]interface{})
						entry["col_name"] = k
						entry["comment"] = colComment[k]
						entry["old_value"] = oldValue
						entry["new_value"] = v
						colInfo = append(colInfo, entry)
					}
				}
				colInfoJson, _ := json.Marshal(colInfo)
				// 日志记录
				if err := opt.insertLog(item["pk"], userId, tableName, c.ClientIP(), tbComment,
					opt.getRequestJson(c.Request, body), string(colInfoJson), 2); err != nil {
					log.Println(err)
					c.Abort()
					return
				}

			}

		case "POST":
			// Read the Body content
			var bodyBytes []byte
			if c.Request.Body != nil {
				bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
			}
			body := strings.Join(strings.Fields(string(bodyBytes)), "")
			// Restore the io.ReadCloser to its original state
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Next()
			if c.Writer.Status() != 200 {
				c.Abort()
				return
			}
			pk, ok := c.Get("pk")
			if !ok {
				c.Abort()
				return
			}
			// 根据HandlerName 映射表名
			tableName, ok := handlerTableName[handlerName]
			if !ok {
				c.Abort()
				return
			}
			// 查询对象
			dataNow, err := opt.getData(fmt.Sprintf(`select * from %s where id=$1`, tableName), pk)
			if err != nil {
				log.Println(err)
				c.Abort()
				return
			}
			if len(dataNow) > 0 {
				colComment := make(map[string]interface{})
				var tbComment string
				// 查询表注解
				if err := opt.getTbComment(tableName, &tbComment); err != nil {
					log.Println(err)
					c.Abort()
					return
				}
				// 查询列注解
				if err := opt.getColComment(tableName, colComment); err != nil {
					log.Println(err)
					c.Abort()
					return
				}
				colInfo := make([]map[string]interface{}, 0)
				for k, v := range dataNow[0] {
					entry := make(map[string]interface{})
					entry["col_name"] = k
					entry["comment"] = colComment[k]
					entry["old_value"] = v
					entry["new_value"] = ""
					colInfo = append(colInfo, entry)
				}
				colInfoJson, _ := json.Marshal(colInfo)
				// 日志记录
				if err := opt.insertLog(pk, userId, tableName, c.ClientIP(), tbComment,
					opt.getRequestJson(c.Request, body), string(colInfoJson), 3); err != nil {
					log.Println(err)
					c.Abort()
					return
				}
			}
		case "DELETE":
			if c.Param("pk") != "" {
				// 根据HandlerName 映射表名
				if tableName, ok := handlerTableName[handlerName]; ok {

					dataOld, err := opt.getData(fmt.Sprintf(`select * from %s where id=$1`, tableName), c.Param("pk"))
					if err != nil && len(dataOld) > 0 {
						log.Println(err)
					} else {
						c.Set("operation", map[string]interface{}{"tableName": tableName,
							"pk": c.Param("pk"), "dataOld": dataOld[0]})
					}
				}

			}
			c.Next()
			if c.Writer.Status() != 200 {
				c.Abort()
				return
			}
			// 获取修改对象
			operation, ok := c.Get("operation")
			if !ok {
				c.Abort()
				return
			}
			item := operation.(map[string]interface{})
			tableName := item["tableName"].(string)
			colComment := make(map[string]interface{})
			var tbComment string
			// 查询表注解
			if err := opt.getTbComment(tableName, &tbComment); err != nil {
				log.Println(err)
				c.Abort()
				return
			}
			// 查询列注解
			if err := opt.getColComment(tableName, colComment); err != nil {
				log.Println(err)
				c.Abort()
				return
			}
			colInfo := make([]map[string]interface{}, 0)
			for k, v := range item["dataOld"].(map[string]interface{}) {
				entry := make(map[string]interface{})
				entry["col_name"] = k
				entry["comment"] = colComment[k]
				entry["old_value"] = v
				entry["new_value"] = ""
				colInfo = append(colInfo, entry)
			}
			colInfoJson, _ := json.Marshal(colInfo)
			// 日志记录
			if err := opt.insertLog(item["pk"], userId, tableName, c.ClientIP(), tbComment,
				opt.getRequestJson(c.Request, ""), string(colInfoJson), 4); err != nil {
				log.Println(err)
				c.Abort()
				return
			}

		}
	}
}

func (opt *operation) getData(sql string, args ...interface{}) ([]map[string]interface{}, error) {
	data := make([]map[string]interface{}, 0)
	rows, err := opt.DB.Query(sql, args...)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			if b, ok := val.([]byte); ok {
				v = string(b)
			} else if t, ok := val.(time.Time); ok {
				list := strings.Split(fmt.Sprintf("%v", t), " ")
				dateTime := fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
				if len(list) >= 2 && list[1] == "00:00:00" {
					dateTime = fmt.Sprintf("\"%s\"", t.Format("2006-01-02"))
				}
				v = dateTime
			} else {
				v = val
			}

			entry[col] = v
		}
		data = append(data, entry)
	}
	return data, nil
}

// 查询表注释
func (opt *operation) getTbComment(tbName string, tbComment *string) error {
	if err := opt.DB.QueryRow(fmt.Sprintf(`select
		COALESCE(obj_description('%s'::regclass),'');`, tbName)).Scan(tbComment); err != nil {
		return err
	}
	return nil
}

// 查询字段注释
func (opt *operation) getColComment(tbName string, colComment map[string]interface{}) error {
	sql := `
	SELECT
    cols.column_name,
	COALESCE((
        SELECT
            pg_catalog.col_description(c.oid, cols.ordinal_position::int)
        FROM pg_catalog.pg_class c
        WHERE
            c.oid     = (SELECT cols.table_name::regclass::oid) AND
            c.relname = cols.table_name
    ),'')
     as column_comment

	FROM information_schema.columns cols
	WHERE
		cols.table_name='%s';
	`

	datas, err := opt.getData(fmt.Sprintf(sql, tbName))
	if err != nil {
		return err
	}
	for _, item := range datas {
		colComment[fmt.Sprintf("%s", item["column_name"])] = item["column_comment"]
	}
	return nil
}

// 日志记录
func (opt *operation) insertLog(operationId, userId interface{}, operationTable, operationIp,
comment, requestInfo, columnInfo string, operationType int) error {
	if _, err := opt.DB.Exec(`insert into log_operation(operation_id,
		operation_table,operation_type,operation_ip,comment,request_info,column_info,user_id) 
		values($1,$2,$3,$4,$5,$6,$7,$8);
	`, operationId, operationTable, operationType, operationIp,
		comment, requestInfo, columnInfo, userId); err != nil {
		return err
	}

	return nil

}

// 获取请求相关参数json
func (opt *operation) getRequestJson(req *http.Request, body string) string {
	proto := "http"
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		proto = "https"
	}
	data := map[string]interface{}{
		"Method":  req.Method,
		"Cookies": req.Header.Get("Cookie"),
		"Query":   req.URL.Query().Encode(),
		"URL":     proto + "://" + req.Host + req.URL.Path,
		"Headers": make(map[string]string, len(req.Header)),
	}
	data["Host"] = req.Host
	for k, v := range req.Header {
		data["Headers"].(map[string]string)[k] = strings.Join(v, ",")
	}
	data["Body"] = body
	jsonStr, _ := json.Marshal(data)
	return string(jsonStr)
}
```

4. 显示log_operation 数据示例
![image.png](https://upload-images.jianshu.io/upload_images/6513868-b2737cdd38c746db.png?imageMogr2/auto-orient/strip%7CimageView2/2/w/1240)

![image.png](https://upload-images.jianshu.io/upload_images/6513868-ca9ebf8f3c5b8a6d.png?imageMogr2/auto-orient/strip%7CimageView2/2/w/1240)


