package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type ProductHandler struct {
	dynamoClient *dynamodb.Client
	tableName    string
}

func NewProductHandler(dynamoClient *dynamodb.Client) *ProductHandler {
	return &ProductHandler{
		dynamoClient: dynamoClient,
		tableName:    TableName,
	}
}

// HandleRequest maneja todas las rutas de la API
func (h *ProductHandler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// CORS headers
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	// Handle preflight OPTIONS request
	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    headers,
			Body:       "",
		}, nil
	}

	// Router básico
	switch {
	case request.HTTPMethod == "GET" && request.PathParameters == nil:
		// GET /products
		return h.listProducts(ctx, request, headers)
		
	case request.HTTPMethod == "GET" && request.PathParameters["id"] != "":
		// GET /products/{id}
		return h.getProduct(ctx, request, headers)
		
	case request.HTTPMethod == "POST":
		// POST /products
		return h.createProduct(ctx, request, headers)
		
	case request.HTTPMethod == "PUT" && request.PathParameters["id"] != "":
		// PUT /products/{id}
		return h.updateProduct(ctx, request, headers)
		
	case request.HTTPMethod == "DELETE" && request.PathParameters["id"] != "":
		// DELETE /products/{id}
		return h.deleteProduct(ctx, request, headers)
		
	case request.HTTPMethod == "GET" && strings.Contains(request.Path, "/stats"):
		// GET /products/stats
		return h.getProductStats(ctx, request, headers)
		
	default:
		return h.errorResponse(headers, 404, "Route not found"), nil
	}
}

// listProducts lista todos los productos con filtros
func (h *ProductHandler) listProducts(ctx context.Context, request events.APIGatewayProxyRequest, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	// Parsear query parameters
	filter := ProductFilter{}
	if category := request.QueryStringParameters["category"]; category != "" {
		filter.Category = category
	}
	if status := request.QueryStringParameters["status"]; status != "" {
		filter.Status = status
	}
	if search := request.QueryStringParameters["search"]; search != "" {
		filter.Search = search
	}
	if page := request.QueryStringParameters["page"]; page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			filter.Page = p
		}
	}
	if pageSize := request.QueryStringParameters["page_size"]; pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			filter.PageSize = ps
		}
	}
	
	filter.Validate()

	// Scan DynamoDB (en producción usaríamos Query con índices)
	input := &dynamodb.ScanInput{
		TableName: aws.String(h.tableName),
	}

	// Aplicar filtros
	var filterExpression strings.Builder
	expressionAttributeValues := make(map[string]types.AttributeValue)
	
	if filter.Category != "" {
		filterExpression.WriteString("category = :category")
		expressionAttributeValues[":category"] = &types.AttributeValueMemberS{Value: filter.Category}
	}
	
	if filter.Status != "" {
		if filterExpression.Len() > 0 {
			filterExpression.WriteString(" AND ")
		}
		filterExpression.WriteString("#status = :status")
		expressionAttributeValues[":status"] = &types.AttributeValueMemberS{Value: filter.Status}
		input.ExpressionAttributeNames = map[string]string{"#status": "status"}
	}

	if filterExpression.Len() > 0 {
		input.FilterExpression = aws.String(filterExpression.String())
		input.ExpressionAttributeValues = expressionAttributeValues
	}

	result, err := h.dynamoClient.Scan(ctx, input)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error scanning products: %v", err)), nil
	}

	// Deserializar productos
	var products []Product
	err = attributevalue.UnmarshalListOfMaps(result.Items, &products)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error unmarshaling products: %v", err)), nil
	}

	// Filtrar por búsqueda si se especifica
	if filter.Search != "" {
		filteredProducts := []Product{}
		searchLower := strings.ToLower(filter.Search)
		for _, p := range products {
			if strings.Contains(strings.ToLower(p.Name), searchLower) || 
			   strings.Contains(strings.ToLower(p.Description), searchLower) {
				filteredProducts = append(filteredProducts, p)
			}
		}
		products = filteredProducts
	}

	// Paginación manual (en producción usaríamos DynamoDB pagination)
	total := len(products)
	start := (filter.Page - 1) * filter.PageSize
	end := start + filter.PageSize
	
	if start > total {
		products = []Product{}
	} else {
		if end > total {
			end = total
		}
		products = products[start:end]
	}

	totalPages := (total + filter.PageSize - 1) / filter.PageSize

	response := ProductsListResponse{
		Success:    true,
		Message:    "Products retrieved successfully",
		Data:       products,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}

	return h.successResponse(headers, response), nil
}

