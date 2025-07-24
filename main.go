package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Estructuras para respuestas
type MultiCloudDashboard struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Services    map[string]interface{} `json:"services"`
	Metrics     DashboardMetrics       `json:"metrics"`
	Alerts      []Alert                `json:"alerts"`
	Version     string                 `json:"version"`
}

type DashboardMetrics struct {
	AWS      AWSMetrics      `json:"aws"`
	GCP      GCPMetrics      `json:"gcp"`
	Business BusinessMetrics `json:"business"`
	System   SystemMetrics   `json:"system"`
}

type AWSMetrics struct {
	Lambda   LambdaMetrics   `json:"lambda"`
	DynamoDB DynamoDBMetrics `json:"dynamodb"`
	Costs    CostMetrics     `json:"costs"`
}

type GCPMetrics struct {
	CloudFunctions CloudFunctionMetrics `json:"cloud_functions"`
	Firestore      FirestoreMetrics     `json:"firestore"`
	Costs          CostMetrics          `json:"costs"`
}

type BusinessMetrics struct {
	Products ProductMetrics `json:"products"`
	Orders   OrderMetrics   `json:"orders"`
	Revenue  RevenueMetrics `json:"revenue"`
}

type SystemMetrics struct {
	Performance PerformanceMetrics `json:"performance"`
	Health      HealthMetrics      `json:"health"`
}

type LambdaMetrics struct {
	Invocations int     `json:"invocations"`
	Duration    float64 `json:"duration"`
	Errors      int     `json:"errors"`
	SuccessRate float64 `json:"success_rate"`
}

type DynamoDBMetrics struct {
	Requests       int     `json:"requests"`
	Latency        float64 `json:"latency"`
	ConsumedReads  int     `json:"consumed_reads"`
	ConsumedWrites int     `json:"consumed_writes"`
}

type CloudFunctionMetrics struct {
	Invocations int     `json:"invocations"`
	Duration    float64 `json:"duration"`
	Errors      int     `json:"errors"`
	SuccessRate float64 `json:"success_rate"`
}

type FirestoreMetrics struct {
	Reads       int     `json:"reads"`
	Writes      int     `json:"writes"`
	Latency     float64 `json:"latency"`
	Collections int     `json:"collections"`
	Documents   int     `json:"documents"`
}

type CostMetrics struct {
	Total    float64 `json:"total_cost"`
	Today    float64 `json:"today_cost"`
	Forecast float64 `json:"forecast"`
}

type ProductMetrics struct {
	Total       int `json:"total"`
	Active      int `json:"active"`
	OutOfStock  int `json:"out_of_stock"`
	LowStock    int `json:"low_stock"`
	Categories  int `json:"categories"`
}

type OrderMetrics struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Cancelled  int `json:"cancelled"`
	Today      int `json:"today"`
}

type RevenueMetrics struct {
	Total          float64 `json:"total"`
	Today          float64 `json:"today"`
	AverageOrder   float64 `json:"average_order"`
	MonthlyTarget  float64 `json:"monthly_target"`
	Progress       float64 `json:"progress"`
}

type PerformanceMetrics struct {
	ResponseTime   int     `json:"response_time"`
	ErrorRate      float64 `json:"error_rate"`
	Uptime         float64 `json:"uptime"`
	RequestsToday  int     `json:"requests_today"`
}

type HealthMetrics struct {
	AWSLambda      string `json:"aws_lambda"`
	AWSDynamoDB    string `json:"aws_dynamodb"`
	GCPFunctions   string `json:"gcp_functions"`
	GCPFirestore   string `json:"gcp_firestore"`
	Overall        string `json:"overall"`
}

type Alert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Service     string    `json:"service"`
	Provider    string    `json:"provider"`
	Timestamp   time.Time `json:"timestamp"`
	Acknowledged bool     `json:"acknowledged"`
}

// Variables globales para simular datos en tiempo real
var (
	currentMetrics = DashboardMetrics{}
	activeAlerts   = []Alert{}
	startTime      = time.Now()
)

