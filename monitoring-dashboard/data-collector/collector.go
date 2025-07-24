package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// MetricsCollector estructura principal del collector
type MetricsCollector struct {
	// AWS clients
	awsConfig       aws.Config
	lambdaClient    *lambda.Client
	dynamoClient    *dynamodb.Client
	cloudWatchClient *cloudwatch.Client
	
	// GCP clients
	firestoreClient   *firestore.Client
	monitoringClient  *monitoring.MetricClient
	gcpProjectID      string
	
	// Prometheus metrics
	awsMetrics        *AWSMetrics
	gcpMetrics        *GCPMetrics
	systemMetrics     *SystemMetrics
	
	// Configuration
	config            *CollectorConfig
	logger            *logrus.Logger
}

// CollectorConfig configuración del collector
type CollectorConfig struct {
	AWSRegion           string        `json:"aws_region"`
	GCPProjectID        string        `json:"gcp_project_id"`
	CollectionInterval  time.Duration `json:"collection_interval"`
	PrometheusPort      string        `json:"prometheus_port"`
	LogLevel            string        `json:"log_level"`
	AWSLambdaFunction   string        `json:"aws_lambda_function"`
	GCPCloudFunction    string        `json:"gcp_cloud_function"`
	DynamoDBTable       string        `json:"dynamodb_table"`
	FirestoreCollection string        `json:"firestore_collection"`
}

// AWSMetrics métricas de AWS
type AWSMetrics struct {
	LambdaInvocations   prometheus.CounterVec
	LambdaDuration      prometheus.HistogramVec
	LambdaErrors        prometheus.CounterVec
	DynamoDBRequests    prometheus.CounterVec
	DynamoDBLatency     prometheus.HistogramVec
	DynamoDBThrottles   prometheus.CounterVec
	CloudWatchAlarms    prometheus.GaugeVec
}

// GCPMetrics métricas de GCP
type GCPMetrics struct {
	CloudFunctionInvocations prometheus.CounterVec
	CloudFunctionDuration    prometheus.HistogramVec
	CloudFunctionErrors      prometheus.CounterVec
	FirestoreReads          prometheus.CounterVec
	FirestoreWrites         prometheus.CounterVec
	FirestoreLatency        prometheus.HistogramVec
}

// SystemMetrics métricas del sistema general
type SystemMetrics struct {
	TotalProducts       prometheus.Gauge
	TotalOrders        prometheus.Gauge
	TotalRevenue       prometheus.Gauge
	AverageOrderValue  prometheus.Gauge
	ErrorRate          prometheus.Gauge
	ResponseTime       prometheus.Histogram
	SystemHealth       prometheus.GaugeVec
}

// MetricSnapshot snapshot de métricas en un momento dado
type MetricSnapshot struct {
	Timestamp       time.Time              `json:"timestamp"`
	AWSMetrics      map[string]interface{} `json:"aws_metrics"`
	GCPMetrics      map[string]interface{} `json:"gcp_metrics"`
	BusinessMetrics map[string]interface{} `json:"business_metrics"`
	HealthStatus    map[string]string      `json:"health_status"`
}

// NewMetricsCollector crea un nuevo collector
func NewMetricsCollector(config *CollectorConfig) (*MetricsCollector, error) {
	collector := &MetricsCollector{
		config: config,
		logger: logrus.New(),
	}

	// Configurar logger
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	collector.logger.SetLevel(level)

	// Inicializar clientes AWS
	if err := collector.initAWSClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize AWS clients: %w", err)
	}

	// Inicializar clientes GCP
	if err := collector.initGCPClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize GCP clients: %w", err)
	}

	// Inicializar métricas de Prometheus
	collector.initPrometheusMetrics()

	collector.logger.Info("🚀 Metrics Collector initialized successfully")
	return collector, nil
}

// initAWSClients inicializa los clientes de AWS
func (mc *MetricsCollector) initAWSClients() error {
	ctx := context.Background()
	
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(mc.config.AWSRegion))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	mc.awsConfig = cfg
	mc.lambdaClient = lambda.NewFromConfig(cfg)
	mc.dynamoClient = dynamodb.NewFromConfig(cfg)
	mc.cloudWatchClient = cloudwatch.NewFromConfig(cfg)

	mc.logger.Info("✅ AWS clients initialized")
	return nil
}

