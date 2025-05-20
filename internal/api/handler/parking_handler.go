package handler

import (
	"net/http"
	"smart_parking/internal/domain"
	"smart_parking/internal/repository"
	"smart_parking/internal/service"
	"strconv"

	"errors"
	"github.com/gin-gonic/gin"
)

type ParkingSessionHandler struct {
	parkingService *service.ParkingService
}

func NewParkingSessionHandler(ps *service.ParkingService) *ParkingSessionHandler {
	return &ParkingSessionHandler{parkingService: ps}
}

// GET /parking-sessions/:id
func (h *ParkingSessionHandler) GetParkingSessionByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên đỗ xe không hợp lệ"})
		return
	}
	session, err := h.parkingService.GetParkingSessionByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên đỗ xe"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin phiên đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, session)
}

// GET /parking-lots/:lot_id/active-sessions
func (h *ParkingSessionHandler) GetActiveSessionsByLotID(c *gin.Context) {
	lotIDStr := c.Param("lot_id")
	lotID, err := strconv.Atoi(lotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lot ID không hợp lệ"})
		return
	}
	sessions, err := h.parkingService.GetActiveSessionsByLot(c.Request.Context(), lotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách phiên đỗ xe đang hoạt động", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

// GET /parking-sessions (Thêm API để filter sessions)
func (h *ParkingSessionHandler) FindParkingSessions(c *gin.Context) {
	var filter domain.ParkingSessionFilterDTO
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tham số filter không hợp lệ: " + err.Error()})
		return
	}

	sessions, err := h.parkingService.FindParkingSessions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tìm kiếm phiên đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}
