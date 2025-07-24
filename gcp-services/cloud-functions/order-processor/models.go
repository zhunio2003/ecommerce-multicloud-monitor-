package orderprocessor

import (
	"time"
)

// Order representa un pedido en nuestro e-commerce
type Order struct {
	ID            string      `json:"id" firestore:"id"`
	UserID        string      `json:"user_id" firestore:"user_id"`
	UserEmail     string      `json:"user_email" firestore:"user_email"`
	Status        string      `json:"status" firestore:"status"`
	Items         []OrderItem `json:"items" firestore:"items"`
	TotalAmount   float64     `json:"total_amount" firestore:"total_amount"`
	Currency      string      `json:"currency" firestore:"currency"`
	PaymentMethod string      `json:"payment_method" firestore:"payment_method"`
	PaymentStatus string      `json:"payment_status" firestore:"payment_status"`
	ShippingInfo  Shipping    `json:"shipping_info" firestore:"shipping_info"`
	CreatedAt     time.Time   `json:"created_at" firestore:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at" firestore:"updated_at"`
	ProcessedAt   *time.Time  `json:"processed_at,omitempty" firestore:"processed_at"`
	Notes         string      `json:"notes" firestore:"notes"`
}

// OrderItem representa un item dentro de un pedido
type OrderItem struct {
	ProductID   string  `json:"product_id" firestore:"product_id"`
	ProductName string  `json:"product_name" firestore:"product_name"`
	SKU         string  `json:"sku" firestore:"sku"`
	Quantity    int     `json:"quantity" firestore:"quantity"`
	UnitPrice   float64 `json:"unit_price" firestore:"unit_price"`
	TotalPrice  float64 `json:"total_price" firestore:"total_price"`
	ImageURL    string  `json:"image_url" firestore:"image_url"`
}

// Shipping información de envío
type Shipping struct {
	FullName    string `json:"full_name" firestore:"full_name"`
	Address     string `json:"address" firestore:"address"`
	City        string `json:"city" firestore:"city"`
	State       string `json:"state" firestore:"state"`
	PostalCode  string `json:"postal_code" firestore:"postal_code"`
	Country     string `json:"country" firestore:"country"`
	Phone       string `json:"phone" firestore:"phone"`
	Method      string `json:"method" firestore:"method"`       // standard, express, overnight
	TrackingID  string `json:"tracking_id" firestore:"tracking_id"`
	EstimatedAt *time.Time `json:"estimated_at,omitempty" firestore:"estimated_at"`
}

// CreateOrderRequest para crear pedidos
type CreateOrderRequest struct {
	UserID        string      `json:"user_id" binding:"required"`
	UserEmail     string      `json:"user_email" binding:"required,email"`
	Items         []OrderItem `json:"items" binding:"required,dive"`
	PaymentMethod string      `json:"payment_method" binding:"required"`
	ShippingInfo  Shipping    `json:"shipping_info" binding:"required"`
	Notes         string      `json:"notes"`
}

// UpdateOrderRequest para actualizar pedidos
type UpdateOrderRequest struct {
	Status        *string     `json:"status,omitempty"`
	PaymentStatus *string     `json:"payment_status,omitempty"`
	ShippingInfo  *Shipping   `json:"shipping_info,omitempty"`
	Notes         *string     `json:"notes,omitempty"`
	TrackingID    *string     `json:"tracking_id,omitempty"`
}

