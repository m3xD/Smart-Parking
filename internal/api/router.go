package api

import (
	"smart_parking/internal/api/handler"
	"smart_parking/internal/api/middleware"
	"smart_parking/internal/service"
	// "parking_system_go/internal/domain" // Không cần trực tiếp ở đây nữa

	"github.com/gin-gonic/gin"
	// "net/http"
	// "strconv"
	// "errors"
	// "parking_system_go/internal/repository"
)

func SetupRouter(as *service.AuthService, ps *service.ParkingService, is *service.IoTService,
	authMw *middleware.AuthMiddleware, lprService *service.LPRService, iotServiceUpdated *service.IoTService, wsManager *handler.WebSocketManager) *gin.Engine {
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// WebSocket endpoint (không cần auth cho real-time connection)
	wsHandler := handler.NewWebSocketHandler(wsManager)
	r.GET("/ws", wsHandler.HandleWebSocket)

	authHandler := handler.NewAuthHandler(as)
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
	}

	v1 := r.Group("/api/v1")
	v1.Use(authMw.Authenticate())
	{
		lotH := handler.NewParkingLotHandler(ps)
		lotRoutes := v1.Group("/parking-lots")
		{
			lotRoutes.POST("", authMw.AuthorizeRole("admin"), lotH.CreateParkingLot)
			lotRoutes.GET("", lotH.GetAllParkingLots)
			lotRoutes.GET("/:id", lotH.GetParkingLotByID)
			lotRoutes.PUT("/:id", authMw.AuthorizeRole("admin"), lotH.UpdateParkingLot)
			lotRoutes.DELETE("/:id", authMw.AuthorizeRole("admin"), lotH.DeleteParkingLot)

			slotH_nested := handler.NewParkingSlotHandler(ps)
			slotRoutesInLot := lotRoutes.Group("/:id/slots")
			{
				slotRoutesInLot.POST("", authMw.AuthorizeRole("admin"), slotH_nested.CreateParkingSlot)
				slotRoutesInLot.GET("", slotH_nested.GetSlotsByLotID)
			}

			barrierH_nested := handler.NewBarrierHandler(ps)
			lotRoutes.GET("/:id/barriers", barrierH_nested.GetBarriersByLotID)

			sessionH_nested := handler.NewParkingSessionHandler(ps)
			lotRoutes.GET("/:id/active-sessions", sessionH_nested.GetActiveSessionsByLotID)
		}

		slotH := handler.NewParkingSlotHandler(ps)
		slotRoutes := v1.Group("/parking-slots")
		{
			slotRoutes.GET("/:slot_id", slotH.GetParkingSlotByID)
			slotRoutes.PUT("/:slot_id", authMw.AuthorizeRole("admin"), slotH.UpdateParkingSlot)
			slotRoutes.DELETE("/:slot_id", authMw.AuthorizeRole("admin"), slotH.DeleteParkingSlot)
		}

		barrierH := handler.NewBarrierHandler(ps)
		barrierRoutes := v1.Group("/barriers")
		{
			barrierRoutes.POST("", authMw.AuthorizeRole("admin"), barrierH.CreateBarrier)
			barrierRoutes.GET("/:id", barrierH.GetBarrierByID)
			barrierRoutes.PUT("/:id", authMw.AuthorizeRole("admin"), barrierH.UpdateBarrier)
			barrierRoutes.DELETE("/:id", authMw.AuthorizeRole("admin"), barrierH.DeleteBarrier)
		}

		sessionH := handler.NewParkingSessionHandler(ps) // Sử dụng handler đã tạo
		sessionRoutes := v1.Group("/parking-sessions")
		{
			sessionRoutes.POST("/check-in", sessionH.VehicleCheckIn)   // API check-in
			sessionRoutes.POST("/check-out", sessionH.VehicleCheckOut) // API check-out
			sessionRoutes.GET("", sessionH.FindParkingSessions)        // API filter sessions
			sessionRoutes.GET("/:id", sessionH.GetParkingSessionByID)
		}

		// Device Monitoring Routes
		deviceH := handler.NewDeviceHandler(ps)
		deviceRoutes := v1.Group("/devices")
		deviceRoutes.Use(authMw.AuthorizeRole("admin")) // Chỉ admin được xem thông tin thiết bị
		{
			deviceRoutes.GET("", deviceH.GetAllDevices)
			deviceRoutes.GET("/:thing_name", deviceH.GetDeviceByThingName)
		}

		if is != nil {
			iotCmdH := handler.NewIoTCommandHandler(is)
			iotRoutes := v1.Group("/iot/commands")
			iotRoutes.Use(authMw.AuthorizeRole("admin", "operator"))
			{
				iotRoutes.POST("/barrier", iotCmdH.ControlBarrier)
			}
		}

		if lprService != nil { // Kiểm tra nếu lprService được truyền vào
			lprH := handler.NewLPRHandler(lprService, ps) // Truyền cả parkingService
			lprRoutes := v1.Group("/lpr")
			// Có thể cần quyền admin hoặc operator cho API này
			lprRoutes.Use(authMw.AuthorizeRole("admin", "operator"))
			{
				lprRoutes.POST("/process-image", lprH.ProcessImage)
			}
		}

		gateEventHandler := handler.NewGateEventHandler(iotServiceUpdated, lprService, ps)
		gateRoutes := v1.Group("/gate-events")
		gateRoutes.Use(authMw.AuthorizeRole("admin", "operator"))
		{
			gateRoutes.POST("/lpr-trigger", gateEventHandler.TriggerLPR)
			gateRoutes.POST("/create-session", gateEventHandler.CreateSessionFromEvent)
			gateRoutes.GET("/pending", gateEventHandler.GetPendingGateEvents)
		}
	}
	return r
}
