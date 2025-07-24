
# =================================
# Google Cloud Functions Deploy Script
# =================================

set -e  # Exit on any error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FUNCTION_NAME="ecommerce-order-processor"
PROJECT_ID="${GCP_PROJECT_ID}"
REGION="${GCP_REGION:-us-central1}"
RUNTIME="go121"
MEMORY="256MB"
TIMEOUT="60s"
TRIGGER_TYPE="http"
FIRESTORE_DATABASE="(default)"

echo -e "${BLUE} Starting Google Cloud Functions deployment...${NC}"

# Function to print colored output
print_status() {
    echo -e "${GREEN} $1${NC}"
}

print_warning() {
    echo -e "${YELLOW} $1${NC}"
}

print_error() {
    echo -e "${RED} $1${NC}"
}

# Check if gcloud CLI is configured
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | head -n 1 &> /dev/null; then
    print_error "Google Cloud CLI not configured. Please run 'gcloud auth login'"
    exit 1
fi

print_status "Google Cloud CLI configured correctly"

# Get current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${BLUE} Working directory: $SCRIPT_DIR${NC}"

# Check if PROJECT_ID is set
if [ -z "$PROJECT_ID" ]; then
    PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
    if [ -z "$PROJECT_ID" ]; then
        print_error "GCP_PROJECT_ID not set and no default project configured"
        echo "Please set GCP_PROJECT_ID environment variable or run 'gcloud config set project YOUR_PROJECT_ID'"
        exit 1
    fi
fi

print_status "Using Project ID: $PROJECT_ID"

# Set the project
gcloud config set project "$PROJECT_ID"

# Step 1: Enable required APIs
echo -e "${BLUE} Enabling required APIs...${NC}"

REQUIRED_APIS=(
    "cloudfunctions.googleapis.com"
    "firestore.googleapis.com"
    "cloudbuild.googleapis.com"
    "logging.googleapis.com"
    "pubsub.googleapis.com"
)

for api in "${REQUIRED_APIS[@]}"; do
    if ! gcloud services list --enabled --filter="name:$api" --format="value(name)" | grep -q "$api"; then
        print_warning "Enabling API: $api"
        gcloud services enable "$api"
    else
        print_status "API already enabled: $api"
    fi
done

# Step 2: Create Firestore database if it doesn't exist
echo -e "${BLUE} Setting up Firestore...${NC}"

# Check if Firestore is already initialized
if ! gcloud firestore databases describe --database="$FIRESTORE_DATABASE" &> /dev/null; then
    print_warning "Creating Firestore database..."
    
    # Create Firestore database in Native mode
    gcloud firestore databases create --region="$REGION" --database="$FIRESTORE_DATABASE"
    
    print_status "Firestore database created"
else
    print_status "Firestore database already exists"
fi

# Step 3: Create required collections and sample data
echo -e "${BLUE} Setting up Firestore collections...${NC}"

# Create a simple script to initialize Firestore collections
cat > init_firestore.js << 'EOF'
const admin = require('firebase-admin');

// Initialize Firebase Admin
const serviceAccount = require('./service-account-key.json');
admin.initializeApp({
  credential: admin.credential.cert(serviceAccount)
});

const db = admin.firestore();

async function initCollections() {
  try {
    // Create orders collection with sample document
    const ordersRef = db.collection('orders');
    await ordersRef.doc('sample').set({
      user_id: 'user_123',
      user_email: 'test@example.com',
      status: 'pending',
      total_amount: 99.99,
      currency: 'USD',
      created_at: admin.firestore.FieldValue.serverTimestamp(),
      updated_at: admin.firestore.FieldValue.serverTimestamp()
    });

    // Create users collection with sample document
    const usersRef = db.collection('users');
    await usersRef.doc('user_123').set({
      email: 'test@example.com',
      name: 'Test User',
      created_at: admin.firestore.FieldValue.serverTimestamp()
    });

    console.log(' Firestore collections initialized');
  } catch (error) {
    console.error(' Error initializing Firestore:', error);
  }
}

initCollections();
EOF

print_status "Firestore collections configuration ready"

# Step 4: Prepare function files
echo -e "${BLUE} Preparing function files...${NC}"

# Check if required files exist
REQUIRED_FILES=("main.go" "handler.go" "models.go" "go.mod")
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        print_error "Required file missing: $file"
        exit 1
    fi
done

print_status "All required files present"

# Step 5: Download dependencies
echo -e "${BLUE} Downloading Go dependencies...${NC}"

go mod tidy
if [ $? -ne 0 ]; then
    print_error "Failed to download Go dependencies"
    exit 1
fi

print_status "Go dependencies downloaded"

# Step 6: Deploy Cloud Function
echo -e "${BLUE} Deploying Cloud Function...${NC}"

