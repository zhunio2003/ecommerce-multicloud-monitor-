package orderprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type OrderHandler struct {
	firestoreClient *firestore.Client
	projectID       string
}

func NewOrderHandler(firestoreClient *firestore.Client, projectID string) *OrderHandler {
	return &OrderHandler{
		firestoreClient: firestoreClient,
		projectID:       projectID,
	}
}

// HandleHTTP maneja las peticiones HTTP para Cloud Functions
func (h *OrderHandler) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if workflow.Status != WorkflowFailed {
		workflow.Status = WorkflowCompleted
		completedAt := time.Now()
		workflow.CompletedAt = &completedAt

		// Actualizar estado del pedido
		updates := []firestore.Update{
			{Path: "status", Value: StatusProcessing},
			{Path: "payment_status", Value: PaymentCompleted},
			{Path: "processed_at", Value: completedAt},
			{Path: "updated_at", Value: completedAt},
		}

		_, err := h.firestoreClient.Collection(CollectionOrders).Doc(orderID).Update(ctx, updates)
		if err != nil {
			log.Printf("Error updating order status: %v", err)
		} else {
			log.Printf("✅ Order workflow completed successfully for: %s", orderID)
		}
	}
}

// Helper functions
func (h *OrderHandler) successResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func (h *OrderHandler) errorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := OrderResponse{
		Success: false,
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}ctx := r.Context()

	// Router básico basado en path y método
	path := strings.TrimPrefix(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")

	switch {
	case r.Method == "GET" && len(pathParts) == 1 && pathParts[0] == "orders":
		// GET /orders
		h.listOrders(ctx, w, r)
		
	case r.Method == "GET" && len(pathParts) == 2 && pathParts[0] == "orders":
		// GET /orders/{id}
		h.getOrder(ctx, w, r, pathParts[1])
		
	case r.Method == "POST" && len(pathParts) == 1 && pathParts[0] == "orders":
		// POST /orders
		h.createOrder(ctx, w, r)
		
	case r.Method == "PUT" && len(pathParts) == 2 && pathParts[0] == "orders":
		// PUT /orders/{id}
		h.updateOrder(ctx, w, r, pathParts[1])
		
	case r.Method == "DELETE" && len(pathParts) == 2 && pathParts[0] == "orders":
		// DELETE /orders/{id}
		h.cancelOrder(ctx, w, r, pathParts[1])
		
	case r.Method == "GET" && len(pathParts) == 2 && pathParts[0] == "orders" && pathParts[1] == "stats":
		// GET /orders/stats
		h.getOrderStats(ctx, w, r)
		
	case r.Method == "POST" && len(pathParts) == 3 && pathParts[0] == "orders" && pathParts[2] == "process":
		// POST /orders/{id}/process
		h.processOrder(ctx, w, r, pathParts[1])
		
	default:
		h.errorResponse(w, http.StatusNotFound, "Route not found")
	}
}

// listOrders lista todos los pedidos con filtros
func (h *OrderHandler) listOrders(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Parsear query parameters
	filter := OrderFilter{}
	query := r.URL.Query()
	
	if userID := query.Get("user_id"); userID != "" {
		filter.UserID = userID
	}
	if status := query.Get("status"); status != "" {
		filter.Status = status
	}
	if paymentStatus := query.Get("payment_status"); paymentStatus != "" {
		filter.PaymentStatus = paymentStatus
	}
	if page := query.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			filter.Page = p
		}
	}
	if pageSize := query.Get("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			filter.PageSize = ps
		}
	}
	
	filter.Validate()

	// Construir query de Firestore
	ordersRef := h.firestoreClient.Collection(CollectionOrders)
	query_builder := ordersRef.Query

	// Aplicar filtros
	if filter.UserID != "" {
		query_builder = query_builder.Where("user_id", "==", filter.UserID)
	}
	if filter.Status != "" {
		query_builder = query_builder.Where("status", "==", filter.Status)
	}
	if filter.PaymentStatus != "" {
		query_builder = query_builder.Where("payment_status", "==", filter.PaymentStatus)
	}

	// Ordenar
	if filter.SortOrder == "desc" {
		query_builder = query_builder.OrderBy(filter.SortBy, firestore.Desc)
	} else {
		query_builder = query_builder.OrderBy(filter.SortBy, firestore.Asc)
	}

	// Obtener documentos
	docs, err := query_builder.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Error querying orders: %v", err)
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error querying orders: %v", err))
		return
	}

	// Convertir documentos a orders
	var orders []Order
	for _, doc := range docs {
		var order Order
		if err := doc.DataTo(&order); err != nil {
			log.Printf("Error converting document to order: %v", err)
			continue
		}
		order.ID = doc.Ref.ID
		orders = append(orders, order)
	}

	// Paginación manual (en producción usaríamos Firestore pagination)
	total := len(orders)
	start := (filter.Page - 1) * filter.PageSize
	end := start + filter.PageSize
	
	if start > total {
		orders = []Order{}
	} else {
		if end > total {
			end = total
		}
		orders = orders[start:end]
	}

	totalPages := (total + filter.PageSize - 1) / filter.PageSize

	response := OrdersListResponse{
		Success:    true,
		Message:    "Orders retrieved successfully",
		Data:       orders,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}

	h.successResponse(w, response)
}