// getProduct obtiene un producto por ID
func (h *ProductHandler) getProduct(ctx context.Context, request events.APIGatewayProxyRequest, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	productID := request.PathParameters["id"]
	if productID == "" {
		return h.errorResponse(headers, 400, "Product ID is required"), nil
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(h.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: productID},
		},
	}

	result, err := h.dynamoClient.GetItem(ctx, input)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error getting product: %v", err)), nil
	}

	if result.Item == nil {
		return h.errorResponse(headers, 404, "Product not found"), nil
	}

	var product Product
	err = attributevalue.UnmarshalMap(result.Item, &product)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error unmarshaling product: %v", err)), nil
	}

	response := ProductResponse{
		Success: true,
		Message: "Product retrieved successfully",
		Data:    product,
	}

	return h.successResponse(headers, response), nil
}

// createProduct crea un nuevo producto
func (h *ProductHandler) createProduct(ctx context.Context, request events.APIGatewayProxyRequest, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	var req CreateProductRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return h.errorResponse(headers, 400, "Invalid JSON body"), nil
	}

	// Validaciones básicas
	if req.Name == "" || req.Price <= 0 || req.Category == "" || req.SKU == "" {
		return h.errorResponse(headers, 400, "Missing required fields: name, price, category, sku"), nil
	}

	// Crear producto
	now := time.Now()
	product := Product{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Stock:       req.Stock,
		ImageURL:    req.ImageURL,
		SKU:         req.SKU,
		Status:      StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        req.Tags,
	}

	if product.Stock <= 0 {
		product.Status = StatusOutOfStock
	}

	// Serializar para DynamoDB
	item, err := attributevalue.MarshalMap(product)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error marshaling product: %v", err)), nil
	}

	// Insertar en DynamoDB
	input := &dynamodb.PutItemInput{
		TableName: aws.String(h.tableName),
		Item:      item,
		ConditionExpression: aws.String("attribute_not_exists(id)"), // Evitar duplicados
	}

	_, err = h.dynamoClient.PutItem(ctx, input)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error creating product: %v", err)), nil
	}

	response := ProductResponse{
		Success: true,
		Message: "Product created successfully",
		Data:    product,
	}

	return h.successResponse(headers, response), nil
}

// updateProduct actualiza un producto existente
func (h *ProductHandler) updateProduct(ctx context.Context, request events.APIGatewayProxyRequest, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	productID := request.PathParameters["id"]
	if productID == "" {
		return h.errorResponse(headers, 400, "Product ID is required"), nil
	}

	var req UpdateProductRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return h.errorResponse(headers, 400, "Invalid JSON body"), nil
	}

	// Construir update expression
	var updateExpression strings.Builder
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeNames := make(map[string]string)
	
	updateExpression.WriteString("SET updated_at = :updated_at")
	expressionAttributeValues[":updated_at"] = &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)}

	if req.Name != nil {
		updateExpression.WriteString(", #name = :name")
		expressionAttributeNames["#name"] = "name"
		expressionAttributeValues[":name"] = &types.AttributeValueMemberS{Value: *req.Name}
	}
	
	if req.Description != nil {
		updateExpression.WriteString(", description = :description")
		expressionAttributeValues[":description"] = &types.AttributeValueMemberS{Value: *req.Description}
	}
	
	if req.Price != nil {
		updateExpression.WriteString(", price = :price")
		expressionAttributeValues[":price"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", *req.Price)}
	}
	
	if req.Category != nil {
		updateExpression.WriteString(", category = :category")
		expressionAttributeValues[":category"] = &types.AttributeValueMemberS{Value: *req.Category}
	}
	
	if req.Stock != nil {
		updateExpression.WriteString(", stock = :stock")
		expressionAttributeValues[":stock"] = &types.AttributeValueMemberN{Value: strconv.Itoa(*req.Stock)}
	}
	
	if req.ImageURL != nil {
		updateExpression.WriteString(", image_url = :image_url")
		expressionAttributeValues[":image_url"] = &types.AttributeValueMemberS{Value: *req.ImageURL}
	}
	
	if req.Status != nil {
		updateExpression.WriteString(", #status = :status")
		expressionAttributeNames["#status"] = "status"
		expressionAttributeValues[":status"] = &types.AttributeValueMemberS{Value: *req.Status}
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(h.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: productID},
		},
		UpdateExpression:          aws.String(updateExpression.String()),
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression:       aws.String("attribute_exists(id)"), // Solo actualizar si existe
		ReturnValues:              types.ReturnValueAllNew,
	}

	if len(expressionAttributeNames) > 0 {
		input.ExpressionAttributeNames = expressionAttributeNames
	}

	result, err := h.dynamoClient.UpdateItem(ctx, input)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error updating product: %v", err)), nil
	}

	var product Product
	err = attributevalue.UnmarshalMap(result.Attributes, &product)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error unmarshaling updated product: %v", err)), nil
	}

	response := ProductResponse{
		Success: true,
		Message: "Product updated successfully",
		Data:    product,
	}

	return h.successResponse(headers, response), nil
}

