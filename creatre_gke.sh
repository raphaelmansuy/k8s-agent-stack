#!/bin/bash
set -e # Exit immediately if a command fails

# --- CONFIGURATION ---
# Replace with your actual Project ID
PROJECT_ID="your-project-id"
CLUSTER_NAME="minimal-autopilot-cluster"
# Autopilot MUST be Regional. us-central1 is generally the cheapest/most reliable.
REGION="us-central1"

# --- PRE-FLIGHT CHECKS ---
echo "Starting GKE Autopilot provisioning..."

# Ensure gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "Error: gcloud CLI is not installed."
    exit 1
fi

# set the project locally for this session
gcloud config set project "$PROJECT_ID" --quiet

# --- STEP 1: ENABLE APIS ---
echo "Enabling Kubernetes Engine API (this may take a minute)..."
gcloud services enable container.googleapis.com --project="$PROJECT_ID"

# --- STEP 2: IAM SETUP ---
echo "Configuring IAM permissions..."
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
COMPUTE_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

# Grant the minimal node service account role to the default Compute SA
# This ensures nodes can pull images and write logs without needing full 'Editor' access
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:${COMPUTE_SA}" \
    --role="roles/container.nodeServiceAccount" \
    --condition=None --quiet > /dev/null

echo "IAM roles verified."

# --- STEP 3: CREATE CLUSTER ---
# Check if cluster already exists to avoid error
if gcloud container clusters describe "$CLUSTER_NAME" --region "$REGION" --project "$PROJECT_ID" &> /dev/null; then
    echo "Cluster '$CLUSTER_NAME' already exists. Skipping creation."
else
    echo "Creating Autopilot Cluster '$CLUSTER_NAME' in '$REGION'..."
    echo "NOTE: This can take 5-10 minutes."
    
    # create-auto implies regional. 
    gcloud container clusters create-auto "$CLUSTER_NAME" \
        --region="$REGION" \
        --project="$PROJECT_ID" \
        --release-channel="regular"
fi

# --- STEP 4: CONFIGURE KUBECTL ---
echo "Fetching cluster credentials..."
gcloud container clusters get-credentials "$CLUSTER_NAME" \
    --region="$REGION" \
    --project="$PROJECT_ID"

# --- SUMMARY ---
echo "--------------------------------------------------------"
echo "✅ SUCCESS! Cluster is ready."
echo "--------------------------------------------------------"
echo "Current Cost Status:"
echo "1. Management Fee: Covered by Free Tier (if this is your only zonal/regional cluster)."
echo "2. Compute Cost:   $0/hr (until you deploy a Pod)."
echo ""
echo "Verify empty state:"
echo "   kubectl get pods --all-namespaces"
echo ""
echo "To DELETE this cluster and stop all potential billing:"
echo "   gcloud container clusters delete $CLUSTER_NAME --region=$REGION --project=$PROJECT_ID --quiet"
echo "--------------------------------------------------------"