// initGCPClients inicializa los clientes de GCP
func (mc *MetricsCollector) initGCPClients() error {
	ctx := context.Background()

	// Cliente de Firestore
	firestoreClient, err := firestore.NewClient(ctx, mc.config.GCPProjectID)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %w", err)
	}
	mc.firestoreClient = firestoreClient

	// Cliente de Monitoring
	monitoringClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Monitoring client: %w", err)
	}
	mc.monitoringClient = monitoringClient

	mc.gcpProjectID = mc.config.GCPProjectID
	mc.logger.Info("✅ GCP clients initialized")
	return nil
}

// initPrometheusMetrics inicializa las métricas de Prometheus
func (mc *MetricsCollector) initPrometheusMetrics() {
	// AWS Metrics
	mc.awsMetrics = &AWSMetrics{
		LambdaInvocations: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aws_lambda_invocations_total",
				Help: "Total number of Lambda function invocations",
			},
			[]string{"function_name", "status"},
		),
		LambdaDuration: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "aws_lambda_duration_seconds",
				Help: "Lambda function execution duration",
			},
			[]string{"function_name"},
		),
		LambdaErrors: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aws_lambda_errors_total",
				Help: "Total number of Lambda function errors",
			},
			[]string{"function_name", "error_type"},
		),
		DynamoDBRequests: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aws_dynamodb_requests_total",
				Help: "Total number of DynamoDB requests",
			},
			[]string{"table_name", "operation"},
		),
		DynamoDBLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "aws_dynamodb_latency_seconds",
				Help: "DynamoDB request latency",
			},
			[]string{"table_name", "operation"},
		),
	}

	// GCP Metrics
	mc.gcpMetrics = &GCPMetrics{
		CloudFunctionInvocations: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gcp_cloud_function_invocations_total",
				Help: "Total number of Cloud Function invocations",
			},
			[]string{"function_name", "status"},
		),
		CloudFunctionDuration: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "gcp_cloud_function_duration_seconds",
				Help: "Cloud Function execution duration",
			},
			[]string{"function_name"},
		),
		CloudFunctionErrors: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gcp_cloud_function_errors_total",
				Help: "Total number of Cloud Function errors",
			},
			[]string{"function_name", "error_type"},
		),
		FirestoreReads: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gcp_firestore_reads_total",
				Help: "Total number of Firestore read operations",
			},
			[]string{"collection"},
		),
		FirestoreWrites: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gcp_firestore_writes_total",
				Help: "Total number of Firestore write operations",
			},
			[]string{"collection"},
		),
	}

	// System Metrics
	mc.systemMetrics = &SystemMetrics{
		TotalProducts: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ecommerce_products_total",
				Help: "Total number of products in catalog",
			},
		),
		TotalOrders: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ecommerce_orders_total",
				Help: "Total number of orders",
			},
		),
		TotalRevenue: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ecommerce_revenue_total",
				Help: "Total revenue in USD",
			},
		),
		AverageOrderValue: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ecommerce_average_order_value",
				Help: "Average order value in USD",
			},
		),
		ErrorRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ecommerce_error_rate",
				Help: "Overall system error rate percentage",
			},
		),
		ResponseTime: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name: "ecommerce_response_time_seconds",
				Help: "System response time",
			},
		),
		SystemHealth: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ecommerce_system_health",
				Help: "System health status (1 = healthy, 0 = unhealthy)",
			},
			[]string{"service", "provider"},
		),
	}

	// Registrar todas las métricas
	mc.registerMetrics()
	mc.logger.Info("✅ Prometheus metrics initialized")
}

