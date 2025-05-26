package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"smart_parking/internal/domain"
	"smart_parking/internal/service"

	"github.com/gin-gonic/gin"
)

type GateEventHandler struct {
	iotService     *service.IoTService
	lprService     *service.LPRService
	parkingService *service.ParkingService
}

func NewGateEventHandler(iotService *service.IoTService, lprService *service.LPRService, parkingService *service.ParkingService) *GateEventHandler {
	return &GateEventHandler{
		iotService:     iotService,
		lprService:     lprService,
		parkingService: parkingService,
	}
}

// POST /api/v1/gate-events/lpr-trigger
func (h *GateEventHandler) TriggerLPR(c *gin.Context) {
	var request domain.LPRTriggerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	// Nếu có manual override, sử dụng luôn
	if request.ManualOverride != "" {
		err := h.iotService.ProcessLPRResult(c.Request.Context(), request, request.ManualOverride, 1.0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xử lý manual override", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":        "Đã xử lý manual override thành công",
			"detected_plate": request.ManualOverride,
			"confidence":     1.0,
			"is_manual":      true,
		})
		return
	}

	// Xử lý LPR với ảnh
	imageBytes, err := base64DecodeImage(request.ImageBase64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu ảnh không hợp lệ"})
		return
	}

	detectedPlate, confidence, err := h.lprService.ProcessImageForLPR(c.Request.Context(), imageBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xử lý LPR", "details": err.Error()})
		return
	}

	// Xử lý kết quả LPR
	err = h.iotService.ProcessLPRResult(c.Request.Context(), request, detectedPlate, confidence)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xử lý kết quả LPR", "details": err.Error()})
		return
	}

	response := gin.H{
		"message":        "LPR đã được xử lý thành công",
		"detected_plate": detectedPlate,
		"confidence":     confidence,
		"is_manual":      false,
	}

	// Thông tin thêm về việc tạo session
	if confidence >= 0.8 && detectedPlate != "" {
		response["auto_session_created"] = true
		response["next_action"] = "Session sẽ được tạo tự động"
	} else if detectedPlate != "" {
		response["auto_session_created"] = false
		response["next_action"] = "Cần xác nhận manual do confidence thấp"
		response["requires_confirmation"] = true
	} else {
		response["auto_session_created"] = false
		response["next_action"] = "Không nhận dạng được biển số, cần nhập manual"
		response["requires_manual_input"] = true
	}

	c.JSON(http.StatusOK, response)
}

// POST /api/v1/gate-events/create-session
func (h *GateEventHandler) CreateSessionFromEvent(c *gin.Context) {
	var request domain.SessionCreationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	// Tạo session thông qua parking service
	sessionDTO := domain.VehicleCheckInDTO{
		LotID:             request.LotID,
		Esp32ThingName:    request.Esp32ThingName,
		VehicleIdentifier: request.DetectedPlate,
	}

	session, err := h.parkingService.VehicleCheckIn(c.Request.Context(), sessionDTO)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phiên đỗ xe", "details": err.Error()})
		return
	}

	// TODO: Cập nhật gate event record với session ID
	// h.iotService.UpdateGateEventWithSession(c.Request.Context(), request.EventID, session.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Phiên đỗ xe đã được tạo thành công",
		"session":    session,
		"event_id":   request.EventID,
		"created_by": "gate_event_trigger",
	})
}

// GET /api/v1/gate-events/pending
func (h *GateEventHandler) GetPendingGateEvents(c *gin.Context) {
	// TODO: Implement lấy danh sách pending gate events
	// Có thể dùng cho fallback nếu WebSocket không hoạt động
	c.JSON(http.StatusOK, gin.H{
		"message": "Feature đang được phát triển",
		"events":  []interface{}{},
	})
}

func base64DecodeImage(base64Str string) ([]byte, error) {
	// Remove data URL prefix nếu có
	if len(base64Str) > 22 && base64Str[:22] == "data:image/jpeg;base64," {
		base64Str = base64Str[23:]
	} else if len(base64Str) > 21 && base64Str[:21] == "data:image/png;base64," {
		base64Str = base64Str[22:]
	}

	imageBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("lỗi decode base64: %w", err)
	}

	if len(imageBytes) == 0 {
		return nil, fmt.Errorf("dữ liệu ảnh rỗng")
	}

	return imageBytes, nil
}
