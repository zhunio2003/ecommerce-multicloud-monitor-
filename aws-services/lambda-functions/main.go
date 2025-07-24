package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var (
	handler *ProductHandler
)

func init() {
	// Cargar configuración de AWS
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Crear cliente de DynamoDB
	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Inicializar handler
	handler = NewProductHandler(dynamoClient)

	log.Println("🚀 Products API Lambda initialized successfully")
	log.Printf("📊 Table: %s", TableName)
	log.Printf("🌍 Region: %s", cfg.Region)
}

func main() {
	// Verificar si estamos en modo local para testing
	if os.Getenv("LOCAL_MODE") == "true" {
		log.Println("🏠 Running in LOCAL MODE for testing")
		// Aquí podrías agregar un servidor HTTP local para pruebas
		return
	}

	// Iniciar función Lambda
	log.Println("🚀 Starting Products API Lambda function")
	lambda.Start(handler.HandleRequest)
}