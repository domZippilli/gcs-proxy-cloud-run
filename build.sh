#!/usr/bin/env bash
set -euf -o pipefail

function usage(){
    echo >&2
    echo "Usage: $0 image_name project" >&2
    echo "Builds this service container in Google Container Registry (gcr.io)." >&2
    echo >&2
    echo "image_name is optional. default value is gcs-streaming-proxy." >&2
    echo >&2
    echo "project is optional; your gcloud config project will be used if it" >&2
    echo "is not provided." >&2
    echo >&2
}

IMAGE_NAME=${1-}
IMAGE_NAME="${1:-gcs-streaming-proxy}"
PROJECT=${2-}
PROJECT="${2:-$(gcloud config get-value project)}"
TAG=gcr.io/"${PROJECT}"/"${IMAGE_NAME}"

# quick and dirty way to catch if the user asks for help, like --help
# downside: you can't tag the image as *help or just "-h"
if [[ "${IMAGE_NAME}" == *help ]] || [[ "${IMAGE_NAME}" == "-h" ]]; then
    usage
    exit
fi

gcloud builds submit --tag "${TAG}"

echo Container image built:
echo "${TAG}" 