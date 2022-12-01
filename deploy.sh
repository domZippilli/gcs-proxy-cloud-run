#!/usr/bin/env bash
# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -euf -o pipefail

function usage(){
    echo >&2
    echo "Usage: $0 BUCKET_NAME REGION project image_name service_name" >&2
    echo "Deploys this service in Cloud Run." >&2
    echo >&2
    echo "BUCKET_NAME is required. This is the bucket to proxy." >&2
    echo >&2
    echo "REGION is required. This is the region to run the proxy in (probably should match bucket)." >&2
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
PROJECT="${3:-$(gcloud config get-value project 2>/dev/null)}"
IMAGE_NAME="${4:-gcr.io/${PROJECT}/gcs-streaming-proxy}"
SERVICE_NAME="${5:-gcs-${BUCKET_NAME}}"

if [[ -z "$PROJECT" ]]; then
    echo >&2 "ERROR: Could not determine project. Please specify it explicitly."
    usage
    exit 2
fi

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
