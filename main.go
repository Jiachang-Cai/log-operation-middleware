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
	r.Use(Operation(handleTableName, Db))
	r.POST("/notices/", AddNotice)
	r.PUT("/notices/:pk/", EditNotice)
	r.DELETE("/notices/:pk/", DelNotice)

	r.Run()
}
