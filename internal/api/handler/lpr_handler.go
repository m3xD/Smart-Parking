package handler

import (
	"encoding/base64"
	"log"
	"net/http"
	"smart_parking/internal/domain"
	"smart_parking/internal/service"

	"github.com/gin-gonic/gin"
)

type LPRHandler struct {
	lprService     *service.LPRService
	parkingService *service.ParkingService // Có thể cần để tạo session sau khi LPR
}

func NewLPRHandler(lprService *service.LPRService, parkingService *service.ParkingService) *LPRHandler {
	return &LPRHandler{lprService: lprService, parkingService: parkingService}
}

// POST /api/v1/lpr/process-image
func (h *LPRHandler) ProcessImage(c *gin.Context) {
	var req domain.LPRRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload không hợp lệ: " + err.Error()})
		return
	}

	// Giải mã ảnh base64
	imageBytes, err := base64.StdEncoding.DecodeString(req.ImageBase64)
	if err != nil {
		log.Printf("LPRHandler: Lỗi giải mã ảnh base64: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu ảnh không hợp lệ"})
		return
	}

	if len(imageBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu ảnh rỗng"})
		return
	}
	log.Printf("LPRHandler: Đã nhận %d bytes ảnh để xử lý LPR.", len(imageBytes))

	detectedPlate, confidence, err := h.lprService.ProcessImageForLPR(c.Request.Context(), imageBytes)
	if err != nil {
		log.Printf("LPRHandler: Lỗi từ LPRService: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xử lý ảnh LPR", "details": err.Error()})
		return
	}

	if detectedPlate == "" {
		c.JSON(http.StatusOK, domain.LPRResponseDTO{
			DetectedPlate: "",
			ErrorMessage:  "Không nhận dạng được biển số.",
		})
		return
	}

	// Tùy chọn: Nếu muốn tự động tạo parking session ngay sau khi LPR thành công
	// (Cần thêm logic lấy lot_id, esp32_thing_name từ request hoặc ngữ cảnh)
	// entryTime := time.Now()
	// sessionDTO := domain.CreateParkingSessionDTO {
	//     LotID: 1, // Ví dụ
	//     Esp32ThingName: "ESP32_EntryGate_Sim", // Ví dụ
	//     VehicleIdentifier: detectedPlate,
	//     EntryTime: entryTime,
	// }
	// _, sessionErr := h.parkingService.StartParkingSessionFromLPR(c.Request.Context(), sessionDTO) // Cần tạo hàm này
	// if sessionErr != nil {
	//     log.Printf("LPRHandler: Lỗi khi tạo parking session sau LPR: %v", sessionErr)
	// // Vẫn trả về biển số, nhưng có thể kèm thông báo lỗi tạo session
	// }

	c.JSON(http.StatusOK, domain.LPRResponseDTO{
		DetectedPlate: detectedPlate,
		Confidence:    confidence,
	})
}
