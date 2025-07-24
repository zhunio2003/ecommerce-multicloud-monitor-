package orderprocessor

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
)

var (
	handler *OrderHandler
)

func init() {
	ctx := context.Background()
	
	// Obtener Project ID
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID or GOOGLE_CLOUD_PROJECT environment variable must be set")
	}

	// Crear cliente de Firestore
	firestoreClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}

	// Inicializar handler
	handler = NewOrderHandler(firestoreClient, projectID)

	log.Println("🚀 Order Processor Cloud Function initialized successfully")
	log.Printf("📊 Project ID: %s", projectID)
	log.Printf("🔥 Firestore: Connected")
}

// ProcessOrder es el punto de entrada para la Cloud Function HTTP
func ProcessOrder(w http.ResponseWriter, r *http.Request) {
	handler.HandleHTTP(w, r)
}

// ProcessOrderPubSub maneja mensajes de Pub/Sub (para procesamiento asíncrono)
func ProcessOrderPubSub(ctx context.Context, m PubSubMessage) error {
	log.Printf("📨 Received Pub/Sub message: %s", string(m.Data))
	
	// Aquí puedes procesar mensajes asíncronos
	// Por ejemplo, para procesar pagos, actualizar inventario, etc.
	
	return nil
}

// PubSubMessage representa un mensaje de Pub/Sub
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// HealthCheck endpoint para verificar que la función está funcionando
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "order-processor",
		"timestamp": "2024-01-01T00:00:00Z",
		"version":   "1.0.0",
	}
	
	w.Write([]byte(`{"status":"healthy","service":"order-processor","version":"1.0.0"}`))
}