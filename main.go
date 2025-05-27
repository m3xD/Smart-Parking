package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo_config "github.com/aws/aws-sdk-go-v2/config" // Alias để tránh trùng tên
	"github.com/aws/aws-sdk-go-v2/service/iotdataplane"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"smart_parking/internal/api/handler"
	"smart_parking/internal/api/middleware"
	"smart_parking/internal/repository"

	"log"
	"net/http"
	"os"
	"os/signal"
	"smart_parking/internal/api"
	"smart_parking/internal/config"
	"smart_parking/internal/iot"
	"smart_parking/internal/repository/postgresql"
	"smart_parking/internal/service"
	"strings"
	"sync"
	"syscall"
	"time"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	// 1. Load Configuration
	cfg := config.Load()
	log.Println("Cấu hình đã được tải.")

	// 2. Setup Database Connection
	db, err := postgresql.NewDB(cfg)
	if err != nil {
		log.Fatalf("Không thể kết nối database: %v", err)
	}
	defer db.Close()
	log.Println("Đã kết nối database thành công!")

	// 3. Khởi tạo AWS SDK Config
	awsSDKCfg, err := awsgo_config.LoadDefaultConfig(context.TODO(), awsgo_config.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Fatalf("Không thể tải AWS SDK config: %v", err)
	}
	log.Println("Đã tải AWS SDK config thành công cho region:", cfg.AWSRegion)

	// 4. Khởi tạo AWS Clients
	sqsClient := sqs.NewFromConfig(awsSDKCfg)
	iotDataPlaneClient := iotdataplane.NewFromConfig(awsSDKCfg, func(o *iotdataplane.Options) {
		if cfg.IoTMQTTEndpoint != "" {
			endpointWithSchema := cfg.IoTMQTTEndpoint
			if !strings.HasPrefix(endpointWithSchema, "https://") && !strings.HasPrefix(endpointWithSchema, "http://") {
				endpointWithSchema = "https://" + endpointWithSchema
			}
			o.BaseEndpoint = aws.String(endpointWithSchema)
		}
	})
	log.Println("Đã khởi tạo SQS client và IoT Data Plane client.")

	rekognitionClient := rekognition.NewFromConfig(awsSDKCfg) // Khởi tạo Rekognition Client
	lprService := service.NewLPRService(rekognitionClient)    // Khởi tạo LPRService

	// 5. Initialize Repositories
	userRepo := postgresql.NewPgUserRepository(db) // Thêm User Repo
	parkingLotRepo := postgresql.NewPgParkingLotRepository(db)
	parkingSlotRepo := postgresql.NewPgParkingSlotRepository(db)
	barrierRepo := postgresql.NewPgBarrierRepository(db)
	deviceEventsLogRepo := postgresql.NewPgDeviceEventsLogRepository(db)
	sessionRepo := postgresql.NewPgParkingSessionRepository(db)
	deviceRepo := postgresql.NewPgDeviceRepository(db)
	gateEventRepo := postgresql.NewPgGateEventRepository(db) // Thêm GateEvent Repository

	// init websocket manager
	webSocketManager := handler.NewWebSocketManager() // Giả sử bạn có một WebSocketManager interface
	go webSocketManager.Start()
	log.Println("WebSocket Manager đã được khởi động.")

	// 6. Initialize Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpirationHours) // Thêm AuthService
	parkingService := service.NewParkingService(parkingLotRepo, parkingSlotRepo, barrierRepo,
		sessionRepo, deviceRepo, deviceEventsLogRepo)
	iotService := service.NewIoTService(parkingService, iotDataPlaneClient, cfg, deviceEventsLogRepo)
	iotServiceUpdated := service.NewIoTServiceUpdated(parkingService, iotDataPlaneClient,
		cfg, deviceEventsLogRepo, gateEventRepo, webSocketManager)

	// 7. Initialize Auth Middleware
	authMiddleware := middleware.NewAuthMiddleware(authService) // Khởi tạo Auth Middleware

	// 8. Khởi tạo và Chạy SQS Consumer
	var wg sync.WaitGroup
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())

	if cfg.SQSEventQueueURL == "" {
		log.Println("CẢNH BÁO: SQS_EVENT_QUEUE_URL chưa được cấu hình. SQS Consumer sẽ không chạy.")
	} else {
		sqsConsumer := iot.NewSQSConsumer(sqsClient, cfg, iotServiceUpdated)
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("SQS Consumer đang bắt đầu lắng nghe queue:", cfg.SQSEventQueueURL)
			sqsConsumer.Start(consumerCtx)
			log.Println("SQS Consumer đã dừng.")
		}()
	}

	// start background job để cleanup gate events
	go startGateEventCleanupJob(gateEventRepo)

	// 9. Setup HTTP Router
	router := api.SetupRouter(authService, parkingService, iotService, authMiddleware, lprService, iotServiceUpdated, webSocketManager) // Truyền authService và authMiddleware

	// 10. Start HTTP Server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Printf("Server đang chạy trên port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Lỗi ListenAndServe(): %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Đang tắt server...")

	cancelConsumer()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server buộc phải tắt: %v", err)
	}

	if cfg.SQSEventQueueURL != "" {
		log.Println("Đang chờ SQS consumer dừng (tối đa 5 giây)...")
		c := make(chan struct{})
		go func() {
			defer close(c)
			wg.Wait()
		}()
		select {
		case <-c:
			log.Println("SQS consumer đã dừng hoàn toàn.")
		case <-time.After(5 * time.Second):
			log.Println("SQS consumer không dừng trong thời gian chờ.")
		}
	}

	log.Println("Server đã tắt.")
}

func startGateEventCleanupJob(gateEventRepo repository.GateEventRepository) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		count, err := gateEventRepo.CleanupExpiredEvents(ctx)
		if err != nil {
			log.Printf("Lỗi cleanup expired gate events: %v", err)
		} else if count > 0 {
			log.Printf("Đã cleanup %d expired gate events", count)
		}
		cancel()
	}
}