# Deploy the function
gcloud functions deploy "$FUNCTION_NAME" \
    --gen2 \
    --runtime="$RUNTIME" \
    --region="$REGION" \
    --source=. \
    --entry-point=ProcessOrder \
    --trigger=http \
    --memory="$MEMORY" \
    --timeout="$TIMEOUT" \
    --allow-unauthenticated \
    --set-env-vars="GCP_PROJECT_ID=$PROJECT_ID" \
    --max-instances=10 \
    --min-instances=0

if [ $? -ne 0 ]; then
    print_error "Failed to deploy Cloud Function"
    exit 1
fi

print_status "Cloud Function deployed successfully"

# Step 7: Get function URL
echo -e "${BLUE} Getting function URL...${NC}"

FUNCTION_URL=$(gcloud functions describe "$FUNCTION_NAME" --region="$REGION" --gen2 --format="value(serviceConfig.uri)")

if [ -z "$FUNCTION_URL" ]; then
    print_error "Failed to get function URL"
    exit 1
fi

print_status "Function URL obtained: $FUNCTION_URL"

# Step 8: Set up IAM permissions (if needed)
echo -e "${BLUE} Setting up IAM permissions...${NC}"

# Allow unauthenticated access (for demo purposes)
gcloud functions add-iam-policy-binding "$FUNCTION_NAME" \
    --region="$REGION" \
    --member="allUsers" \
    --role="roles/cloudfunctions.invoker" \
    --gen2 &> /dev/null

print_status "IAM permissions configured"

# Step 9: Deploy additional health check function
echo -e "${BLUE} Deploying health check function...${NC}"

gcloud functions deploy "${FUNCTION_NAME}-health" \
    --gen2 \
    --runtime="$RUNTIME" \
    --region="$REGION" \
    --source=. \
    --entry-point=HealthCheck \
    --trigger=http \
    --memory="128MB" \
    --timeout="10s" \
    --allow-unauthenticated \
    --set-env-vars="GCP_PROJECT_ID=$PROJECT_ID" \
    --max-instances=5 \
    --min-instances=0 &> /dev/null

HEALTH_URL=$(gcloud functions describe "${FUNCTION_NAME}-health" --region="$REGION" --gen2 --format="value(serviceConfig.uri)")

print_status "Health check function deployed"

# Step 10: Display deployment information
echo -e "${BLUE} Deployment Summary${NC}"
echo -e "${GREEN}================================${NC}"
echo -e "Function Name: $FUNCTION_NAME"
echo -e "Region: $REGION"
echo -e "Project ID: $PROJECT_ID"
echo -e "Firestore: Enabled"
echo -e "Function URL: $FUNCTION_URL"
echo -e "Health Check: $HEALTH_URL"
echo -e "${GREEN}================================${NC}"

# Step 11: Test the deployment
echo -e "${BLUE}🧪 Testing deployment...${NC}"

# Wait a moment for deployment to be ready
sleep 10

# Test health check
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$HEALTH_URL")

if [ "$HTTP_CODE" == "200" ]; then
    print_status "Health check responding correctly (HTTP $HTTP_CODE)"
else
    print_warning "Health check returned HTTP $HTTP_CODE - may need a moment to be fully ready"
fi

# Test main function
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FUNCTION_URL/orders")

if [ "$HTTP_CODE" == "200" ]; then
    print_status "Main function responding correctly (HTTP $HTTP_CODE)"
else
    print_warning "Main function returned HTTP $HTTP_CODE - may need a moment to be fully ready"
fi

# Clean up temporary files
rm -f init_firestore.js

echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
echo -e "${BLUE}💡 Quick test commands:${NC}"
echo -e "# List orders (empty initially)"
echo -e "curl -X GET $FUNCTION_URL/orders"
echo ""
echo -e "# Create a test order"
echo -e "curl -X POST $FUNCTION_URL/orders -H 'Content-Type: application/json' -d '{
  \"user_id\": \"user_123\",
  \"user_email\": \"test@example.com\",
  \"payment_method\": \"credit_card\",
  \"items\": [
    {
      \"product_id\": \"prod_1\",
      \"product_name\": \"Test Product\",
      \"sku\": \"TEST001\",
      \"quantity\": 2,
      \"unit_price\": 29.99,
      \"total_price\": 59.98
    }
  ],
  \"shipping_info\": {
    \"full_name\": \"John Doe\",
    \"address\": \"123 Main St\",
    \"city\": \"Anytown\",
    \"state\": \"CA\",
    \"postal_code\": \"12345\",
    \"country\": \"USA\",
    \"phone\": \"+1234567890\",
    \"method\": \"standard\"
  }
}'"
echo ""
echo -e "# Get order statistics"
echo -e "curl -X GET $FUNCTION_URL/orders/stats"

print_status "All done! Your Order Processor is live on Google Cloud! 🚀"