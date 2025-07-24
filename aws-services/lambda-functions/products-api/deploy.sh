#!/bin/bash

# =================================
# 🚀 AWS Lambda Deploy Script
# =================================

set -e  # Exit on any error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FUNCTION_NAME="ecommerce-products-api"
REGION="${AWS_REGION:-us-east-1}"
RUNTIME="go1.x"
HANDLER="main"
ROLE_NAME="lambda-products-api-role"
TABLE_NAME="products"
ZIP_FILE="products-api.zip"

echo -e "${BLUE}🚀 Starting AWS Lambda deployment...${NC}"

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if AWS CLI is configured
if ! aws sts get-caller-identity &> /dev/null; then
    print_error "AWS CLI not configured. Please run 'aws configure'"
    exit 1
fi

print_status "AWS CLI configured correctly"

# Get current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${BLUE}📁 Working directory: $SCRIPT_DIR${NC}"

# Step 1: Create DynamoDB table if it doesn't exist
echo -e "${BLUE}📊 Setting up DynamoDB table...${NC}"

if ! aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" &> /dev/null; then
    print_warning "Creating DynamoDB table: $TABLE_NAME"
    
    aws dynamodb create-table \
        --table-name "$TABLE_NAME" \
        --attribute-definitions \
            AttributeName=id,AttributeType=S \
            AttributeName=category,AttributeType=S \
            AttributeName=status,AttributeType=S \
        --key-schema \
            AttributeName=id,KeyType=HASH \
        --global-secondary-indexes \
            IndexName=category-index,KeySchema=[{AttributeName=category,KeyType=HASH}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5} \
            IndexName=status-index,KeySchema=[{AttributeName=status,KeyType=HASH}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5} \
        --provisioned-throughput \
            ReadCapacityUnits=5,WriteCapacityUnits=5 \
        --region "$REGION"
    
    print_status "DynamoDB table created. Waiting for it to be active..."
    
    # Wait for table to be active
    aws dynamodb wait table-exists --table-name "$TABLE_NAME" --region "$REGION"
    print_status "DynamoDB table is active"
else
    print_status "DynamoDB table already exists"
fi

# Step 2: Create IAM role for Lambda if it doesn't exist
echo -e "${BLUE}🔐 Setting up IAM role...${NC}"

if ! aws iam get-role --role-name "$ROLE_NAME" &> /dev/null; then
    print_warning "Creating IAM role: $ROLE_NAME"
    
    # Create trust policy
    cat > trust-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "lambda.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}
EOF

    # Create role
    aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document file://trust-policy.json \
        --region "$REGION"

    # Attach basic Lambda execution policy
    aws iam attach-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

    # Create and attach DynamoDB policy
    cat > dynamodb-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "dynamodb:GetItem",
                "dynamodb:PutItem",
                "dynamodb:UpdateItem",
                "dynamodb:DeleteItem",
                "dynamodb:Scan",
                "dynamodb:Query"
            ],
            "Resource": [
                "arn:aws:dynamodb:$REGION:*:table/$TABLE_NAME",
                "arn:aws:dynamodb:$REGION:*:table/$TABLE_NAME/index/*"
            ]
        }
    ]
}
EOF

    aws iam put-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-name "DynamoDBAccess" \
        --policy-document file://dynamodb-policy.json

    # Clean up policy files
    rm -f trust-policy.json dynamodb-policy.json
    
    print_status "IAM role created and configured"
    
    # Wait a bit for role to propagate
    sleep 10
else
    print_status "IAM role already exists"
fi

# Get role ARN
ROLE_ARN=$(aws iam get-role --role-name "$ROLE_NAME" --query 'Role.Arn' --output text)
print_status "Role ARN: $ROLE_ARN"

# Step 3: Build the Go binary
echo -e "${BLUE}🔨 Building Go binary...${NC}"

# Clean previous builds
rm -f main "$ZIP_FILE"

# Build for Linux (Lambda runtime)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main .

if [ ! -f "main" ]; then
    print_error "Failed to build Go binary"
    exit 1
fi

print_status "Go binary built successfully"

# Step 4: Create deployment package
echo -e "${BLUE}📦 Creating deployment package...${NC}"

zip "$ZIP_FILE" main

if [ ! -f "$ZIP_FILE" ]; then
    print_error "Failed to create ZIP file"
    exit 1
fi

ZIP_SIZE=$(du -h "$ZIP_FILE" | cut -f1)
print_status "Deployment package created: $ZIP_FILE ($ZIP_SIZE)"

# Step 5: Deploy or update Lambda function
echo -e "${BLUE}🚀 Deploying Lambda function...${NC}"

if aws lambda get-function --function-name "$FUNCTION_NAME" --region "$REGION" &> /dev/null; then
    print_warning "Updating existing Lambda function..."
    
    aws lambda update-function-code \
        --function-name "$FUNCTION_NAME" \
        --zip-file "fileb://$ZIP_FILE" \
        --region "$REGION" > /dev/null
    
    # Update configuration if needed
    aws lambda update-function-configuration \
        --function-name "$FUNCTION_NAME" \
        --runtime "$RUNTIME" \
        --handler "$HANDLER" \
        --role "$ROLE_ARN" \
        --timeout 30 \
        --memory-size 128 \
        --environment Variables="{TABLE_NAME=$TABLE_NAME}" \
        --region "$REGION" > /dev/null
    
    print_status "Lambda function updated successfully"
else
    print_warning "Creating new Lambda function..."
    
    aws lambda create-function \
        --function-name "$FUNCTION_NAME" \
        --runtime "$RUNTIME" \
        --role "$ROLE_ARN" \
        --handler "$HANDLER" \
        --zip-file "fileb://$ZIP_FILE" \
        --timeout 30 \
        --memory-size 128 \
        --environment Variables="{TABLE_NAME=$TABLE_NAME}" \
        --region "$REGION" > /dev/null
    
    print_status "Lambda function created successfully"