// deleteProduct elimina un producto
func (h *ProductHandler) deleteProduct(ctx context.Context, request events.APIGatewayProxyRequest, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	productID := request.PathParameters["id"]
	if productID == "" {
		return h.errorResponse(headers, 400, "Product ID is required"), nil
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(h.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: productID},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
		ReturnValues:        types.ReturnValueAllOld,
	}

	result, err := h.dynamoClient.DeleteItem(ctx, input)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error deleting product: %v", err)), nil
	}

	if result.Attributes == nil {
		return h.errorResponse(headers, 404, "Product not found"), nil
	}

	response := ProductResponse{
		Success: true,
		Message: "Product deleted successfully",
	}

	return h.successResponse(headers, response), nil
}

// getProductStats obtiene estadísticas de productos
func (h *ProductHandler) getProductStats(ctx context.Context, request events.APIGatewayProxyRequest, headers map[string]string) (events.APIGatewayProxyResponse, error) {
	// Scan todos los productos para calcular stats
	input := &dynamodb.ScanInput{
		TableName: aws.String(h.tableName),
	}

	result, err := h.dynamoClient.Scan(ctx, input)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error scanning products: %v", err)), nil
	}

	var products []Product
	err = attributevalue.UnmarshalListOfMaps(result.Items, &products)
	if err != nil {
		return h.errorResponse(headers, 500, fmt.Sprintf("Error unmarshaling products: %v", err)), nil
	}

	// Calcular estadísticas
	stats := ProductStats{
		TotalProducts: len(products),
	}

	if len(products) > 0 {
		totalValue := 0.0
		activeCount := 0
		outOfStockCount := 0
		lowStockCount := 0
		categoryCount := make(map[string]int)

		for _, p := range products {
			totalValue += p.Price * float64(p.Stock)
			
			if p.Status == StatusActive {
				activeCount++
			}
			if p.Status == StatusOutOfStock {
				outOfStockCount++
			}
			if p.IsLowStock() {
				lowStockCount++
			}
			
			categoryCount[p.Category]++
		}

		stats.ActiveProducts = activeCount
		stats.OutOfStock = outOfStockCount
		stats.TotalValue = totalValue
		stats.AveragePrice = totalValue / float64(len(products))
		stats.LowStockItems = lowStockCount

		// Encontrar categoría top
		maxCount := 0
		for category, count := range categoryCount {
			if count > maxCount {
				maxCount = count
				stats.TopCategory = category
			}
		}
	}

	response := ProductResponse{
		Success: true,
		Message: "Product statistics retrieved successfully",
		Data:    stats,
	}

	return h.successResponse(headers, response), nil
}

// Helper functions
func (h *ProductHandler) successResponse(headers map[string]string, data interface{}) events.APIGatewayProxyResponse {
	body, _ := json.Marshal(data)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(body),
	}
}

func (h *ProductHandler) errorResponse(headers map[string]string, statusCode int, message string) events.APIGatewayProxyResponse {
	response := ErrorResponse{
		Success: false,
		Message: message,
	}
	body, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       string(body),
	}
}