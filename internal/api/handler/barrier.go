// File: internal/api/handler/barrier.go
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

type BarrierHandler struct {
	parkingService *service.ParkingService
}

func NewBarrierHandler(ps *service.ParkingService) *BarrierHandler {
	return &BarrierHandler{parkingService: ps}
}

// POST /barriers
func (h *BarrierHandler) CreateBarrier(c *gin.Context) {
	var dto domain.BarrierDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	barrier, err := h.parkingService.CreateBarrier(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo rào chắn", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, barrier)
}

// GET /barriers/:id
func (h *BarrierHandler) GetBarrierByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rào chắn không hợp lệ"})
		return
	}
	barrier, err := h.parkingService.GetBarrierByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy rào chắn"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin rào chắn"})
		return
	}
	c.JSON(http.StatusOK, barrier)
}

// GET /parking-lots/:lot_id/barriers
func (h *BarrierHandler) GetBarriersByLotID(c *gin.Context) {
	lotIDStr := c.Param("id")
	lotID, err := strconv.Atoi(lotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lot ID không hợp lệ"})
		return
	}
	barriers, err := h.parkingService.GetBarriersByLotID(c.Request.Context(), lotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách rào chắn"})
		return
	}
	c.JSON(http.StatusOK, barriers)
}

// PUT /barriers/:id
func (h *BarrierHandler) UpdateBarrier(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rào chắn không hợp lệ"})
		return
	}
	var dto domain.BarrierDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updatedBarrier, err := h.parkingService.UpdateBarrier(c.Request.Context(), id, dto)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy rào chắn để cập nhật"})
			return
		}
		if errors.Is(err, repository.ErrDuplicateEntry) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật rào chắn", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedBarrier)
}

// DELETE /barriers/:id
func (h *BarrierHandler) DeleteBarrier(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rào chắn không hợp lệ"})
		return
	}
	err = h.parkingService.DeleteBarrier(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy rào chắn để xóa"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa rào chắn", "details": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
