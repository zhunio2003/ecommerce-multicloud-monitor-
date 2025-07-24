package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Structs para las respuestas
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
}

type CloudStatus struct {
	Provider string `json:"provider"`
	Region   string `json:"region"`
	Status   string `json:"status"`
	Services int    `json:"services"`
}

type MultiCloudResponse struct {
	Message string        `json:"message"`
	Clouds  []CloudStatus `json:"clouds"`
	Total   int           `json:"total_services"`
}

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

	// Middleware para CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 🏠 Ruta principal
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "🚀 Multi-Cloud E-commerce Monitor",
			"version": "1.0.0",
			"status":  "running",
			"time":    time.Now(),
			"endpoints": map[string]string{
				"health":     "/health",
				"multicloud": "/api/multicloud",
				"aws":        "/api/aws/status",
				"gcp":        "/api/gcp/status",
				"dashboard":  "/dashboard",
			},
		})
	})

	// ❤️ Health Check
	router.GET("/health", healthCheck)

	// 🌍 Multi-Cloud Status
	router.GET("/api/multicloud", getMultiCloudStatus)

	// ☁️ AWS Status
	router.GET("/api/aws/status", getAWSStatus)

	// 🌐 GCP Status  
	router.GET("/api/gcp/status", getGCPStatus)

	// 🎛️ Dashboard (HTML simple por ahora)
	router.GET("/dashboard", serveDashboard)

	// 📊 API para métricas (mock por ahora)
	router.GET("/api/metrics", getMetrics)

	// Obtener puerto
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Banner de inicio
	printStartupBanner(port)

	// Iniciar servidor
	log.Printf("🚀 Server starting on port %s", port)
	log.Fatal(router.Run(":" + port))
}

func healthCheck(c *gin.Context) {
	services := map[string]string{
		"database":   "healthy",
		"aws":        "connected",
		"gcp":        "connected",
		"monitoring": "active",
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  services,
		Version:   "1.0.0",
	}

	c.JSON(http.StatusOK, response)
}

func getMultiCloudStatus(c *gin.Context) {
	clouds := []CloudStatus{
		{
			Provider: "AWS",
			Region:   os.Getenv("AWS_REGION"),
			Status:   "active",
			Services: 4, // Lambda, DynamoDB, S3, CloudWatch
		},
		{
			Provider: "Google Cloud",
			Region:   os.Getenv("GCP_REGION"),
			Status:   "active",
			Services: 4, // Cloud Functions, Firestore, Storage, Monitoring
		},
	}

	response := MultiCloudResponse{
		Message: "Multi-cloud infrastructure is operational",
		Clouds:  clouds,
		Total:   8,
	}

	c.JSON(http.StatusOK, response)
}

func getAWSStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"provider": "AWS",
		"region":   os.Getenv("AWS_REGION"),
		"services": map[string]interface{}{
			"lambda": map[string]string{
				"status":   "active",
				"functions": "3",
			},
			"dynamodb": map[string]string{
				"status": "active",
				"tables": "2",
			},
			"s3": map[string]string{
				"status":  "active",
				"buckets": "1",
			},
			"cloudwatch": map[string]string{
				"status": "monitoring",
				"alarms": "5",
			},
		},
		"costs": map[string]string{
			"current_month": "$23.45",
			"last_month":    "$18.32",
		},
	})
}

func getGCPStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"provider": "Google Cloud Platform",
		"region":   os.Getenv("GCP_REGION"),
		"services": map[string]interface{}{
			"cloud_functions": map[string]string{
				"status":   "active",
				"functions": "3",
			},
			"firestore": map[string]string{
				"status":      "active",
				"collections": "2",
			},
			"cloud_storage": map[string]string{
				"status":  "active",
				"buckets": "1",
			},
			"monitoring": map[string]string{
				"status": "active",
				"alerts": "4",
			},
		},
		"costs": map[string]string{
			"current_month": "$19.78",
			"last_month":    "$15.44",
		},
	})
}

func getMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now(),
		"aws": map[string]interface{}{
			"cpu_usage":    75.5,
			"memory_usage": 68.2,
			"requests_per_second": 145,
			"error_rate": 0.02,
		},
		"gcp": map[string]interface{}{
			"cpu_usage":    82.1,
			"memory_usage": 71.8,
			"requests_per_second": 98,
			"error_rate": 0.01,
		},
		"overall": map[string]interface{}{
			"total_requests": 243,
			"avg_response_time": 150,
			"uptime": "99.98%",
		},
	})
}

