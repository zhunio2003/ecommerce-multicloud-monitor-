#!/bin/bash

# =================================
# 🚀 MULTI-CLOUD MONITORING STARTUP
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
DASHBOARD_PORT=8080
COLLECTOR_PORT=8081
PROMETHEUS_PORT=9090

echo -e "${BLUE}🚀 Starting Multi-Cloud E-commerce Monitoring System${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"

print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to check if port is available
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        return 1
    else
        return 0
    fi
}

# Function to kill process on port
kill_port() {
    local port=$1
    local pid=$(lsof -ti:$port)
    if [ ! -z "$pid" ]; then
        print_warning "Killing existing process on port $port (PID: $pid)"
        kill -9 $pid 2>/dev/null || true
        sleep 2
    fi
}

# Step 1: Check system requirements
echo -e "${PURPLE}🔍 Checking system requirements...${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.21 or later."
    exit 1
fi
print_status "Go $(go version | awk '{print $3}') detected"

# Check if we're in the right directory
if [ ! -f "main.go" ] || [ ! -f "go.mod" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi
print_status "Project structure verified"

# Step 2: Set up environment
echo -e "${PURPLE}🛠️  Setting up environment...${NC}"

if [ ! -f ".env" ]; then
    if [ -f ".env.example" ]; then
        cp .env.example .env
        print_warning "Created .env from .env.example. Please configure your credentials."
    else
        print_error ".env file not found. Please create one with your configuration."
        exit 1
    fi
fi
print_status "Environment configuration loaded"

# Step 3: Install dependencies
echo -e "${PURPLE}📦 Installing dependencies...${NC}"

print_info "Installing main application dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    print_error "Failed to install main dependencies"
    exit 1
fi

# Install collector dependencies
if [ -d "monitoring-dashboard/data-collector" ]; then
    print_info "Installing collector dependencies..."
    cd monitoring-dashboard/data-collector
    go mod tidy
    if [ $? -ne 0 ]; then
        print_error "Failed to install collector dependencies"
        exit 1
    fi
    cd ../..
fi

print_status "All dependencies installed"

# Step 4: Check and free ports
echo -e "${PURPLE}🔌 Checking ports...${NC}"

PORTS_TO_CHECK=($DASHBOARD_PORT $COLLECTOR_PORT $PROMETHEUS_PORT)
for port in "${PORTS_TO_CHECK[@]}"; do
    if ! check_port $port; then
        print_warning "Port $port is in use"
        kill_port $port
    fi
done

# Wait a moment for ports to be freed
sleep 3

# Verify ports are now free
for port in "${PORTS_TO_CHECK[@]}"; do
    if check_port $port; then
        print_status "Port $port is available"
    else
        print_error "Port $port is still in use"
        exit 1
    fi
done

# Step 5: Create necessary directories
echo -e "${PURPLE}📁 Creating directories...${NC}"

DIRECTORIES=(
    "logs"
    "data"
    "monitoring-dashboard/web-dashboard"
    "monitoring-dashboard/data-collector"
)

for dir in "${DIRECTORIES[@]}"; do
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        print_info "Created directory: $dir"
    fi
done

print_status "Directory structure ready"

# Step 6: Start the metrics collector
echo -e "${PURPLE}📊 Starting metrics collector...${NC}"

if [ -f "monitoring-dashboard/data-collector/collector.go" ]; then
    cd monitoring-dashboard/data-collector
    
    print_info "Building collector..."
    go build -o collector collector.go
    
    if [ $? -eq 0 ]; then
        print_info "Starting collector on port $COLLECTOR_PORT..."
        nohup ./collector > ../../logs/collector.log 2>&1 &
        COLLECTOR_PID=$!
        echo $COLLECTOR_PID > ../../data/collector.pid
        
        # Wait a moment and check if it's running
        sleep 3
        if kill -0 $COLLECTOR_PID 2>/dev/null; then
            print_status "Metrics collector started (PID: $COLLECTOR_PID)"
        else
            print_error "Failed to start metrics collector"
            cat ../../logs/collector.log
            exit 1
        fi
    else
        print_error "Failed to build collector"
        exit 1
    fi
    
    cd ../..
else
    print_warning "Collector not found, running without real-time metrics"
fi

# Step 7: Start the main dashboard
echo -e "${PURPLE}🎛️  Starting main dashboard...${NC}"

print_info "Building main application..."
go build -o ecommerce-monitor main.go

if [ $? -eq 0 ]; then
    print_info "Starting dashboard on port $DASHBOARD_PORT..."
    nohup ./ecommerce-monitor > logs/dashboard.log 2>&1 &
    DASHBOARD_PID=$!
    echo $DASHBOARD_PID > data/dashboard.pid
    
    # Wait a moment and check if it's running
    sleep 3
    if kill -0 $DASHBOARD_PID 2>/dev/null; then
        print_status "Dashboard started (PID: $DASHBOARD_PID)"
    else
        print_error "Failed to start dashboard"
        cat logs/dashboard.log
        exit 1
    fi
else
    print_error "Failed to build main application"
    exit 1
fi

# Step 8: Health checks
echo -e "${PURPLE}🩺 Running health checks...${NC}"

print_info "Waiting for services to be ready..."
sleep 5

# Check dashboard
DASHBOARD_URL="http://localhost:$DASHBOARD_PORT/api/v1/health"
if curl -s -f "$DASHBOARD_URL" > /dev/null; then
    print_status "Dashboard is responding"
else
    print_warning "Dashboard health check failed"
fi

# Check collector (if running)
if [ -f "data/collector.pid" ]; then
    COLLECTOR_URL="http://localhost:$COLLECTOR_PORT/api/health"
    if curl -s -f "$COLLECTOR_URL" > /dev/null; then
        print_status "Metrics collector is responding"
    else
        print_warning "Collector health check failed"
    fi
fi

# Step 9: Display startup summary
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}🎉 Multi-Cloud Monitoring System Started Successfully!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${CYAN}📍 Access Points:${NC}"
echo -e "   🌐 Main Dashboard:    http://localhost:$DASHBOARD_PORT"
echo -e "   🩺 Health Check:      http://localhost:$DASHBOARD_PORT/api/v1/health"
echo -e "   📊 Metrics API:       http://localhost:$DASHBOARD_PORT/api/v1/metrics/snapshot"
if [ -f "data/collector.pid" ]; then
echo -e "   📈 Collector API:     http://localhost:$COLLECTOR_PORT/api/health"
echo -e "   🔬 Prometheus:        http://localhost:$COLLECTOR_PORT/metrics"
fi
echo ""
echo -e "${CYAN}📋 Process Information:${NC}"
if [ -f "data/dashboard.pid" ]; then
    DASHBOARD_PID=$(cat data/dashboard.pid)
    echo -e "   🎛️  Dashboard PID:     $DASHBOARD_PID"
