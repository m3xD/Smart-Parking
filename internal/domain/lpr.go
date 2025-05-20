package domain

// LPRRequestDTO dùng khi frontend gửi ảnh lên
type LPRRequestDTO struct {
	// Frontend có thể gửi ảnh dưới dạng base64 encoded string
	ImageBase64 string `json:"image_base64" binding:"required"`
	// Hoặc backend xử lý multipart/form-data nếu frontend upload file trực tiếp
}

// LPRResponseDTO trả về biển số đã nhận dạng
type LPRResponseDTO struct {
	DetectedPlate string  `json:"detected_plate"`
	Confidence    float32 `json:"confidence,omitempty"` // Độ tin cậy (nếu có)
	ErrorMessage  string  `json:"error_message,omitempty"`
}