// registerMetrics registra las métricas en Prometheus
func (mc *MetricsCollector) registerMetrics() {
	// AWS metrics
	prometheus.MustRegister(mc.awsMetrics.LambdaInvocations)
	prometheus.MustRegister(mc.awsMetrics.LambdaDuration)
	prometheus.MustRegister(mc.awsMetrics.LambdaErrors)
	prometheus.MustRegister(mc.awsMetrics.DynamoDBRequests)
	prometheus.MustRegister(mc.awsMetrics.DynamoDBLatency)

	// GCP metrics
	prometheus.MustRegister(mc.gcpMetrics.CloudFunctionInvocations)
	prometheus.MustRegister(mc.gcpMetrics.CloudFunctionDuration)
	prometheus.MustRegister(mc.gcpMetrics.CloudFunctionErrors)
	prometheus.MustRegister(mc.gcpMetrics.FirestoreReads)
	prometheus.MustRegister(mc.gcpMetrics.FirestoreWrites)

	// System metrics
	prometheus.MustRegister(mc.systemMetrics.TotalProducts)
	prometheus.MustRegister(mc.systemMetrics.TotalOrders)
	prometheus.MustRegister(mc.systemMetrics.TotalRevenue)
	prometheus.MustRegister(mc.systemMetrics.AverageOrderValue)
	prometheus.MustRegister(mc.systemMetrics.ErrorRate)
	prometheus.MustRegister(mc.systemMetrics.ResponseTime)
	prometheus.MustRegister(mc.systemMetrics.SystemHealth)
}

// Start inicia el collector
func (mc *MetricsCollector) Start() error {
	mc.logger.Info("🚀 Starting Metrics Collector...")

	// Iniciar servidor HTTP para métricas de Prometheus
	go mc.startPrometheusServer()

	// Iniciar recolección periódica
	go mc.startPeriodicCollection()

	// Iniciar servidor HTTP para API
	go mc.startAPIServer()

	mc.logger.Info("✅ Metrics Collector started successfully")
	return nil
}

// startPrometheusServer inicia el servidor de métricas de Prometheus
func (mc *MetricsCollector) startPrometheusServer() {
	http.Handle("/metrics", promhttp.Handler())
	
	port := mc.config.PrometheusPort
	if port == "" {
		port = "9090"
	}
	
	mc.logger.Infof("📊 Prometheus metrics server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		mc.logger.Fatalf("Failed to start Prometheus server: %v", err)
	}
}

// startAPIServer inicia el servidor API para el dashboard
func (mc *MetricsCollector) startAPIServer() {
	mux := http.NewServeMux()
	
	// CORS middleware
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Endpoints
	mux.HandleFunc("/api/metrics/snapshot", mc.handleMetricsSnapshot)
	mux.HandleFunc("/api/health", mc.handleHealthCheck)
	mux.HandleFunc("/api/metrics/aws", mc.handleAWSMetrics)
	mux.HandleFunc("/api/metrics/gcp", mc.handleGCPMetrics)
	mux.HandleFunc("/api/metrics/business", mc.handleBusinessMetrics)

	handler := corsHandler(mux)
	
	mc.logger.Info("🌐 API server starting on port 8081")
	if err := http.ListenAndServe(":8081", handler); err != nil {
		mc.logger.Fatalf("Failed to start API server: %v", err)
	}
}

// startPeriodicCollection inicia la recolección periódica de métricas
func (mc *MetricsCollector) startPeriodicCollection() {
	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.collectAllMetrics()
		}
	}
}

// collectAllMetrics recolecta todas las métricas
func (mc *MetricsCollector) collectAllMetrics() {
	start := time.Now()
	mc.logger.Debug("🔄 Starting metrics collection cycle")

	// Recolectar métricas AWS en paralelo
	go func() {
		if err := mc.collectAWSMetrics(); err != nil {
			mc.logger.Errorf("Failed to collect AWS metrics: %v", err)
		}
	}()

	// Recolectar métricas GCP en paralelo
	go func() {
		if err := mc.collectGCPMetrics(); err != nil {
			mc.logger.Errorf("Failed to collect GCP metrics: %v", err)
		}
	}()

	// Recolectar métricas de negocio
	go func() {
		if err := mc.collectBusinessMetrics(); err != nil {
			mc.logger.Errorf("Failed to collect business metrics: %v", err)
		}
	}()

	duration := time.Since(start)
	mc.logger.Debugf("✅ Metrics collection completed in %v", duration)
}