// OrderResponse respuesta estándar para pedidos
type OrderResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// OrdersListResponse para listar pedidos
type OrdersListResponse struct {
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
	Data       []Order `json:"data"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// OrderStats estadísticas de pedidos
type OrderStats struct {
	TotalOrders       int     `json:"total_orders"`
	PendingOrders     int     `json:"pending_orders"`
	ProcessedOrders   int     `json:"processed_orders"`
	CompletedOrders   int     `json:"completed_orders"`
	CancelledOrders   int     `json:"cancelled_orders"`
	TotalRevenue      float64 `json:"total_revenue"`
	AverageOrderValue float64 `json:"average_order_value"`
	TopPaymentMethod  string  `json:"top_payment_method"`
	RecentOrders      int     `json:"recent_orders_24h"`
}

// PaymentResult resultado del procesamiento de pago
type PaymentResult struct {
	Success       bool      `json:"success"`
	TransactionID string    `json:"transaction_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	ProcessedAt   time.Time `json:"processed_at"`
	ProviderID    string    `json:"provider_id"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// InventoryUpdate para actualizar inventario en AWS
type InventoryUpdate struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	Operation string `json:"operation"` // reserve, release, consume
}

// NotificationPayload para enviar notificaciones
type NotificationPayload struct {
	Type      string                 `json:"type"`      // email, sms, push
	Recipient string                 `json:"recipient"` // email, phone, user_id
	Template  string                 `json:"template"`  // order_created, order_shipped, etc
	Data      map[string]interface{} `json:"data"`
	Priority  string                 `json:"priority"`  // low, normal, high
}

// OrderFilter filtros para búsqueda de pedidos
type OrderFilter struct {
	UserID        string    `json:"user_id,omitempty"`
	Status        string    `json:"status,omitempty"`
	PaymentStatus string    `json:"payment_status,omitempty"`
	PaymentMethod string    `json:"payment_method,omitempty"`
	MinAmount     float64   `json:"min_amount,omitempty"`
	MaxAmount     float64   `json:"max_amount,omitempty"`
	DateFrom      time.Time `json:"date_from,omitempty"`
	DateTo        time.Time `json:"date_to,omitempty"`
	Page          int       `json:"page,omitempty"`
	PageSize      int       `json:"page_size,omitempty"`
	SortBy        string    `json:"sort_by,omitempty"`    // created_at, total_amount, status
	SortOrder     string    `json:"sort_order,omitempty"` // asc, desc
}

// Constants para el sistema
const (
	// Status de pedidos
	StatusPending   = "pending"
	StatusProcessing = "processing"
	StatusPaid      = "paid"
	StatusShipped   = "shipped"
	StatusDelivered = "delivered"
	StatusCancelled = "cancelled"
	StatusRefunded  = "refunded"
	
	// Payment Status
	PaymentPending   = "pending"
	PaymentCompleted = "completed"
	PaymentFailed    = "failed"
	PaymentRefunded  = "refunded"
	
	// Payment Methods
	PaymentCreditCard = "credit_card"
	PaymentPayPal     = "paypal"
	PaymentBankTransfer = "bank_transfer"
	PaymentCrypto     = "crypto"
	
	// Shipping Methods
	ShippingStandard  = "standard"
	ShippingExpress   = "express"
	ShippingOvernight = "overnight"
	
	// Firestore Collections
	CollectionOrders = "orders"
	CollectionUsers  = "users"
	
	// Limits
	MaxPageSize     = 100
	DefaultPageSize = 20
	
	// Currency
	DefaultCurrency = "USD"
)

// Helper functions
func (o *Order) CalculateTotal() {
	total := 0.0
	for _, item := range o.Items {
		item.TotalPrice = item.UnitPrice * float64(item.Quantity)
		total += item.TotalPrice
	}
	o.TotalAmount = total
}

func (o *Order) IsPaymentComplete() bool {
	return o.PaymentStatus == PaymentCompleted
}

func (o *Order) CanBeCancelled() bool {
	return o.Status == StatusPending || o.Status == StatusProcessing
}

func (o *Order) CanBeShipped() bool {
	return o.Status == StatusPaid && o.PaymentStatus == PaymentCompleted
}

func (o *Order) GetTotalItems() int {
	total := 0
	for _, item := range o.Items {
		total += item.Quantity
	}
	return total
}

func (f *OrderFilter) Validate() error {
	if f.PageSize <= 0 {
		f.PageSize = DefaultPageSize
	}
	if f.PageSize > MaxPageSize {
		f.PageSize = MaxPageSize
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.SortBy == "" {
		f.SortBy = "created_at"
	}
	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}
	return nil
}

// ProcessOrderWorkflow representa el flujo de procesamiento
type ProcessOrderWorkflow struct {
	OrderID           string    `json:"order_id"`
	Step              string    `json:"step"`
	Status            string    `json:"status"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	RetryCount        int       `json:"retry_count"`
	NextRetryAt       *time.Time `json:"next_retry_at,omitempty"`
	PaymentResult     *PaymentResult `json:"payment_result,omitempty"`
	InventoryUpdates  []InventoryUpdate `json:"inventory_updates,omitempty"`
	NotificationsSent []NotificationPayload `json:"notifications_sent,omitempty"`
}

// Workflow Steps
const (
	StepValidateOrder    = "validate_order"
	StepReserveInventory = "reserve_inventory"
	StepProcessPayment   = "process_payment"
	StepUpdateInventory  = "update_inventory"
	StepSendNotifications = "send_notifications"
	StepCompleteOrder    = "complete_order"
)

// Workflow Status
const (
	WorkflowPending   = "pending"
	WorkflowRunning   = "running"
	WorkflowCompleted = "completed"
	WorkflowFailed    = "failed"
	WorkflowRetrying  = "retrying"
)