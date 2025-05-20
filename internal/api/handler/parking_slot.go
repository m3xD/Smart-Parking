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

type ParkingSlotHandler struct {
	parkingService *service.ParkingService
}

func NewParkingSlotHandler(ps *service.ParkingService) *ParkingSlotHandler {
	return &ParkingSlotHandler{parkingService: ps}
}

// POST /parking-lots/:lot_id/slots
func (h *ParkingSlotHandler) CreateParkingSlot(c *gin.Context) {
	lotIDStr := c.Param("id")
	lotID, err := strconv.Atoi(lotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lot ID không hợp lệ"})
		return
	}

	var dto domain.ParkingSlotDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dto.LotID = lotID

	slot, err := h.parkingService.CreateParkingSlot(c.Request.Context(), dto)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo chỗ đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, slot)
}

// GET /parking-lots/:lot_id/slots
func (h *ParkingSlotHandler) GetSlotsByLotID(c *gin.Context) {
	lotIDStr := c.Param("id")
	lotID, err := strconv.Atoi(lotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lot ID không hợp lệ"})
		return
	}

	slots, err := h.parkingService.GetSlotsByLotID(c.Request.Context(), lotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy danh sách chỗ đỗ xe"})
		return
	}
	c.JSON(http.StatusOK, slots)
}

// GET /parking-slots/:slot_id
func (h *ParkingSlotHandler) GetParkingSlotByID(c *gin.Context) {
	slotIDStr := c.Param("slot_id")
	slotID, err := strconv.Atoi(slotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID không hợp lệ"})
		return
	}

	slot, err := h.parkingService.GetParkingSlotByID(c.Request.Context(), slotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy chỗ đỗ xe"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi lấy thông tin chỗ đỗ xe"})
		return
	}
	c.JSON(http.StatusOK, slot)
}

// PUT /parking-slots/:slot_id
func (h *ParkingSlotHandler) UpdateParkingSlot(c *gin.Context) {
	slotIDStr := c.Param("slot_id")
	slotID, err := strconv.Atoi(slotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID không hợp lệ"})
		return
	}

	var dto domain.ParkingSlotDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedSlot, err := h.parkingService.UpdateParkingSlot(c.Request.Context(), slotID, dto)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy chỗ đỗ xe để cập nhật"})
			return
		}
		if errors.Is(err, repository.ErrDuplicateEntry) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật chỗ đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedSlot)
}

// DELETE /parking-slots/:slot_id
func (h *ParkingSlotHandler) DeleteParkingSlot(c *gin.Context) {
	slotIDStr := c.Param("slot_id")
	slotID, err := strconv.Atoi(slotIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot ID không hợp lệ"})
		return
	}

	err = h.parkingService.DeleteParkingSlot(c.Request.Context(), slotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy chỗ đỗ xe để xóa"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa chỗ đỗ xe", "details": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