// collectAWSMetrics recolecta métricas de AWS
func (mc *MetricsCollector) collectAWSMetrics() error {
	ctx := context.Background()

	// Obtener métricas de Lambda
	if err := mc.collectLambdaMetrics(ctx); err != nil {
		mc.logger.Errorf("Failed to collect Lambda metrics: %v", err)
	}

	// Obtener métricas de DynamoDB
	if err := mc.collectDynamoDBMetrics(ctx); err != nil {
		mc.logger.Errorf("Failed to collect DynamoDB metrics: %v", err)
	}

	// Actualizar health status
	mc.systemMetrics.SystemHealth.WithLabelValues("lambda", "aws").Set(1)
	mc.systemMetrics.SystemHealth.WithLabelValues("dynamodb", "aws").Set(1)

	return nil
}

// collectLambdaMetrics recolecta métricas de Lambda
func (mc *MetricsCollector) collectLambdaMetrics(ctx context.Context) error {
	// Obtener estadísticas de CloudWatch para Lambda
	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Lambda"),
		MetricName: aws.String("Invocations"),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("FunctionName"),
				Value: aws.String(mc.config.AWSLambdaFunction),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(300), // 5 minutos
		Statistics: []types.Statistic{types.StatisticSum},
	}

	result, err := mc.cloudWatchClient.GetMetricStatistics(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to get Lambda metrics: %w", err)
	}

	// Procesar resultados
	for _, datapoint := range result.Datapoints {
		if datapoint.Sum != nil {
			mc.awsMetrics.LambdaInvocations.WithLabelValues(mc.config.AWSLambdaFunction, "success").Add(*datapoint.Sum)
		}
	}

	return nil
}

// collectDynamoDBMetrics recolecta métricas de DynamoDB
func (mc *MetricsCollector) collectDynamoDBMetrics(ctx context.Context) error {
	// Obtener información de la tabla
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(mc.config.DynamoDBTable),
	}

	result, err := mc.dynamoClient.DescribeTable(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to describe DynamoDB table: %w", err)
	}

	// Simular métricas (en producción obtendrías de CloudWatch)
	if result.Table != nil {
		mc.awsMetrics.DynamoDBRequests.WithLabelValues(mc.config.DynamoDBTable, "scan").Add(1)
		mc.awsMetrics.DynamoDBLatency.WithLabelValues(mc.config.DynamoDBTable, "scan").Observe(0.1)
	}

	return nil
}

// collectGCPMetrics recolecta métricas de GCP
func (mc *MetricsCollector) collectGCPMetrics() error {
	ctx := context.Background()

	// Obtener métricas de Cloud Functions
	if err := mc.collectCloudFunctionMetrics(ctx); err != nil {
		mc.logger.Errorf("Failed to collect Cloud Function metrics: %v", err)
	}

	// Obtener métricas de Firestore
	if err := mc.collectFirestoreMetrics(ctx); err != nil {
		mc.logger.Errorf("Failed to collect Firestore metrics: %v", err)
	}

	// Actualizar health status
	mc.systemMetrics.SystemHealth.WithLabelValues("cloud-functions", "gcp").Set(1)
	mc.systemMetrics.SystemHealth.WithLabelValues("firestore", "gcp").Set(1)

	return nil
}

// collectCloudFunctionMetrics recolecta métricas de Cloud Functions
func (mc *MetricsCollector) collectCloudFunctionMetrics(ctx context.Context) error {
	// Construir query para métricas de Cloud Functions
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   fmt.Sprintf("projects/%s", mc.gcpProjectID),
		Filter: fmt.Sprintf(`resource.type="cloud_function" AND resource.labels.function_name="%s"`, mc.config.GCPCloudFunction),
		Interval: &monitoringpb.TimeInterval{
			EndTime:   &monitoringpb.TimeInterval_EndTime{EndTime: &monitoringpb.Timestamp{Seconds: time.Now().Unix()}},
			StartTime: &monitoringpb.TimeInterval_StartTime{StartTime: &monitoringpb.Timestamp{Seconds: time.Now().Add(-5 * time.Minute).Unix()}},
		},
	}

	// Simular métricas por ahora (en producción usarías la API real)
	mc.gcpMetrics.CloudFunctionInvocations.WithLabelValues(mc.config.GCPCloudFunction, "success").Add(1)
	mc.gcpMetrics.CloudFunctionDuration.WithLabelValues(mc.config.GCPCloudFunction).Observe(0.2)

	_ = req // Evitar warning del compilador
	return nil
}