fi

# Step 6: Create API Gateway (optional)
echo -e "${BLUE}🌐 Setting up API Gateway...${NC}"

API_NAME="ecommerce-products-api"

# Check if API exists
API_ID=$(aws apigateway get-rest-apis --region "$REGION" --query "items[?name=='$API_NAME'].id" --output text)

if [ -z "$API_ID" ] || [ "$API_ID" == "None" ]; then
    print_warning "Creating API Gateway..."
    
    # Create API
    API_ID=$(aws apigateway create-rest-api \
        --name "$API_NAME" \
        --description "E-commerce Products API" \
        --region "$REGION" \
        --query 'id' --output text)
    
    # Get root resource ID
    ROOT_ID=$(aws apigateway get-resources \
        --rest-api-id "$API_ID" \
        --region "$REGION" \
        --query 'items[?path==`/`].id' --output text)
    
    # Create /products resource
    PRODUCTS_RESOURCE_ID=$(aws apigateway create-resource \
        --rest-api-id "$API_ID" \
        --parent-id "$ROOT_ID" \
        --path-part "products" \
        --region "$REGION" \
        --query 'id' --output text)
    
    # Create /{id} resource under /products
    ID_RESOURCE_ID=$(aws apigateway create-resource \
        --rest-api-id "$API_ID" \
        --parent-id "$PRODUCTS_RESOURCE_ID" \
        --path-part "{id}" \
        --region "$REGION" \
        --query 'id' --output text)
    
    # Get Lambda function ARN
    LAMBDA_ARN=$(aws lambda get-function \
        --function-name "$FUNCTION_NAME" \
        --region "$REGION" \
        --query 'Configuration.FunctionArn' --output text)
    
    # Create methods for /products resource
    for METHOD in GET POST OPTIONS; do
        aws apigateway put-method \
            --rest-api-id "$API_ID" \
            --resource-id "$PRODUCTS_RESOURCE_ID" \
            --http-method "$METHOD" \
            --authorization-type NONE \
            --region "$REGION" > /dev/null
        
        aws apigateway put-integration \
            --rest-api-id "$API_ID" \
            --resource-id "$PRODUCTS_RESOURCE_ID" \
            --http-method "$METHOD" \
            --type AWS_PROXY \
            --integration-http-method POST \
            --uri "arn:aws:apigateway:$REGION:lambda:path/2015-03-31/functions/$LAMBDA_ARN/invocations" \
            --region "$REGION" > /dev/null
    done
    
    # Create methods for /products/{id} resource
    for METHOD in GET PUT DELETE OPTIONS; do
        aws apigateway put-method \
            --rest-api-id "$API_ID" \
            --resource-id "$ID_RESOURCE_ID" \
            --http-method "$METHOD" \
            --authorization-type NONE \
            --region "$REGION" > /dev/null
        
        aws apigateway put-integration \
            --rest-api-id "$API_ID" \
            --resource-id "$ID_RESOURCE_ID" \
            --http-method "$METHOD" \
            --type AWS_PROXY \
            --integration-http-method POST \
            --uri "arn:aws:apigateway:$REGION:lambda:path/2015-03-31/functions/$LAMBDA_ARN/invocations" \
            --region "$REGION" > /dev/null
    done
    
    # Deploy API
    aws apigateway create-deployment \
        --rest-api-id "$API_ID" \
        --stage-name "prod" \
        --region "$REGION" > /dev/null
    
    # Add Lambda permission for API Gateway
    aws lambda add-permission \
        --function-name "$FUNCTION_NAME" \
        --statement-id "apigateway-invoke" \
        --action lambda:InvokeFunction \
        --principal apigateway.amazonaws.com \
        --source-arn "arn:aws:execute-api:$REGION:*:$API_ID/*/*" \
        --region "$REGION" > /dev/null
    
    print_status "API Gateway created and deployed"
else
    print_status "API Gateway already exists"
fi

# Step 7: Display deployment information
echo -e "${BLUE}📋 Deployment Summary${NC}"
echo -e "${GREEN}================================${NC}"
echo -e "🚀 Function Name: $FUNCTION_NAME"
echo -e "🌍 Region: $REGION"
echo -e "📊 DynamoDB Table: $TABLE_NAME"
echo -e "🔐 IAM Role: $ROLE_NAME"
echo -e "🌐 API Gateway ID: $API_ID"
echo -e "📡 API Endpoint: https://$API_ID.execute-api.$REGION.amazonaws.com/prod/products"
echo -e "${GREEN}================================${NC}"

# Step 8: Test the deployment
echo -e "${BLUE}🧪 Testing deployment...${NC}"

API_ENDPOINT="https://$API_ID.execute-api.$REGION.amazonaws.com/prod/products"

# Wait a moment for deployment to be ready
sleep 5

# Test GET /products
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_ENDPOINT")

if [ "$HTTP_CODE" == "200" ]; then
    print_status "✅ API is responding correctly (HTTP $HTTP_CODE)"
else
    print_warning "⚠️ API returned HTTP $HTTP_CODE - may need a moment to be fully ready"
fi

# Clean up
rm -f main "$ZIP_FILE"

echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
echo -e "${BLUE}💡 Quick test commands:${NC}"
echo -e "curl -X GET $API_ENDPOINT"
echo -e "curl -X POST $API_ENDPOINT -H 'Content-Type: application/json' -d '{\"name\":\"Test Product\",\"price\":19.99,\"category\":\"electronics\",\"sku\":\"TEST001\",\"stock\":10}'"

print_status "All done! Your Products API is live! 🚀"