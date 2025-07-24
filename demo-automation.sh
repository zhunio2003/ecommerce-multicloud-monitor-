#!/bin/bash

# =================================
# AUTOMATED DEMO SCRIPT
# =================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
DASHBOARD_URL="http://localhost:8080"
COLLECTOR_URL="http://localhost:8081"
AWS_API_URL="https://YOUR-AWS-API.execute-api.us-east-1.amazonaws.com/prod"
GCP_FUNCTION_URL="https://YOUR-GCP-FUNCTION.cloudfunctions.net"

echo -e "${PURPLE}🎭 MULTI-CLOUD DEMO AUTOMATION${NC}"
echo -e "${PURPLE}══════════════════════════════════${NC}"

print_demo_step() {
    echo -e "${CYAN}🎯 DEMO STEP: $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

print_command() {
    echo -e "${PURPLE}🔧 Executing: $1${NC}"
}

wait_for_demo() {
    echo -e "${YELLOW}⏱️  Press ENTER to continue to next demo step...${NC}"
    read -r
}

# Function to make API calls and show results
demo_api_call() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    print_command "curl -X $method $url"
    
    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -w "\n%{http_code}")
    else
        response=$(curl -s -X "$method" "$url" -w "\n%{http_code}")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        print_success "$description - HTTP $http_code"
        echo -e "${CYAN}Response:${NC}"
        echo "$response_body" | jq . 2>/dev/null || echo "$response_body"
    else
        echo -e "${RED}❌ $description failed - HTTP $http_code${NC}"
        echo "$response_body"
    fi
    
    echo ""
}

# Check prerequisites
check_prerequisites() {
    print_demo_step "Checking Prerequisites"
    
    # Check if services are running
    if curl -s -f "$DASHBOARD_URL/api/v1/health" > /dev/null; then
        print_success "Dashboard is running"
    else
        echo -e "${RED}❌ Dashboard not running. Please start with ./start-monitoring.sh${NC}"
        exit 1
    fi
    
    if curl -s -f "$COLLECTOR_URL/api/health" > /dev/null; then
        print_success "Metrics collector is running"
    else
        print_info "Metrics collector not running (optional)"
    fi
    
    # Check if jq is available for JSON formatting
    if command -v jq &> /dev/null; then
        print_success "jq available for JSON formatting"
    else
        print_info "jq not available - JSON responses may not be formatted"
    fi
    
    echo ""
    wait_for_demo
}

# Demo 1: System Health Check
demo_health_check() {
    print_demo_step "System Health Check"
    
    print_info "Checking overall system health..."
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/health" "" "System Health Check"
    
    print_info "Checking metrics collector health..."
    demo_api_call "GET" "$COLLECTOR_URL/api/health" "" "Collector Health Check"
    
    wait_for_demo
}

# Demo 2: Real-time Metrics
demo_metrics() {
    print_demo_step "Real-time Metrics Snapshot"
    
    print_info "Getting complete metrics snapshot..."
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/metrics/snapshot" "" "Metrics Snapshot"
    
    print_info "Getting AWS-specific metrics..."
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/metrics/aws" "" "AWS Metrics"
    
    print_info "Getting GCP-specific metrics..."
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/metrics/gcp" "" "GCP Metrics"
    
    print_info "Getting business metrics..."
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/metrics/business" "" "Business Metrics"
    
    wait_for_demo
}

# Demo 3: AWS Lambda Products API
demo_aws_products() {
    print_demo_step "AWS Lambda - Products API Demo"
    
    print_info "Testing AWS Lambda Products API..."
    
    # Create a sample product
    local product_data='{
        "name": "Demo Laptop",
        "description": "High-performance laptop for demonstrations",
        "price": 1299.99,
        "category": "electronics",
        "sku": "DEMO-LAPTOP-001",
        "stock": 50,
        "tags": ["demo", "laptop", "electronics"]
    }'
    
    if [ "$AWS_API_URL" != "https://YOUR-AWS-API.execute-api.us-east-1.amazonaws.com/prod" ]; then
        print_info "Creating a new product in AWS..."
        demo_api_call "POST" "$AWS_API_URL/products" "$product_data" "Create Product in AWS"
        
        print_info "Listing all products from AWS..."
        demo_api_call "GET" "$AWS_API_URL/products" "" "List Products from AWS"
        
        print_info "Getting product statistics..."
        demo_api_call "GET" "$AWS_API_URL/products/stats" "" "AWS Product Statistics"
    else
        print_info "AWS API URL not configured - using local simulation"
        demo_api_call "POST" "$DASHBOARD_URL/api/v1/simulate/load" "" "Simulate AWS Load"
    fi
    
    wait_for_demo
}

