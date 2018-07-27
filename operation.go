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
