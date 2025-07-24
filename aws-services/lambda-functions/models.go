package main

import (
	"time"
)

// Product representa un producto en nuestro e-commerce
type Product struct {
	ID          string    `json:"id" dynamodbav:"id"`
	Name        string    `json:"name" dynamodbav:"name"`
	Description string    `json:"description" dynamodbav:"description"`
	Price       float64   `json:"price" dynamodbav:"price"`
	Category    string    `json:"category" dynamodbav:"category"`
	Stock       int       `json:"stock" dynamodbav:"stock"`
	ImageURL    string    `json:"image_url" dynamodbav:"image_url"`
	SKU         string    `json:"sku" dynamodbav:"sku"`
	Status      string    `json:"status" dynamodbav:"status"` // active, inactive, out_of_stock
	CreatedAt   time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" dynamodbav:"updated_at"`
	Tags        []string  `json:"tags" dynamodbav:"tags"`
}

// CreateProductRequest para crear productos
type CreateProductRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Price       float64  `json:"price" binding:"required,gt=0"`
	Category    string   `json:"category" binding:"required"`
	Stock       int      `json:"stock" binding:"required,gte=0"`
	ImageURL    string   `json:"image_url"`
	SKU         string   `json:"sku" binding:"required"`
	Tags        []string `json:"tags"`
}

// UpdateProductRequest para actualizar productos
type UpdateProductRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Price       *float64  `json:"price,omitempty"`
	Category    *string   `json:"category,omitempty"`
	Stock       *int      `json:"stock,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
}

// ProductResponse respuesta estándar para productos
type ProductResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ProductsListResponse para listar productos
type ProductsListResponse struct {
	Success    bool      `json:"success"`
	Message    string    `json:"message"`
	Data       []Product `json:"data"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// ProductStats estadísticas de productos
type ProductStats struct {
	TotalProducts    int     `json:"total_products"`
	ActiveProducts   int     `json:"active_products"`
	OutOfStock      int     `json:"out_of_stock"`
	TotalValue      float64 `json:"total_value"`
	AveragePrice    float64 `json:"average_price"`
	TopCategory     string  `json:"top_category"`
	LowStockItems   int     `json:"low_stock_items"` // Stock < 10
}

// ValidationError para errores de validación
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse respuesta de error
type ErrorResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// ProductFilter filtros para búsqueda
type ProductFilter struct {
	Category    string  `json:"category,omitempty"`
	MinPrice    float64 `json:"min_price,omitempty"`
	MaxPrice    float64 `json:"max_price,omitempty"`
	Status      string  `json:"status,omitempty"`
	InStock     bool    `json:"in_stock,omitempty"`
	Search      string  `json:"search,omitempty"` // Buscar en nombre/descripción
	Page        int     `json:"page,omitempty"`
	PageSize    int     `json:"page_size,omitempty"`
	SortBy      string  `json:"sort_by,omitempty"`      // name, price, created_at
	SortOrder   string  `json:"sort_order,omitempty"`   // asc, desc
}

// Constants para el sistema
const (
	// Status de productos
	StatusActive      = "active"
	StatusInactive    = "inactive"
	StatusOutOfStock  = "out_of_stock"
	
	// Límites
	MaxPageSize       = 100
	DefaultPageSize   = 20
	LowStockThreshold = 10
	
	// DynamoDB
	TableName         = "products"
	GSIByCategory     = "category-index"
	GSIByStatus       = "status-index"
)

// Helper functions
func (p *Product) IsLowStock() bool {
	return p.Stock <= LowStockThreshold
}

func (p *Product) IsAvailable() bool {
	return p.Status == StatusActive && p.Stock > 0
}

func (p *Product) UpdateStock(quantity int) {
	p.Stock = quantity
	p.UpdatedAt = time.Now()
	
	if p.Stock <= 0 {
		p.Status = StatusOutOfStock
	} else if p.Status == StatusOutOfStock {
		p.Status = StatusActive
	}
}

func (f *ProductFilter) Validate() error {
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