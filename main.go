package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var logger = logrus.New()

func initializeDB() {
	var err error
	dsn := "host=localhost user=postgres password=Priyajit@2002 dbname=devices port=5432 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}
}

type Device struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	DeviceName   string `gorm:"column:device_name" json:"device_name"`
	DeviceType   string `gorm:"column:device_type" json:"device_type"`
	Brand        string `gorm:"column:brand" json:"brand"`
	Model        string `gorm:"column:model" json:"model"`
	Os           string `gorm:"column:os" json:"os"`
	OsVersion    string `gorm:"column:os_version" json:"os_version"`
	PurchaseDate string `gorm:"column:purchase_date" json:"purchase_date"`
	WarrantyEnd  string `gorm:"column:warranty_end" json:"warranty_end"`
	Status       string `gorm:"column:status" json:"status"`
	Price        uint   `gorm:"column:price" json:"price"`
}

func main() {
	setupRouter()
	setupLogger() // Initialize the logger
	initializeDB()

	r := gin.Default()
	r.POST("/device", registerDevice)
	r.PUT("/device/:id", updateDevice)
	r.GET("/device", listDevices)
	r.GET("/device/:id", getDeviceByID)
	r.DELETE("/device/:id", deleteDevice)
	r.POST("/upload", uploadCSV)
	r.GET("/logs", getLogs)

	logger.Info("Starting server on port 8080")
	if err := r.Run(":8080"); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
