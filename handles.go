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
	if _, err := Db.Exec("DELETE FROM notice WHERE id=$1", pk); err != nil {
		fmt.Println(err)
		c.String(400, "server error")
		return
	}
	c.String(200, "success")
}