# Demo 4: Google Cloud Functions Orders API
demo_gcp_orders() {
    print_demo_step "Google Cloud Functions - Orders API Demo"
    
    print_info "Testing Google Cloud Functions Orders API..."
    
    # Create a sample order
    local order_data='{
        "user_id": "demo_user_123",
        "user_email": "demo@example.com",
        "payment_method": "credit_card",
        "items": [
            {
                "product_id": "DEMO-LAPTOP-001",
                "product_name": "Demo Laptop",
                "sku": "DEMO-LAPTOP-001",
                "quantity": 1,
                "unit_price": 1299.99,
                "total_price": 1299.99
            }
        ],
        "shipping_info": {
            "full_name": "John Demo",
            "address": "123 Demo Street",
            "city": "Demo City",
            "state": "DC",
            "postal_code": "12345",
            "country": "USA",
            "phone": "+1234567890",
            "method": "standard"
        },
        "notes": "Demo order for presentation"
    }'
    
    if [ "$GCP_FUNCTION_URL" != "https://YOUR-GCP-FUNCTION.cloudfunctions.net" ]; then
        print_info "Creating a new order in GCP..."
        demo_api_call "POST" "$GCP_FUNCTION_URL/orders" "$order_data" "Create Order in GCP"
        
        print_info "Listing all orders from GCP..."
        demo_api_call "GET" "$GCP_FUNCTION_URL/orders" "" "List Orders from GCP"
        
        print_info "Getting order statistics..."
        demo_api_call "GET" "$GCP_FUNCTION_URL/orders/stats" "" "GCP Order Statistics"
    else
        print_info "GCP Function URL not configured - using local simulation"
        demo_api_call "POST" "$DASHBOARD_URL/api/v1/simulate/load" "" "Simulate GCP Load"
    fi
    
    wait_for_demo
}

# Demo 5: Load Simulation
demo_load_simulation() {
    print_demo_step "Load Simulation & Auto-scaling"
    
    print_info "Simulating high traffic load..."
    demo_api_call "POST" "$DASHBOARD_URL/api/v1/simulate/load" "" "Simulate High Load"
    
    print_info "Checking updated metrics after load..."
    sleep 3
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/metrics/snapshot" "" "Metrics After Load"
    
    wait_for_demo
}

# Demo 6: Error Simulation & Alerting
demo_error_simulation() {
    print_demo_step "Error Simulation & Alerting System"
    
    print_info "Simulating system errors..."
    demo_api_call "POST" "$DASHBOARD_URL/api/v1/simulate/error" "" "Simulate System Errors"
    
    print_info "Checking generated alerts..."
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/alerts" "" "Check Active Alerts"
    
    print_info "Creating custom alert..."
    local alert_data='{
        "message": "Demo alert: High response time detected",
        "severity": "warning",
        "service": "demo-service"
    }'
    demo_api_call "POST" "$DASHBOARD_URL/api/v1/simulate/alert" "$alert_data" "Create Custom Alert"
    
    wait_for_demo
}

# Demo 7: Dashboard Integration
demo_dashboard() {
    print_demo_step "Interactive Dashboard Demo"
    
    print_info "Opening dashboard in browser..."
    
    # Try to open browser
    if command -v xdg-open &> /dev/null; then
        xdg-open "$DASHBOARD_URL" &
    elif command -v open &> /dev/null; then
        open "$DASHBOARD_URL" &
    else
        print_info "Please open $DASHBOARD_URL in your browser"
    fi
    
    print_info "Dashboard features to demonstrate:"
    echo -e "  ${CYAN}• Real-time metrics updating every 30 seconds${NC}"
    echo -e "  ${CYAN}• Interactive charts and graphs${NC}"
    echo -e "  ${CYAN}• Multi-cloud service status indicators${NC}"
    echo -e "  ${CYAN}• Load simulation buttons${NC}"
    echo -e "  ${CYAN}• Alert testing functionality${NC}"
    echo -e "  ${CYAN}• Business metrics and KPIs${NC}"
    
    print_info "Try these interactions in the dashboard:"
    echo -e "  ${YELLOW}1. Click 'Refresh Data' to update metrics${NC}"
    echo -e "  ${YELLOW}2. Use 'Test Alert' to generate sample alerts${NC}"
    echo -e "  ${YELLOW}3. Toggle 'Auto-Refresh' to control updates${NC}"
    echo -e "  ${YELLOW}4. Observe real-time charts updating${NC}"
    
    wait_for_demo
}

