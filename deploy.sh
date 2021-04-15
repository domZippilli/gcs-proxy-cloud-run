#!/usr/bin/env bash
set -euf -o pipefail

function usage(){
    echo >&2
    echo "Usage: $0 BUCKET_NAME REGION project image_name service_name" >&2
    echo "Deploys this service in Cloud Run." >&2
    echo >&2
    echo "BUCKET_NAME is required. This is the bucket to proxy." >&2
    echo >&2
    echo "project is optional; your gcloud config project will be used if it" >&2
    echo "is not provided." >&2
    echo >&2
    echo "image_name is optional. default value is gcr.io/PROJECT/gcs-streaming-proxy." >&2
    echo >&2
    echo "service_name is optional. default value is gcs-BUCKET_NAME." >&2
    echo >&2
}

BUCKET_NAME=${1?$(usage)}
REGION=${2?$(usage)}
PROJECT=${3-}
PROJECT="${3:-$(gcloud config get-value project)}"
IMAGE_NAME=${4-}
IMAGE_NAME="${4:-gcr.io/${PROJECT}/gcs-streaming-proxy}"
SERVICE_NAME=${5-}
SERVICE_NAME="${5:-gcs-${BUCKET_NAME}}"

# quick and dirty way to catch if the user asks for help, like --help
# downside: you can't tag the image as *help or just "-h"
if [[ "${BUCKET_NAME}" == *help ]] || [[ "${BUCKET_NAME}" == "-h" ]]; then
    usage
    exit
fi

gcloud run deploy "${SERVICE_NAME}" \
    --project "${PROJECT}" \
    --region "${REGION}" \
    --image "${IMAGE_NAME}" \
    --set-env-vars BUCKET_NAME="${BUCKET_NAME}" \
    --cpu=2 \
    --memory=512Mi \
    --concurrency=100 \
    --max-instances=100 \
    --timeout=300s \
    --platform managed \
    --allow-unauthenticated \
    --ingress=all

echo Service deployed.