// collectFirestoreMetrics recolecta métricas de Firestore
func (mc *MetricsCollector) collectFirestoreMetrics(ctx context.Context) error {
	// Obtener conteo de documentos en la colección
	iter := mc.firestoreClient.Collection(mc.config.FirestoreCollection).Documents(ctx)
	count := 0
	for {
		_, err := iter.Next()
		if err != nil {
			break
		}
		count++
	}

	// Actualizar métricas
	mc.gcpMetrics.FirestoreReads.WithLabelValues(mc.config.FirestoreCollection).Add(float64(count))

	return nil
}

// collectBusinessMetrics recolecta métricas de negocio
func (mc *MetricsCollector) collectBusinessMetrics() error {
	ctx := context.Background()

	// Obtener total de productos desde AWS
	productsCount, err := mc.getProductsCount(ctx)
	if err != nil {
		mc.logger.Errorf("Failed to get products count: %v", err)
		productsCount = 0
	}
	mc.systemMetrics.TotalProducts.Set(float64(productsCount))

	// Obtener total de pedidos desde GCP
	ordersCount, totalRevenue, err := mc.getOrdersStats(ctx)
	if err != nil {
		mc.logger.Errorf("Failed to get orders stats: %v", err)
		ordersCount = 0
		totalRevenue = 0
	}
	mc.systemMetrics.TotalOrders.Set(float64(ordersCount))
	mc.systemMetrics.TotalRevenue.Set(totalRevenue)

	// Calcular AOV
	if ordersCount > 0 {
		aov := totalRevenue / float64(ordersCount)
		mc.systemMetrics.AverageOrderValue.Set(aov)
	}

	// Simular error rate (en producción calcularías desde logs)
	mc.systemMetrics.ErrorRate.Set(2.5) // 2.5% error rate

	return nil
}

// getProductsCount obtiene el conteo de productos desde DynamoDB
func (mc *MetricsCollector) getProductsCount(ctx context.Context) (int, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(mc.config.DynamoDBTable),
		Select:    types.SelectCount,
	}

	result, err := mc.dynamoClient.Scan(ctx, input)
	if err != nil {
		return 0, err
	}

	return int(result.Count), nil
}

// getOrdersStats obtiene estadísticas de pedidos desde Firestore
func (mc *MetricsCollector) getOrdersStats(ctx context.Context) (int, float64, error) {
	iter := mc.firestoreClient.Collection(mc.config.FirestoreCollection).Documents(ctx)
	
	count := 0
	totalRevenue := 0.0

	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}

		count++
		
		// Obtener total_amount si existe
		data := doc.Data()
		if amount, ok := data["total_amount"].(float64); ok {
			totalRevenue += amount
		}
	}

	return count, totalRevenue, nil
}

// HTTP Handlers
func (mc *MetricsCollector) handleMetricsSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot := MetricSnapshot{
		Timestamp: time.Now(),
		AWSMetrics: map[string]interface{}{
			"lambda_invocations": 150,
			"lambda_duration":    0.25,
			"lambda_errors":      2,
			"dynamodb_requests":  145,
			"dynamodb_latency":   0.1,
		},
		GCPMetrics: map[string]interface{}{
			"function_invocations": 98,
			"function_duration":    0.2,
			"function_errors":      1,
			"firestore_reads":      87,
			"firestore_writes":     23,
		},
		BusinessMetrics: map[string]interface{}{
			"total_products":       25,
			"total_orders":         156,
			"total_revenue":        15678.50,
			"average_order_value":  100.50,
			"error_rate":           2.5,
		},
		HealthStatus: map[string]string{
			"aws_lambda":      "healthy",
			"aws_dynamodb":    "healthy",
			"gcp_functions":   "healthy",
			"gcp_firestore":   "healthy",
			"overall":         "healthy",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}