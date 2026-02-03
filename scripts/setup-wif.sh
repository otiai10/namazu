#!/bin/bash
#
# Setup Workload Identity Federation for GitHub Actions
#
# Usage:
#   ./scripts/setup-wif.sh
#   ./scripts/setup-wif.sh --project=my-project --repo=owner/repo
#
set -e

# Default values (can be overridden via arguments)
PROJECT_ID="${PROJECT_ID:-namazu-live}"
GITHUB_REPO="${GITHUB_REPO:-otiai10/namazu}"
POOL_NAME="${POOL_NAME:-github-pool}"
PROVIDER_NAME="${PROVIDER_NAME:-github-provider}"
SA_NAME="${SA_NAME:-github-actions}"

# Parse arguments
for arg in "$@"; do
  case $arg in
    --project=*)
      PROJECT_ID="${arg#*=}"
      ;;
    --repo=*)
      GITHUB_REPO="${arg#*=}"
      ;;
    --help)
      echo "Usage: $0 [--project=PROJECT_ID] [--repo=OWNER/REPO]"
      exit 0
      ;;
  esac
done

# Get project number
echo "=== Getting Project Number ==="
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
echo "Project: $PROJECT_ID (Number: $PROJECT_NUMBER)"
echo "GitHub Repo: $GITHUB_REPO"
echo ""

echo "=== Creating Workload Identity Pool ==="
gcloud iam workload-identity-pools create "$POOL_NAME" \
  --project="$PROJECT_ID" \
  --location="global" \
  --display-name="GitHub Actions Pool" 2>/dev/null || echo "Pool already exists, continuing..."

echo "=== Creating OIDC Provider ==="
# NOTE: --attribute-condition is REQUIRED for security
# It restricts which GitHub repositories can authenticate
gcloud iam workload-identity-pools providers create-oidc "$PROVIDER_NAME" \
  --project="$PROJECT_ID" \
  --location="global" \
  --workload-identity-pool="$POOL_NAME" \
  --display-name="GitHub Provider" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
  --attribute-condition="assertion.repository=='${GITHUB_REPO}'" \
  --issuer-uri="https://token.actions.githubusercontent.com" 2>/dev/null || echo "Provider already exists, continuing..."

echo "=== Creating Service Account ==="
gcloud iam service-accounts create "$SA_NAME" \
  --project="$PROJECT_ID" \
  --display-name="GitHub Actions" 2>/dev/null || echo "Service account already exists, continuing..."

echo "=== Granting Workload Identity User role ==="
gcloud iam service-accounts add-iam-policy-binding \
  "${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --project="$PROJECT_ID" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_NAME}/attribute.repository/${GITHUB_REPO}"

echo "=== Granting Artifact Registry Writer role ==="
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer"

echo "=== Granting Compute Instance Admin role ==="
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/compute.instanceAdmin.v1"

echo ""
echo "=============================================="
echo "  Setup Complete!"
echo "=============================================="
echo ""
echo "Add these to your GitHub Actions workflow:"
echo ""
echo "env:"
echo "  GCP_PROJECT_ID: $PROJECT_ID"
echo "  GCP_WIF_PROVIDER: projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_NAME}/providers/${PROVIDER_NAME}"
echo "  GCP_SERVICE_ACCOUNT: ${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
echo ""
echo "permissions:"
echo "  contents: read"
echo "  id-token: write"
echo ""
echo "steps:"
echo "  - uses: google-github-actions/auth@v2"
echo "    with:"
echo "      workload_identity_provider: \${{ env.GCP_WIF_PROVIDER }}"
echo "      service_account: \${{ env.GCP_SERVICE_ACCOUNT }}"