fi
if [ -f "data/collector.pid" ]; then
    COLLECTOR_PID=$(cat data/collector.pid)
    echo -e "   📊 Collector PID:     $COLLECTOR_PID"
fi
echo ""
echo -e "${CYAN}📁 Log Files:${NC}"
echo -e "   📄 Dashboard:         logs/dashboard.log"
echo -e "   📄 Collector:         logs/collector.log"
echo ""
echo -e "${CYAN}🛠️  Management Commands:${NC}"
echo -e "   🔄 Restart:           ./start-monitoring.sh"
echo -e "   🛑 Stop:              ./stop-monitoring.sh"
echo -e "   📋 Status:            ./status-monitoring.sh"
echo -e "   📊 View Logs:         tail -f logs/dashboard.log"
echo ""
echo -e "${YELLOW}💡 Quick Start Guide:${NC}"
echo -e "   1. Open http://localhost:$DASHBOARD_PORT in your browser"
echo -e "   2. Click 'Refresh Data' to load real-time metrics"
echo -e "   3. Use 'Test Alert' to simulate monitoring alerts"
echo -e "   4. Check the API endpoints for integration"
echo ""
echo -e "${GREEN}✨ Enjoy monitoring your multi-cloud infrastructure!${NC}"
echo ""

# Step 10: Create management scripts
echo -e "${PURPLE}📝 Creating management scripts...${NC}"

# Create stop script
cat > stop-monitoring.sh << 'EOF'
#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🛑 Stopping Multi-Cloud Monitoring System...${NC}"

# Stop dashboard
if [ -f "data/dashboard.pid" ]; then
    DASHBOARD_PID=$(cat data/dashboard.pid)
    if kill -0 $DASHBOARD_PID 2>/dev/null; then
        kill $DASHBOARD_PID
        echo -e "${GREEN}✅ Dashboard stopped (PID: $DASHBOARD_PID)${NC}"
    else
        echo -e "${YELLOW}⚠️  Dashboard was not running${NC}"
    fi
    rm -f data/dashboard.pid
fi

