package main

import (
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/device", registerDevice)
	r.PUT("/device/:id", updateDevice)
	r.GET("/device", listDevices)
	r.GET("/device/:id", getDeviceByID)
	r.DELETE("/device/:id", deleteDevice)
	r.POST("/upload", uploadCSV)
	r.GET("/logs", getLogs)

	return r
}