// getOrder obtiene un pedido por ID
func (h *OrderHandler) getOrder(ctx context.Context, w http.ResponseWriter, r *http.Request, orderID string) {
	if orderID == "" {
		h.errorResponse(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	doc, err := h.firestoreClient.Collection(CollectionOrders).Doc(orderID).Get(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, http.StatusNotFound, "Order not found")
		} else {
			h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error getting order: %v", err))
		}
		return
	}

	var order Order
	if err := doc.DataTo(&order); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error converting order: %v", err))
		return
	}
	order.ID = doc.Ref.ID

	response := OrderResponse{
		Success: true,
		Message: "Order retrieved successfully",
		Data:    order,
	}

	h.successResponse(w, response)
}

// createOrder crea un nuevo pedido
func (h *OrderHandler) createOrder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// Validaciones básicas
	if req.UserID == "" || req.UserEmail == "" || len(req.Items) == 0 {
		h.errorResponse(w, http.StatusBadRequest, "Missing required fields: user_id, user_email, items")
		return
	}

	// Crear pedido
	now := time.Now()
	order := Order{
		ID:            uuid.New().String(),
		UserID:        req.UserID,
		UserEmail:     req.UserEmail,
		Status:        StatusPending,
		Items:         req.Items,
		Currency:      DefaultCurrency,
		PaymentMethod: req.PaymentMethod,
		PaymentStatus: PaymentPending,
		ShippingInfo:  req.ShippingInfo,
		CreatedAt:     now,
		UpdatedAt:     now,
		Notes:         req.Notes,
	}

	// Calcular total
	order.CalculateTotal()

	// Guardar en Firestore
	_, err := h.firestoreClient.Collection(CollectionOrders).Doc(order.ID).Set(ctx, order)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error creating order: %v", err))
		return
	}

	// Iniciar workflow de procesamiento (async)
	go h.startOrderWorkflow(context.Background(), order.ID)

	response := OrderResponse{
		Success: true,
		Message: "Order created successfully",
		Data:    order,
	}

	h.successResponse(w, response)
}

// updateOrder actualiza un pedido existente
func (h *OrderHandler) updateOrder(ctx context.Context, w http.ResponseWriter, r *http.Request, orderID string) {
	if orderID == "" {
		h.errorResponse(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	var req UpdateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// Construir updates
	updates := []firestore.Update{
		{Path: "updated_at", Value: time.Now()},
	}

	if req.Status != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: *req.Status})
	}
	if req.PaymentStatus != nil {
		updates = append(updates, firestore.Update{Path: "payment_status", Value: *req.PaymentStatus})
	}
	if req.ShippingInfo != nil {
		updates = append(updates, firestore.Update{Path: "shipping_info", Value: *req.ShippingInfo})
	}
	if req.Notes != nil {
		updates = append(updates, firestore.Update{Path: "notes", Value: *req.Notes})
	}
	if req.TrackingID != nil {
		updates = append(updates, firestore.Update{Path: "shipping_info.tracking_id", Value: *req.TrackingID})
	}

	// Actualizar en Firestore
	_, err := h.firestoreClient.Collection(CollectionOrders).Doc(orderID).Update(ctx, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, http.StatusNotFound, "Order not found")
		} else {
			h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error updating order: %v", err))
		}
		return
	}

	// Obtener pedido actualizado
	doc, err := h.firestoreClient.Collection(CollectionOrders).Doc(orderID).Get(ctx)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error getting updated order: %v", err))
		return
	}

	var order Order
	if err := doc.DataTo(&order); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error converting order: %v", err))
		return
	}
	order.ID = doc.Ref.ID

	response := OrderResponse{
		Success: true,
		Message: "Order updated successfully",
		Data:    order,
	}

	h.successResponse(w, response)
}

