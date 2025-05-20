package handler

import (
	"net/http"
	"smart_parking/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IoTCommandHandler struct {
	iotService *service.IoTService
}

func NewIoTCommandHandler(is *service.IoTService) *IoTCommandHandler {
	return &IoTCommandHandler{iotService: is}
}

type ControlBarrierRequest struct {
	Esp32ControllerID string `json:"esp32_controller_id" binding:"required"`           // Thing Name của ESP32
	BarrierType       string `json:"barrier_type" binding:"required,oneof=entry exit"` // "entry" hoặc "exit"
	Command           string `json:"command" binding:"required,oneof=open close"`      // "open" hoặc "close"
}

// POST /iot/commands/barrier
func (h *IoTCommandHandler) ControlBarrier(c *gin.Context) {
	var req ControlBarrierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requestID := uuid.New().String()

	err := h.iotService.SendBarrierControlCommand(c.Request.Context(), req.Esp32ControllerID, req.BarrierType, req.Command, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể gửi lệnh điều khiển rào chắn", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lệnh điều khiển rào chắn đã được gửi", "request_id": requestID})
}
