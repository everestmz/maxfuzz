// Base stats server

package main

import (
  "maxfuzz/fuzzer-base/internal/helpers"

  "github.com/gin-gonic/gin"
)

var log = helpers.BasicLogger()

func main() {
  helpers.QuickLog(log, "Starting fuzzer stats server")
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run()
}
