// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gcs

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/DomZippilli/gcs-proxy-cloud-function/filter"

	storage "cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
)

// Read returns objects from a GCS bucket, mapping the URL to object names.
// Media caching is bypassed.
func ReadMetadata(ctx context.Context, response http.ResponseWriter,
	request *http.Request, pipeline filter.Pipeline) {
	// normalize path
	objectName := common.NormalizePath(request.URL.Path)

	// get the object handle and headers. Attributes are always cached and obey
	// Cache-Control header, so this will not call GCS unless there's a miss.
	// In general, header hits and media hits should line up.
	objectHandle := gcs.Bucket(bucket).Object(objectName)
	// get static-serving metadata and set headers
	err := setHeaders(ctx, objectHandle, response)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(response, "", http.StatusNotFound)
			return
		} else {
			log.Error().Msgf("get: %v", err)
		}
	}

	// serve the metadata
	media := strings.NewReader("")
	if len(pipeline) > 0 {
		// use a filter pipeline
		_, err = filter.PipelineCopy(ctx, response, media, request, pipeline)
	} else {
		// unfiltered, simple copy
		_, err = io.Copy(response, media)
	}
	if err != nil {
		log.Error().Msgf("get: %v", err)
	}
}