func main() {
	// Cargar variables de entorno
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment")
	}

	// Configurar Gin
	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware CORS
	router.Use(corsMiddleware())

	// Middleware para servir archivos estáticos
	router.Static("/static", "./monitoring-dashboard/web-dashboard")

	// Inicializar datos de ejemplo
	initializeMockData()

	// Rutas principales
	setupRoutes(router)

	// Banner de inicio
	printEnhancedBanner()

	// Iniciar simulación de datos en tiempo real
	go startDataSimulation()

	// Obtener puerto
	port := getPort()

	log.Printf("🚀 Enhanced Multi-Cloud Server starting on port %s", port)
	log.Fatal(router.Run(":" + port))
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func setupRoutes(router *gin.Engine) {
	// Ruta principal - Dashboard
	router.GET("/", serveDashboard)
	
	// API v1
	v1 := router.Group("/api/v1")
	{
		v1.GET("/dashboard", getDashboardData)
		v1.GET("/health", getHealthStatus)
		v1.GET("/metrics/snapshot", getMetricsSnapshot)
		
		// Métricas específicas
		v1.GET("/metrics/aws", getAWSMetrics)
		v1.GET("/metrics/gcp", getGCPMetrics)
		v1.GET("/metrics/business", getBusinessMetrics)
		v1.GET("/metrics/system", getSystemMetrics)
		
		// Alertas
		v1.GET("/alerts", getAlerts)
		v1.POST("/alerts/:id/acknowledge", acknowledgeAlert)
		
		// Simulación y testing
		v1.POST("/simulate/load", simulateLoad)
		v1.POST("/simulate/error", simulateError)
		v1.POST("/simulate/alert", simulateAlert)
	}

	// API para el collector (puerto 8081)
	api := router.Group("/api")
	{
		api.GET("/metrics/snapshot", getMetricsSnapshot)
		api.GET("/health", getHealthStatus)
		api.GET("/metrics/aws", getAWSMetrics)
		api.GET("/metrics/gcp", getGCPMetrics)
		api.GET("/metrics/business", getBusinessMetrics)
	}
}

func serveDashboard(c *gin.Context) {
	// Servir el dashboard HTML actualizado
	dashboardPath := filepath.Join("monitoring-dashboard", "web-dashboard", "dashboard.html")
	
	if _, err := os.Stat(dashboardPath); os.IsNotExist(err) {
		// Si no existe el archivo, servir dashboard básico
		c.HTML(200, "dashboard.html", gin.H{
			"title": "Multi-Cloud Dashboard",
		})
		return
	}
	
	c.File(dashboardPath)
}

func getDashboardData(c *gin.Context) {
	dashboard := MultiCloudDashboard{
		Status:    "operational",
		Timestamp: time.Now(),
		Services: map[string]interface{}{
			"aws_lambda":      map[string]string{"status": "healthy", "region": "us-east-1"},
			"aws_dynamodb":    map[string]string{"status": "healthy", "region": "us-east-1"},
			"gcp_functions":   map[string]string{"status": "healthy", "region": "us-central1"},
			"gcp_firestore":   map[string]string{"status": "healthy", "region": "us-central1"},
			"monitoring":      map[string]string{"status": "active", "collector": "running"},
		},
		Metrics: currentMetrics,
		Alerts:  activeAlerts,
		Version: "2.0.0",
	}

	c.JSON(200, dashboard)
}

func getHealthStatus(c *gin.Context) {
	uptime := time.Since(startTime)
	
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    uptime.String(),
		"services": map[string]string{
			"aws_lambda":      "connected",
			"aws_dynamodb":    "connected",
			"gcp_functions":   "connected",
			"gcp_firestore":   "connected",
			"prometheus":      "active",
			"dashboard":       "running",
		},
		"version": "2.0.0",
		"environment": os.Getenv("APP_ENV"),
	}

	c.JSON(200, health)
}