// cancelOrder cancela un pedido
func (h *OrderHandler) cancelOrder(ctx context.Context, w http.ResponseWriter, r *http.Request, orderID string) {
	if orderID == "" {
		h.errorResponse(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	// Obtener pedido actual
	doc, err := h.firestoreClient.Collection(CollectionOrders).Doc(orderID).Get(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, http.StatusNotFound, "Order not found")
		} else {
			h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error getting order: %v", err))
		}
		return
	}

	var order Order
	if err := doc.DataTo(&order); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error converting order: %v", err))
		return
	}
	order.ID = doc.Ref.ID

	// Verificar si se puede cancelar
	if !order.CanBeCancelled() {
		h.errorResponse(w, http.StatusBadRequest, "Order cannot be cancelled in current status")
		return
	}

	// Actualizar status
	updates := []firestore.Update{
		{Path: "status", Value: StatusCancelled},
		{Path: "updated_at", Value: time.Now()},
	}

	_, err = h.firestoreClient.Collection(CollectionOrders).Doc(orderID).Update(ctx, updates)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error cancelling order: %v", err))
		return
	}

	response := OrderResponse{
		Success: true,
		Message: "Order cancelled successfully",
	}

	h.successResponse(w, response)
}

// processOrder procesa un pedido manualmente
func (h *OrderHandler) processOrder(ctx context.Context, w http.ResponseWriter, r *http.Request, orderID string) {
	if orderID == "" {
		h.errorResponse(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	// Iniciar workflow de procesamiento
	go h.startOrderWorkflow(context.Background(), orderID)

	response := OrderResponse{
		Success: true,
		Message: "Order processing started",
	}

	h.successResponse(w, response)
}

// getOrderStats obtiene estadísticas de pedidos
func (h *OrderHandler) getOrderStats(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Obtener todos los pedidos para calcular stats
	docs, err := h.firestoreClient.Collection(CollectionOrders).Documents(ctx).GetAll()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error querying orders: %v", err))
		return
	}

	// Calcular estadísticas
	stats := OrderStats{
		TotalOrders: len(docs),
	}

	if len(docs) > 0 {
		totalRevenue := 0.0
		statusCount := make(map[string]int)
		paymentMethodCount := make(map[string]int)
		recent24h := 0

		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)

		for _, doc := range docs {
			var order Order
			if err := doc.DataTo(&order); err != nil {
				continue
			}

			totalRevenue += order.TotalAmount
			statusCount[order.Status]++
			paymentMethodCount[order.PaymentMethod]++

			if order.CreatedAt.After(yesterday) {
				recent24h++
			}
		}

		stats.PendingOrders = statusCount[StatusPending]
		stats.ProcessedOrders = statusCount[StatusProcessing]
		stats.CompletedOrders = statusCount[StatusDelivered]
		stats.CancelledOrders = statusCount[StatusCancelled]
		stats.TotalRevenue = totalRevenue
		stats.AverageOrderValue = totalRevenue / float64(len(docs))
		stats.RecentOrders = recent24h

		// Encontrar método de pago más usado
		maxCount := 0
		for method, count := range paymentMethodCount {
			if count > maxCount {
				maxCount = count
				stats.TopPaymentMethod = method
			}
		}
	}

	response := OrderResponse{
		Success: true,
		Message: "Order statistics retrieved successfully",
		Data:    stats,
	}

	h.successResponse(w, response)
}

// startOrderWorkflow inicia el workflow de procesamiento de pedidos
func (h *OrderHandler) startOrderWorkflow(ctx context.Context, orderID string) {
	log.Printf("Starting order workflow for order: %s", orderID)

	workflow := ProcessOrderWorkflow{
		OrderID:   orderID,
		Step:      StepValidateOrder,
		Status:    WorkflowRunning,
		StartedAt: time.Now(),
	}

	// Simular pasos del workflow
	steps := []string{
		StepValidateOrder,
		StepReserveInventory,
		StepProcessPayment,
		StepUpdateInventory,
		StepSendNotifications,
		StepCompleteOrder,
	}

	for _, step := range steps {
		workflow.Step = step
		log.Printf("📋 Processing step: %s for order: %s", step, orderID)

		// Simular procesamiento (en producción sería lógica real)
		time.Sleep(1 * time.Second)

		// Simular algunos fallos ocasionalmente
		if step == StepProcessPayment && time.Now().UnixNano()%10 == 0 {
			log.Printf("Payment failed for order: %s", orderID)
			workflow.Status = WorkflowFailed
			workflow.ErrorMessage = "Payment processing failed"
			break
		}
	}