# Stop collector
if [ -f "data/collector.pid" ]; then
    COLLECTOR_PID=$(cat data/collector.pid)
    if kill -0 $COLLECTOR_PID 2>/dev/null; then
        kill $COLLECTOR_PID
        echo -e "${GREEN}✅ Collector stopped (PID: $COLLECTOR_PID)${NC}"
    else
        echo -e "${YELLOW}⚠️  Collector was not running${NC}"
    fi
    rm -f data/collector.pid
fi

# Clean up any remaining processes
pkill -f "ecommerce-monitor" 2>/dev/null || true
pkill -f "collector" 2>/dev/null || true

echo -e "${GREEN}🎉 All services stopped successfully!${NC}"
EOF

# Create status script
cat > status-monitoring.sh << 'EOF'
#!/bin/bash

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}📊 Multi-Cloud Monitoring System Status${NC}"
echo -e "${CYAN}════════════════════════════════════════${NC}"

# Check dashboard
if [ -f "data/dashboard.pid" ]; then
    DASHBOARD_PID=$(cat data/dashboard.pid)
    if kill -0 $DASHBOARD_PID 2>/dev/null; then
        echo -e "${GREEN}✅ Dashboard: Running (PID: $DASHBOARD_PID)${NC}"
        
        # Check if responding
        if curl -s -f "http://localhost:8080/api/v1/health" > /dev/null; then
            echo -e "   🌐 HTTP: Responding on port 8080"
        else
            echo -e "   ${YELLOW}⚠️  HTTP: Not responding${NC}"
        fi
    else
        echo -e "${RED}❌ Dashboard: Not running${NC}"
        rm -f data/dashboard.pid
    fi
else
    echo -e "${RED}❌ Dashboard: Not started${NC}"
fi

# Check collector
if [ -f "data/collector.pid" ]; then
    COLLECTOR_PID=$(cat data/collector.pid)
    if kill -0 $COLLECTOR_PID 2>/dev/null; then
        echo -e "${GREEN}✅ Collector: Running (PID: $COLLECTOR_PID)${NC}"
        
        # Check if responding
        if curl -s -f "http://localhost:8081/api/health" > /dev/null; then
            echo -e "   📊 HTTP: Responding on port 8081"
        else
            echo -e "   ${YELLOW}⚠️  HTTP: Not responding${NC}"
        fi
    else
        echo -e "${RED}❌ Collector: Not running${NC}"
        rm -f data/collector.pid
    fi
else
    echo -e "${YELLOW}⚠️  Collector: Not configured${NC}"
fi

# Check log files
echo ""
echo -e "${CYAN}📄 Log Files:${NC}"
if [ -f "logs/dashboard.log" ]; then
    LOG_SIZE=$(du -h logs/dashboard.log | cut -f1)
    LOG_LINES=$(wc -l < logs/dashboard.log)
    echo -e "   📄 Dashboard: $LOG_SIZE ($LOG_LINES lines)"
else
    echo -e "   ${YELLOW}⚠️  Dashboard log not found${NC}"
fi

if [ -f "logs/collector.log" ]; then
    LOG_SIZE=$(du -h logs/collector.log | cut -f1)
    LOG_LINES=$(wc -l < logs/collector.log)
    echo -e "   📄 Collector: $LOG_SIZE ($LOG_LINES lines)"
else
    echo -e "   ${YELLOW}⚠️  Collector log not found${NC}"
fi

# Port usage
echo ""
echo -e "${CYAN}🔌 Port Usage:${NC}"
PORTS=(8080 8081 9090)
for port in "${PORTS[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        PID=$(lsof -ti:$port)
        PROCESS=$(ps -p $PID -o comm= 2>/dev/null || echo "unknown")
        echo -e "   🔌 Port $port: In use by $PROCESS (PID: $PID)"
    else
        echo -e "   ⚪ Port $port: Available"
    fi
done

echo ""
EOF

# Create restart script
cat > restart-monitoring.sh << 'EOF'
#!/bin/bash

echo "🔄 Restarting Multi-Cloud Monitoring System..."
./stop-monitoring.sh
sleep 3
./start-monitoring.sh
EOF

# Make scripts executable
chmod +x stop-monitoring.sh status-monitoring.sh restart-monitoring.sh

print_status "Management scripts created"

# Step 11: Open browser (optional)
if command -v xdg-open &> /dev/null; then
    print_info "Opening dashboard in browser..."
    sleep 2
    xdg-open "http://localhost:$DASHBOARD_PORT" &
elif command -v open &> /dev/null; then
    print_info "Opening dashboard in browser..."
    sleep 2
    open "http://localhost:$DASHBOARD_PORT" &
fi

print_status "Startup complete! System is ready for monitoring."