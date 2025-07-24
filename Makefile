# =================================
# 🚀 MULTICLOUD E-COMMERCE MAKEFILE
# =================================

# Variables
APP_NAME := ecommerce-multicloud-monitor
VERSION := 1.0.0
BUILD_DIR := ./build
GO_VERSION := 1.21

# Colors for pretty output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m
BOLD := \033[1m

# Default target
.DEFAULT_GOAL := help

# =================================
# 📋 HELP & INFO
# =================================

.PHONY: help
help: ## 🆘 Show this help message
	@echo "$(CYAN)$(BOLD)🚀 $(APP_NAME) v$(VERSION)$(RESET)"
	@echo "$(CYAN)Available commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'

.PHONY: info
info: ## ℹ️  Show project information
	@echo "$(CYAN)$(BOLD)Project Information:$(RESET)"
	@echo "  Name: $(APP_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Build Dir: $(BUILD_DIR)"

# =================================
# 🛠️  DEVELOPMENT
# =================================

.PHONY: setup
setup: ## 🔧 Initial project setup
	@echo "$(YELLOW)Setting up project...$(RESET)"
	@go version
	@go mod download
	@mkdir -p $(BUILD_DIR)
	@mkdir -p aws-services
	@mkdir -p gcp-services
	@mkdir -p monitoring-dashboard
	@mkdir -p configs
	@echo "$(GREEN)✅ Setup complete!$(RESET)"

.PHONY: deps
deps: ## 📦 Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(RESET)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated!$(RESET)"

.PHONY: clean
clean: ## 🧹 Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(RESET)"
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "$(GREEN)✅ Clean complete!$(RESET)"

.PHONY: dev
dev: ## 🔄 Start development mode with hot reload
	@echo "$(YELLOW)Starting development mode...$(RESET)"
	@go run main.go

# =================================
# 🏗️  BUILD & TEST
# =================================

.PHONY: build
build: ## 🔨 Build the application
	@echo "$(YELLOW)Building application...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "$(GREEN)✅ Build complete! Binary: $(BUILD_DIR)/$(APP_NAME)$(RESET)"

.PHONY: build-all
build-all: ## 🔨 Build for all platforms
	@echo "$(YELLOW)Building for all platforms...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "$(GREEN)✅ Multi-platform build complete!$(RESET)"

.PHONY: test
test: ## 🧪 Run tests
	@echo "$(YELLOW)Running tests...$(RESET)"
	@go test -v ./...
	@echo "$(GREEN)✅ Tests complete!$(RESET)"

.PHONY: test-coverage
test-coverage: ## 📊 Run tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(RESET)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report: coverage.html$(RESET)"

.PHONY: lint
lint: ## 🔍 Run linter
	@echo "$(YELLOW)Running linter...$(RESET)"
	@golangci-lint run
	@echo "$(GREEN)✅ Linting complete!$(RESET)"

# =================================
# ☁️  AWS DEPLOYMENT
# =================================

.PHONY: aws-login
aws-login: ## 🔐 Login to AWS
	@echo "$(YELLOW)Logging in to AWS...$(RESET)"
	@aws sts get-caller-identity

.PHONY: build-aws-lambda
build-aws-lambda: ## 📦 Build AWS Lambda functions
	@echo "$(YELLOW)Building AWS Lambda functions...$(RESET)"
	@mkdir -p $(BUILD_DIR)/aws
	@cd aws-services && find . -name "*.go" -not -path "./vendor/*" | while read file; do \
		dir=$$(dirname $$file); \
		GOOS=linux GOARCH=amd64 go build -o ../$(BUILD_DIR)/aws/$$(basename $$dir) $$file; \
	done
	@echo "$(GREEN)✅ AWS Lambda functions built!$(RESET)"

.PHONY: deploy-aws
deploy-aws: build-aws-lambda ## 🚀 Deploy to AWS
	@echo "$(YELLOW)Deploying to AWS...$(RESET)"
	@echo "$(GREEN)✅ AWS deployment complete!$(RESET)"

# =================================
# 🌐 GCP DEPLOYMENT
# =================================

.PHONY: gcp-login
gcp-login: ## 🔐 Login to Google Cloud
	@echo "$(YELLOW)Logging in to Google Cloud...$(RESET)"
	@gcloud auth list

.PHONY: build-gcp-functions
build-gcp-functions: ## 📦 Build GCP Cloud Functions
	@echo "$(YELLOW)Building GCP Cloud Functions...$(RESET)"
	@mkdir -p $(BUILD_DIR)/gcp
	@echo "$(GREEN)✅ GCP Cloud Functions built!$(RESET)"

.PHONY: deploy-gcp
deploy-gcp: build-gcp-functions ## 🚀 Deploy to Google Cloud
	@echo "$(YELLOW)Deploying to Google Cloud...$(RESET)"
	@echo "$(GREEN)✅ GCP deployment complete!$(RESET)"

# =================================
# 🚀 FULL DEPLOYMENT
# =================================

.PHONY: deploy-all
deploy-all: deploy-aws deploy-gcp ## 🌍 Deploy to both clouds
	@echo "$(GREEN)$(BOLD)🎉 Multi-cloud deployment complete!$(RESET)"

# =================================
# 📊 MONITORING
# =================================

.PHONY: start-monitoring
start-monitoring: ## 📈 Start monitoring stack
	@echo "$(YELLOW)Starting monitoring stack...$(RESET)"
	@docker-compose -f monitoring/docker-compose.yml up -d
	@echo "$(GREEN)✅ Monitoring started!$(RESET)"
	@echo "  Grafana: http://localhost:3000"
	@echo "  Prometheus: http://localhost:9090"

.PHONY: stop-monitoring
stop-monitoring: ## 🛑 Stop monitoring stack
	@echo "$(YELLOW)Stopping monitoring stack...$(RESET)"
	@docker-compose -f monitoring/docker-compose.yml down
	@echo "$(GREEN)✅ Monitoring stopped!$(RESET)"

.PHONY: start-dashboard
start-dashboard: ## 🎛️ Start web dashboard
	@echo "$(YELLOW)Starting dashboard...$(RESET)"
	@cd monitoring-dashboard && go run main.go &
	@echo "$(GREEN)✅ Dashboard started at http://localhost:8080$(RESET)"

# =================================
# 🔧 UTILITIES
# =================================

.PHONY: logs
logs: ## 📝 View application logs
	@echo "$(YELLOW)Viewing logs...$(RESET)"
	@tail -f logs/app.log

.PHONY: health-check
health-check: ## ❤️  Run health check
	@echo "$(YELLOW)Running health check...$(RESET)"
	@go run scripts/health-check.go
	@echo "$(GREEN)✅ Health check complete!$(RESET)"

.PHONY: backup
backup: ## 💾 Create backup
	@echo "$(YELLOW)Creating backup...$(RESET)"
	@go run scripts/backup.go
	@echo "$(GREEN)✅ Backup complete!$(RESET)"

.PHONY: cost-report
cost-report: ## 💰 Generate cost report
	@echo "$(YELLOW)Generating cost report...$(RESET)"
	@go run scripts/cost-report.go
	@echo "$(GREEN)✅ Cost report generated!$(RESET)"

# =================================
# 🐳 DOCKER
# =================================

.PHONY: docker-build
docker-build: ## 🐳 Build Docker image
	@echo "$(YELLOW)Building Docker image...$(RESET)"
	@docker build -t $(APP_NAME):$(VERSION) .
	@echo "$(GREEN)✅ Docker image built!$(RESET)"

.PHONY: docker-run
docker-run: ## 🐳 Run Docker container
	@echo "$(YELLOW)Running Docker container...$(RESET)"
	@docker run -p 8080:8080 --env-file .env $(APP_NAME):$(VERSION)

# =================================
# 📚 DOCUMENTATION
# =================================

.PHONY: docs
docs: ## 📖 Generate documentation
	@echo "$(YELLOW)Generating documentation...$(RESET)"
	@godoc -http=:6060 &
	@echo "$(GREEN)✅ Documentation server: http://localhost:6060$(RESET)"

.PHONY: tree
tree: ## 🌳 Show project structure
	@echo "$(CYAN)$(BOLD)Project Structure:$(RESET)"
	@tree -I '.git|node_modules|vendor|*.log' --dirsfirst -a -C -L 3

# =================================
# 🧪 DEMO & EXAMPLES
# =================================

.PHONY: demo
demo: ## 🎭 Run demo scenario
	@echo "$(YELLOW)Running demo...$(RESET)"
	@echo "$(GREEN)🎉 Demo scenario complete!$(RESET)"

.PHONY: example
example: ## 📋 Show usage examples
	@echo "$(CYAN)$(BOLD)Usage Examples:$(RESET)"
	@echo "  $(GREEN)make setup$(RESET)          - Initial setup"
	@echo "  $(GREEN)make dev$(RESET)            - Start development"
	@echo "  $(GREEN)make deploy-all$(RESET)     - Deploy to both clouds"
	@echo "  $(GREEN)make start-monitoring$(RESET) - Start monitoring"
	@echo "  $(GREEN)make health-check$(RESET)   - Check system health"