func getMetricsSnapshot(c *gin.Context) {
	snapshot := map[string]interface{}{
		"timestamp": time.Now(),
		"aws_metrics": map[string]interface{}{
			"lambda_invocations": currentMetrics.AWS.Lambda.Invocations,
			"lambda_duration":    currentMetrics.AWS.Lambda.Duration,
			"lambda_errors":      currentMetrics.AWS.Lambda.Errors,
			"dynamodb_requests":  currentMetrics.AWS.DynamoDB.Requests,
			"dynamodb_latency":   currentMetrics.AWS.DynamoDB.Latency,
		},
		"gcp_metrics": map[string]interface{}{
			"function_invocations": currentMetrics.GCP.CloudFunctions.Invocations,
			"function_duration":    currentMetrics.GCP.CloudFunctions.Duration,
			"function_errors":      currentMetrics.GCP.CloudFunctions.Errors,
			"firestore_reads":      currentMetrics.GCP.Firestore.Reads,
			"firestore_writes":     currentMetrics.GCP.Firestore.Writes,
		},
		"business_metrics": map[string]interface{}{
			"total_products":       currentMetrics.Business.Products.Total,
			"total_orders":         currentMetrics.Business.Orders.Total,
			"total_revenue":        currentMetrics.Business.Revenue.Total,
			"average_order_value":  currentMetrics.Business.Revenue.AverageOrder,
			"error_rate":           currentMetrics.System.Performance.ErrorRate,
		},
		"health_status": map[string]string{
			"aws_lambda":      currentMetrics.System.Health.AWSLambda,
			"aws_dynamodb":    currentMetrics.System.Health.AWSDynamoDB,
			"gcp_functions":   currentMetrics.System.Health.GCPFunctions,
			"gcp_firestore":   currentMetrics.System.Health.GCPFirestore,
			"overall":         currentMetrics.System.Health.Overall,
		},
	}

	c.JSON(200, snapshot)
}

func getAWSMetrics(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"lambda": currentMetrics.AWS.Lambda,
		"dynamodb": currentMetrics.AWS.DynamoDB,
		"costs": currentMetrics.AWS.Costs,
	})
}

func getGCPMetrics(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"cloud_functions": currentMetrics.GCP.CloudFunctions,
		"firestore": currentMetrics.GCP.Firestore,
		"costs": currentMetrics.GCP.Costs,
	})
}

func getBusinessMetrics(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"products": currentMetrics.Business.Products,
		"orders": currentMetrics.Business.Orders,
		"revenue": currentMetrics.Business.Revenue,
		"performance": currentMetrics.System.Performance,
	})
}

func getSystemMetrics(c *gin.Context) {
	c.JSON(200, currentMetrics.System)
}

func getAlerts(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"alerts": activeAlerts,
		"total": len(activeAlerts),
		"unacknowledged": countUnacknowledgedAlerts(),
	})
}

func acknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	
	for i, alert := range activeAlerts {
		if alert.ID == alertID {
			activeAlerts[i].Acknowledged = true
			c.JSON(200, map[string]interface{}{
				"success": true,
				"message": "Alert acknowledged",
				"alert_id": alertID,
			})
			return
		}
	}
	
	c.JSON(404, map[string]interface{}{
		"success": false,
		"message": "Alert not found",
	})
}

// Simulación de carga y errores
func simulateLoad(c *gin.Context) {
	// Simular aumento de tráfico
	currentMetrics.AWS.Lambda.Invocations += 50
	currentMetrics.GCP.CloudFunctions.Invocations += 30
	currentMetrics.System.Performance.RequestsToday += 80
	
	c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Load simulation started",
		"duration": "30 seconds",
	})
}

func simulateError(c *gin.Context) {
	// Simular errores
	currentMetrics.AWS.Lambda.Errors += 5
	currentMetrics.GCP.CloudFunctions.Errors += 3
	currentMetrics.System.Performance.ErrorRate += 1.0
	
	// Crear alerta
	alert := Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().Unix()),
		Type:      "error_spike",
		Severity:  "warning",
		Message:   "Increased error rate detected across services",
		Service:   "multi-cloud",
		Provider:  "system",
		Timestamp: time.Now(),
		Acknowledged: false,
	}
	activeAlerts = append(activeAlerts, alert)
	
	c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Error simulation started",
		"alert_created": alert.ID,
	})
}

