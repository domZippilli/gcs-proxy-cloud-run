// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gcs

import (
	"context"
	"net/http"
	"strings"

	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/DomZippilli/gcs-proxy-cloud-function/filter"

	storage "cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
)

// ReadMetadata returns object metadata from a GCS bucket, mapping the URL to
// object names.
func ReadMetadata(ctx context.Context, response http.ResponseWriter,
	request *http.Request, pipeline filter.Pipeline) {
	// normalize path
	objectName := common.NormalizePath(request.URL.Path)

	// get the object handle and headers. Attributes are always cached and obey
	// Cache-Control header, so this will not call GCS unless there's a miss.
	// In general, header hits and media hits should line up.
	objectHandle := gcs.Bucket(bucket).Object(objectName)
	// get object metadata and set headers
	objectAttrs, _ := getAttrs(ctx, objectHandle)
	err := setHeaders(ctx, objectAttrs, response)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(response, "", http.StatusNotFound)
			return
		} else {
			log.Error().Msgf("get: %v", err)
		}
	}

	// serve the metadata
	emptyMedia := &common.ReadFFwder{
		Media: strings.NewReader(""),
		Size:  0,
	}
	if len(pipeline) > 0 {
		// use a filter pipeline
		filterReader := filter.PipelineCopy(ctx, response, emptyMedia, request, pipeline)
		finalMedia := &common.ReadFFwder{
			Media: filterReader,
			// Adding body in HEAD pipelines is not conformant HTTP; in this proxy, it leads to undefined behavior
			Size: 0,
		}
		http.ServeContent(response, request, objectHandle.ObjectName(), objectAttrs.Created, finalMedia)
	} else {
		// unfiltered, simple copy
		http.ServeContent(response, request, objectHandle.ObjectName(), objectAttrs.Created, emptyMedia)
	}
	if err != nil {
		log.Error().Msgf("ReadMetadata: %v", err)
	}
}
