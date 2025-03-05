package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/zcalusic/sysinfo"
)

func main() {
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	if host == "" {
		host = "localhost"
	}

	if port == "" {
		port = "9000"
	}

	var si sysinfo.SysInfo
	si.GetSysInfo()

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, &si)
	})
	r.Run(host + ":" + port)
}