func simulateAlert(c *gin.Context) {
	var request struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Service  string `json:"service"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, map[string]interface{}{
			"success": false,
			"error": "Invalid request body",
		})
		return
	}
	
	alert := Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().Unix()),
		Type:      "manual",
		Severity:  request.Severity,
		Message:   request.Message,
		Service:   request.Service,
		Provider:  "manual",
		Timestamp: time.Now(),
		Acknowledged: false,
	}
	
	activeAlerts = append(activeAlerts, alert)
	
	c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Alert created successfully",
		"alert": alert,
	})
}

// Funciones de utilidad
func initializeMockData() {
	currentMetrics = DashboardMetrics{
		AWS: AWSMetrics{
			Lambda: LambdaMetrics{
				Invocations: 150,
				Duration:    0.25,
				Errors:      2,
				SuccessRate: 98.7,
			},
			DynamoDB: DynamoDBMetrics{
				Requests:       145,
				Latency:        0.1,
				ConsumedReads:  75,
				ConsumedWrites: 23,
			},
			Costs: CostMetrics{
				Total:    21.35,
				Today:    2.45,
				Forecast: 650.00,
			},
		},
		GCP: GCPMetrics{
			CloudFunctions: CloudFunctionMetrics{
				Invocations: 98,
				Duration:    0.2,
				Errors:      1,
				SuccessRate: 99.0,
			},
			Firestore: FirestoreMetrics{
				Reads:       87,
				Writes:      23,
				Latency:     0.15,
				Collections: 2,
				Documents:   156,
			},
			Costs: CostMetrics{
				Total:    16.25,
				Today:    1.80,
				Forecast: 490.00,
			},
		},
		Business: BusinessMetrics{
			Products: ProductMetrics{
				Total:       25,
				Active:      23,
				OutOfStock:  2,
				LowStock:    5,
				Categories:  4,
			},
			Orders: OrderMetrics{
				Total:      156,
				Pending:    12,
				Processing: 8,
				Completed:  130,
				Cancelled:  6,
				Today:      15,
			},
			Revenue: RevenueMetrics{
				Total:         15678.50,
				Today:         1234.60,
				AverageOrder:  100.50,
				MonthlyTarget: 20000.00,
				Progress:      78.4,
			},
		},
		System: SystemMetrics{
			Performance: PerformanceMetrics{
				ResponseTime:  145,
				ErrorRate:     2.5,
				Uptime:        99.9,
				RequestsToday: 543,
			},
			Health: HealthMetrics{
				AWSLambda:    "healthy",
				AWSDynamoDB:  "healthy",
				GCPFunctions: "healthy",
				GCPFirestore: "healthy",
				Overall:      "healthy",
			},
		},
	}
}

func startDataSimulation() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Simular variaciones en los datos
		currentMetrics.AWS.Lambda.Invocations += randomInt(-5, 15)
		currentMetrics.GCP.CloudFunctions.Invocations += randomInt(-3, 10)
		currentMetrics.Business.Orders.Today += randomInt(0, 3)
		currentMetrics.System.Performance.RequestsToday += randomInt(1, 20)
		
		// Limpiar alertas muy antiguas
		cleanOldAlerts()
	}
}

func randomInt(min, max int) int {
	return min + int(time.Now().UnixNano()%int64(max-min+1))
}

func countUnacknowledgedAlerts() int {
	count := 0
	for _, alert := range activeAlerts {
		if !alert.Acknowledged {
			count++
		}
	}
	return count
}

func cleanOldAlerts() {
	// Mantener solo alertas de las últimas 24 horas
	cutoff := time.Now().Add(-24 * time.Hour)
	var filteredAlerts []Alert
	
	for _, alert := range activeAlerts {
		if alert.Timestamp.After(cutoff) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}
	
	activeAlerts = filteredAlerts
}

func getPort() string {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func printEnhancedBanner() {
	banner := `
╔══════════════════════════════════════════════════════════════════════════╗
║                                                                          ║
║     🚀 ENHANCED MULTI-CLOUD E-COMMERCE MONITOR v2.0.0                   ║
║                                                                          ║
║     ☁️  AWS Lambda + DynamoDB                                           ║
║     🌐 Google Cloud Functions + Firestore                              ║
║     📊 Real-time Prometheus Metrics                                     ║
║     🎛️  Interactive Dashboard                                           ║
║     🔄 Live Data Simulation                                             ║
║                                                                          ║
╠══════════════════════════════════════════════════════════════════════════╣
║                                                                          ║
║      Enhanced Dashboard: http://localhost:8080                           ║
║      Health Check:       http://localhost:8080/api/v1/health             ║
║      API Endpoints:      http://localhost:8080/api/v1/                   ║
║      Metrics Collector:  http://localhost:8081/metrics                   ║
║                                                                          ║
║  🎯 New Features:                                                        ║
║     • Real-time data simulation                                         ║
║     • Interactive charts and graphs                                     ║
║     • Alert management system                                           ║
║     • Load and error simulation                                         ║
║     • Enhanced business metrics                                         ║
║     • Multi-provider cost tracking                                      ║
║                                                                          ║
╚══════════════════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}