func serveDashboard(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🚀 Multi-Cloud Dashboard</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            margin-bottom: 40px;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        .card {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 25px;
            border: 1px solid rgba(255, 255, 255, 0.2);
            transition: transform 0.3s ease;
        }
        .card:hover {
            transform: translateY(-5px);
        }
        .status {
            display: inline-block;
            padding: 5px 15px;
            border-radius: 20px;
            font-size: 0.8em;
            font-weight: bold;
        }
        .status.active {
            background: #4CAF50;
        }
        .metric {
            font-size: 2em;
            font-weight: bold;
            color: #FFD700;
        }
        .refresh-btn {
            background: #4CAF50;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 25px;
            cursor: pointer;
            font-size: 1em;
            margin: 10px;
        }
        .refresh-btn:hover {
            background: #45a049;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Multi-Cloud E-commerce Monitor</h1>
            <p>Monitoring AWS & Google Cloud Platform in Real-Time</p>
            <button onclick="refreshData()" class="refresh-btn">🔄 Refresh Data</button>
        </div>
        
        <div class="grid">
            <div class="card">
                <h3>☁️ AWS Status</h3>
                <div class="status active">ACTIVE</div>
                <p>Region: <strong id="aws-region">Loading...</strong></p>
                <p>Services: <span class="metric" id="aws-services">-</span></p>
                <p>Cost: <strong id="aws-cost">Loading...</strong></p>
            </div>
            
            <div class="card">
                <h3>🌐 Google Cloud Status</h3>
                <div class="status active">ACTIVE</div>
                <p>Region: <strong id="gcp-region">Loading...</strong></p>
                <p>Services: <span class="metric" id="gcp-services">-</span></p>
                <p>Cost: <strong id="gcp-cost">Loading...</strong></p>
            </div>
            
            <div class="card">
                <h3>📊 Performance Metrics</h3>
                <p>Total Requests: <span class="metric" id="total-requests">-</span></p>
                <p>Avg Response: <strong id="avg-response">-</strong></p>
                <p>Uptime: <strong id="uptime">-</strong></p>
            </div>
            
            <div class="card">
                <h3>🚨 System Health</h3>
                <p>Overall Status: <span class="status active">HEALTHY</span></p>
                <p>Active Alerts: <span class="metric">0</span></p>
                <p>Last Check: <strong id="last-check">-</strong></p>
            </div>
        </div>
    </div>

    <script>
        async function loadData() {
            try {
                // Cargar datos de AWS
                const awsResponse = await fetch('/api/aws/status');
                const awsData = await awsResponse.json();
                document.getElementById('aws-region').textContent = awsData.region || 'us-east-1';
                document.getElementById('aws-services').textContent = '4';
                document.getElementById('aws-cost').textContent = awsData.costs.current_month;

                // Cargar datos de GCP
                const gcpResponse = await fetch('/api/gcp/status');
                const gcpData = await gcpResponse.json();
                document.getElementById('gcp-region').textContent = gcpData.region || 'us-central1';
                document.getElementById('gcp-services').textContent = '4';
                document.getElementById('gcp-cost').textContent = gcpData.costs.current_month;

                // Cargar métricas
                const metricsResponse = await fetch('/api/metrics');
                const metricsData = await metricsResponse.json();
                document.getElementById('total-requests').textContent = metricsData.overall.total_requests;
                document.getElementById('avg-response').textContent = metricsData.overall.avg_response_time + 'ms';
                document.getElementById('uptime').textContent = metricsData.overall.uptime;

                // Actualizar timestamp
                document.getElementById('last-check').textContent = new Date().toLocaleTimeString();
                
            } catch (error) {
                console.error('Error loading data:', error);
            }
        }

        function refreshData() {
            loadData();
        }

        // Cargar datos al inicio
        loadData();

        // Auto-refresh cada 30 segundos
        setInterval(loadData, 30000);
    </script>
</body>
</html>
    `
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

func printStartupBanner(port string) {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║     🚀 MULTI-CLOUD E-COMMERCE MONITOR v1.0.0                ║
║                                                              ║
║     ☁️  AWS + Google Cloud Platform Integration             ║
║     📊 Real-time Monitoring & Analytics                     ║
║     🔄 Automated Resource Management                        ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║  🌐 Dashboard: http://localhost:` + port + `                        ║
║  🩺 Health:    http://localhost:` + port + `/health                 ║
║  📡 API:       http://localhost:` + port + `/api                    ║
║                                                              ║
║  📚 Available Endpoints:                                     ║
║     GET /                    - API Information              ║
║     GET /health              - Health Check                 ║
║     GET /dashboard           - Web Dashboard                ║
║     GET /api/multicloud      - Multi-Cloud Status          ║
║     GET /api/aws/status      - AWS Services Status         ║
║     GET /api/gcp/status      - GCP Services Status         ║
║     GET /api/metrics         - Performance Metrics         ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}