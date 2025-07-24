module ecommerce-multicloud-monitoring

go 1.22.2

require (
    // AWS SDK
	github.com/aws/aws-lambda-go v1.41.0
	github.com/aws/aws-sdk-go-v2 v1.21.0
	github.com/aws/aws-sdk-go-v2/config v1.18.39
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.21.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.38.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.39.7
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.27.8
	
	// Google Cloud SDK
	cloud.google.com/go/functions v1.15.4
	cloud.google.com/go/firestore v1.13.0
	cloud.google.com/go/storage v1.33.0
	cloud.google.com/go/monitoring v1.16.0
	
	// Web Framework
	github.com/gin-gonic/gin v1.9.1
	github.com/gorilla/websocket v1.5.0
	
	// Monitoring & Metrics
	github.com/prometheus/client_golang v1.17.0
	github.com/sirupsen/logrus v1.9.3
	
	// Configuration
	github.com/spf13/viper v1.17.0
	github.com/joho/godotenv v1.5.1
	
	// Database
	gorm.io/gorm v1.25.5
	gorm.io/driver/sqlite v1.5.4
	
	// Utilities
	github.com/google/uuid v1.3.1
	github.com/stretchr/testify v1.8.4

)

