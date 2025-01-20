package main

import (
	"bufio"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const chunkSize = 1000

func registerDevice(c *gin.Context) {
	var device Device
	if err := c.ShouldBindJSON(&device); err != nil {
		logger.Warnf("Invalid input: %v", err)
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := db.Create(&device).Error; err != nil {
		logger.Errorf("Failed to register device: %v", err)
		respondWithError(c, http.StatusInternalServerError, "Failed to register device")
		return
	}

	logger.Infof("Device registered: %v", device)
	c.JSON(http.StatusCreated, device)
}

func updateDevice(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		logger.Warnf("Invalid ID format: %v", err)
		respondWithError(c, http.StatusBadRequest, "Invalid ID format")
		return
	}

	var device Device
	if err := c.ShouldBindJSON(&device); err != nil {
		logger.Warnf("Invalid input: %v", err)
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	result := db.Model(&Device{}).Where("id = ?", idInt).Updates(device)
	if result.Error != nil {
		logger.Errorf("Failed to update device: %v", result.Error)
		respondWithError(c, http.StatusInternalServerError, "Failed to update device")
		return
	}

	if result.RowsAffected == 0 {
		logger.Warnf("Device not found for ID: %d", idInt)
		respondWithError(c, http.StatusNotFound, "Device not found")
		return
	}

	logger.Infof("Device updated: %v", device)
	c.JSON(http.StatusOK, gin.H{"message": "Device updated successfully"})
}

func listDevices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var devices []Device
	if err := db.Limit(limit).Offset(offset).Find(&devices).Error; err != nil {
		logger.Errorf("Failed to retrieve devices: %v", err)
		respondWithError(c, http.StatusInternalServerError, "Failed to retrieve devices")
		return
	}

	logger.Infof("Devices retrieved: %d", len(devices))
	c.JSON(http.StatusOK, devices)
}

func getDeviceByID(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		logger.Warnf("Invalid ID format: %v", err)
		respondWithError(c, http.StatusBadRequest, "Invalid ID format")
		return
	}

	var device Device
	if err := db.First(&device, idInt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnf("Device not found for ID: %d", idInt)
			respondWithError(c, http.StatusNotFound, "Device not found")
		} else {
			logger.Errorf("Failed to retrieve device: %v", err)
			respondWithError(c, http.StatusInternalServerError, "Failed to retrieve device")
		}
		return
	}

	logger.Infof("Device retrieved: %v", device)
	c.JSON(http.StatusOK, device)
}

func deleteDevice(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		logger.Warnf("Invalid ID format: %v", err)
		respondWithError(c, http.StatusBadRequest, "Invalid ID format")
		return
	}

	result := db.Delete(&Device{}, idInt)
	if result.Error != nil {
		logger.Errorf("Failed to delete device: %v", result.Error)
		respondWithError(c, http.StatusInternalServerError, "Failed to delete device")
		return
	}

	if result.RowsAffected == 0 {
		logger.Warnf("Device not found for ID: %d", idInt)
		respondWithError(c, http.StatusNotFound, "Device not found")
		return
	}

	logger.Infof("Device deleted with ID: %d", idInt)
	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}

func uploadCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		logger.Warnf("File upload error: %v", err)
		respondWithError(c, http.StatusBadRequest, "File is required")
		return
	}

	src, err := file.Open()
	if err != nil {
		logger.Errorf("Failed to open file: %v", err)
		respondWithError(c, http.StatusInternalServerError, "Failed to open file")
		return
	}
	defer src.Close()

	var wg sync.WaitGroup
	recordChannel := make(chan string, 10000) // Channel to hold raw CSV lines
	batchChannel := make(chan []Device, 100)  // Channel to hold processed Device batches

	// Worker pool for processing batches
	numWorkers := 10 // Number of workers for batch processing
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range batchChannel {
				if len(batch) > 0 {
					processBatch(batch)
				}
			}
		}()
	}

	// Goroutine to read file and feed records to the recordChannel
	go func() {
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			recordChannel <- scanner.Text()
		}
		close(recordChannel)
		if err := scanner.Err(); err != nil {
			logger.Errorf("Error reading file: %v", err)
		}
	}()

	// Goroutine to group records into batches and send to batchChannel
	go func() {
		var batch []Device
		for record := range recordChannel {
			data := strings.Split(record, ",")
			if len(data) < 10 {
				logger.Warnf("Skipping invalid record: %s", record)
				continue
			}
			device := Device{
				DeviceName:   data[0],
				DeviceType:   data[1],
				Brand:        data[2],
				Model:        data[3],
				Os:           data[4],
				OsVersion:    data[5],
				PurchaseDate: data[6],
				WarrantyEnd:  data[7],
				Status:       data[8],
				Price:        uint(atoiSafe(data[9])),
			}
			batch = append(batch, device)

			if len(batch) >= chunkSize {
				batchChannel <- batch
				batch = nil // Reset batch
			}
		}

		// Send remaining batch if any
		if len(batch) > 0 {
			batchChannel <- batch
		}
		close(batchChannel)
	}()

	wg.Wait()

	logger.Info("CSV uploaded and processed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "CSV uploaded and processed successfully"})
}

func processBatch(batch []Device) {
	// Bulk insert for efficiency
	if err := db.Create(&batch).Error; err != nil {
		logger.Errorf("Error inserting batch: %v", err)
	}
}

func atoiSafe(str string) int {
	value, _ := strconv.Atoi(str)
	return value
}

func respondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}

func getLogs(c *gin.Context) {
	logger.Info("Log retrieval endpoint hit")
	c.JSON(http.StatusOK, gin.H{"message": "Logs endpoint under construction"})
}
