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

type ParkingLotHandler struct {
	parkingService *service.ParkingService
}

func NewParkingLotHandler(ps *service.ParkingService) *ParkingLotHandler {
	return &ParkingLotHandler{parkingService: ps}
}

// POST /parking-lots
func (h *ParkingLotHandler) CreateParkingLot(c *gin.Context) {
	var dto domain.ParkingLotDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lot, err := h.parkingService.CreateParkingLot(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo bãi đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, lot)
}

// GET /parking-lots/:id
func (h *ParkingLotHandler) GetParkingLotByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID bãi đỗ không hợp lệ"})
		return
	}

	lot, err := h.parkingService.GetParkingLotByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy bãi đỗ xe"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin bãi đỗ xe"})
		return
	}
	c.JSON(http.StatusOK, lot)
}

// GET /parking-lots
func (h *ParkingLotHandler) GetAllParkingLots(c *gin.Context) {
	lots, err := h.parkingService.GetAllParkingLots(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách bãi đỗ xe"})
		return
	}
	c.JSON(http.StatusOK, lots)
}

// PUT /parking-lots/:id
func (h *ParkingLotHandler) UpdateParkingLot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID bãi đỗ không hợp lệ"})
		return
	}

	var dto domain.ParkingLotDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lot, err := h.parkingService.UpdateParkingLot(c.Request.Context(), id, dto)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy bãi đỗ xe để cập nhật"})
			return
		}
		if errors.Is(err, repository.ErrDuplicateEntry) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bãi đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, lot)
}

// DELETE /parking-lots/:id
func (h *ParkingLotHandler) DeleteParkingLot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID bãi đỗ không hợp lệ"})
		return
	}

	err = h.parkingService.DeleteParkingLot(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy bãi đỗ xe để xóa"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa bãi đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
