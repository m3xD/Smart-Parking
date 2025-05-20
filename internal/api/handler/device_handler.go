package handler

import (
	"net/http"
	// "parking_system_go/internal/domain"
	"smart_parking/internal/repository"
	"smart_parking/internal/service"
	// "strconv"

	"errors"
	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	parkingService *service.ParkingService // Device logic hiện đang nằm trong ParkingService
}

func NewDeviceHandler(ps *service.ParkingService) *DeviceHandler {
	return &DeviceHandler{parkingService: ps}
}

// GET /devices
func (h *DeviceHandler) GetAllDevices(c *gin.Context) {
	devices, err := h.parkingService.GetAllDevices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách thiết bị", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, devices)
}

// GET /devices/:thing_name
func (h *DeviceHandler) GetDeviceByThingName(c *gin.Context) {
	thingName := c.Param("thing_name")
	if thingName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thing name không được để trống"})
		return
	}
	device, err := h.parkingService.GetDeviceByThingName(c.Request.Context(), thingName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thiết bị"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin thiết bị", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, device)
}