# Demo 8: Monitoring & Observability
demo_monitoring() {
    print_demo_step "Monitoring & Observability"
    
    print_info "Prometheus metrics endpoint (if available)..."
    if curl -s -f "$COLLECTOR_URL/metrics" > /dev/null; then
        print_command "curl $COLLECTOR_URL/metrics | head -20"
        curl -s "$COLLECTOR_URL/metrics" | head -20
        print_success "Prometheus metrics available"
    else
        print_info "Prometheus metrics not available (using mock data)"
    fi
    
    print_info "System status overview..."
    demo_api_call "GET" "$DASHBOARD_URL/api/v1/dashboard" "" "Complete Dashboard Data"
    
    wait_for_demo
}

# Demo 9: Cost Optimization
demo_cost_optimization() {
    print_demo_step "Cost Optimization & Reporting"
    
    print_info "Demonstrating cost tracking across providers..."
    
    # Show AWS costs
    echo -e "${CYAN}AWS Cost Breakdown:${NC}"
    curl -s "$DASHBOARD_URL/api/v1/metrics/aws" | jq '.costs' 2>/dev/null || echo "AWS costs not available"
    
    echo ""
    # Show GCP costs
    echo -e "${CYAN}GCP Cost Breakdown:${NC}"
    curl -s "$DASHBOARD_URL/api/v1/metrics/gcp" | jq '.costs' 2>/dev/null || echo "GCP costs not available"
    
    print_info "Cost optimization features:"
    echo -e "  ${CYAN}• Real-time cost tracking per service${NC}"
    echo -e "  ${CYAN}• Budget alerts and notifications${NC}"
    echo -e "  ${CYAN}• Auto-scaling based on cost thresholds${NC}"
    echo -e "  ${CYAN}• Resource utilization optimization${NC}"
    echo -e "  ${CYAN}• Cross-provider cost comparison${NC}"
    
    wait_for_demo
}

# Main demo execution
main() {
    echo -e "${BLUE}🎭 Starting Automated Multi-Cloud Demo${NC}"
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
    echo ""
    
    echo -e "${YELLOW}This demo will showcase:${NC}"
    echo -e "  🩺 System health monitoring"
    echo -e "  📊 Real-time metrics collection"
    echo -e "  ☁️  AWS Lambda products API"
    echo -e "  🌐 Google Cloud Functions orders API"
    echo -e "  🔄 Load simulation and auto-scaling"
    echo -e "  🚨 Error simulation and alerting"
    echo -e "  🎛️  Interactive dashboard"
    echo -e "  📈 Monitoring and observability"
    echo -e "  💰 Cost optimization"
    echo ""
    
    print_info "Make sure your system is running with ./start-monitoring.sh"
    echo ""
    wait_for_demo
    
    # Execute all demo steps
    check_prerequisites
    demo_health_check
    demo_metrics
    demo_aws_products
    demo_gcp_orders
    demo_load_simulation
    demo_error_simulation
    demo_dashboard
    demo_monitoring
    demo_cost_optimization
    
    # Demo conclusion
    print_demo_step "Demo Complete!"
    
    echo -e "${GREEN}🎉 Multi-Cloud Demo Completed Successfully!${NC}"
    echo ""
    echo -e "${CYAN}Key Points Demonstrated:${NC}"
    echo -e "  ✅ Multi-cloud architecture working seamlessly"
    echo -e "  ✅ Real-time monitoring and alerting"
    echo -e "  ✅ Automated scaling and error recovery"
    echo -e "  ✅ Cost optimization across providers"
    echo -e "  ✅ Complete observability and metrics"
    echo ""
    echo -e "${YELLOW}🔗 Useful Links:${NC}"
    echo -e "  Dashboard: $DASHBOARD_URL"
    echo -e "  Health Check: $DASHBOARD_URL/api/v1/health"
    echo -e "  Metrics: $COLLECTOR_URL/api/health"
    echo ""
    echo -e "${PURPLE}Thank you for watching the demo! 🚀${NC}"
}

# Handle script interruption
trap 'echo -e "\n${YELLOW}Demo interrupted. System is still running.${NC}"; exit 0' INT

# Execute main function
main "$@"