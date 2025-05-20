package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
)

type LPRService struct {
	rekognitionClient *rekognition.Client
}

func NewLPRService(rekClient *rekognition.Client) *LPRService {
	return &LPRService{rekognitionClient: rekClient}
}

// ProcessImageForLPR nhận ảnh dưới dạng bytes, gọi Rekognition và cố gắng trích xuất biển số
func (s *LPRService) ProcessImageForLPR(ctx context.Context, imageBytes []byte) (string, float32, error) {
	if s.rekognitionClient == nil {
		return "", 0, fmt.Errorf("Rekognition client chưa được khởi tạo")
	}

	input := &rekognition.DetectTextInput{
		Image: &types.Image{
			Bytes: imageBytes,
		},
	}

	log.Println("LPRService: Đang gọi Rekognition DetectText...")
	result, err := s.rekognitionClient.DetectText(ctx, input)
	if err != nil {
		log.Printf("LPRService: Lỗi khi gọi Rekognition DetectText: %v", err)
		return "", 0, fmt.Errorf("lỗi Rekognition: %w", err)
	}

	log.Printf("LPRService: Rekognition trả về %d khối văn bản.", len(result.TextDetections))
	var detectedTexts []string
	var highestConfidencePlate string
	var maxConfidence float32 = 0.0

	// Regex cơ bản cho biển số Việt Nam (cần cải thiện nhiều)
	// Ví dụ: 29A-123.45, 51G-12345, 80B-123.45
	// Regex này rất đơn giản và cần được làm phức tạp hơn để xử lý nhiều trường hợp
	// và giảm thiểu false positives.
	// Ví dụ: ^\d{2}[A-Z]{1,2}-\d{3,5}(\.\d{2})?$ (Rất cơ bản)
	// Hoặc một regex phức tạp hơn:
	// ^(\d{2}|80)([A-ZĐ]{1,2}|[A-ZĐ]{1}\d{1}|[A-ZĐ]{1}[A-ZĐ]{1})-(\d{3}\.\d{2}|\d{4,5})$
	// Regex này cũng chỉ là ví dụ, bạn cần tinh chỉnh kỹ lưỡng.
	plateRegex := regexp.MustCompile(`^[0-9]{2}[A-Z]{1,2}[- ]?[0-9]{3,5}(\.[0-9]{2})?$`)

	for _, textDetection := range result.TextDetections {
		if textDetection.Type == types.TextTypesLine || textDetection.Type == types.TextTypesWord {
			if textDetection.DetectedText != nil && textDetection.Confidence != nil {
				txt := strings.ToUpper(strings.ReplaceAll(*textDetection.DetectedText, " ", "")) // Loại bỏ khoảng trắng và viết hoa
				txt = strings.ReplaceAll(txt, ".", "")                                           // Loại bỏ dấu chấm để regex đơn giản hơn (tạm thời)

				log.Printf("LPRService: Text: '%s', Confidence: %.2f", txt, *textDetection.Confidence)
				detectedTexts = append(detectedTexts, fmt.Sprintf("%s (%.2f)", txt, *textDetection.Confidence))

				// Áp dụng regex để tìm biển số tiềm năng
				// và chọn cái có confidence cao nhất
				if plateRegex.MatchString(txt) {
					log.Printf("LPRService: Biển số tiềm năng khớp regex: %s", txt)
					if *textDetection.Confidence > maxConfidence {
						maxConfidence = *textDetection.Confidence
						highestConfidencePlate = strings.ReplaceAll(*textDetection.DetectedText, " ", "") // Giữ lại định dạng gốc có thể có khoảng trắng
					}
				}
			}
		}
	}

	if highestConfidencePlate != "" {
		log.Printf("LPRService: Biển số được chọn: '%s' với độ tin cậy: %.2f", highestConfidencePlate, maxConfidence)
		return highestConfidencePlate, maxConfidence, nil
	}

	log.Println("LPRService: Không tìm thấy biển số nào khớp regex từ văn bản nhận dạng.")
	log.Printf("LPRService: Tất cả văn bản nhận dạng: %s", strings.Join(detectedTexts, ", "))
	return "", 0, fmt.Errorf("không nhận dạng được biển số từ ảnh (Văn bản: %s)", strings.Join(detectedTexts, ", "))
}
