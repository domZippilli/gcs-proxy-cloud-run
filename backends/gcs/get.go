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

	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/DomZippilli/gcs-proxy-cloud-function/filter"

	storage "cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
)

// Get returns objects from a GCS bucket, mapping the URL to object names.
func Get(ctx context.Context, response http.ResponseWriter, request *http.Request, pipeline filter.Pipeline) {
	// identify the object path
	objectName := common.ConvertURLtoObject(request.URL.String())
	// Do the request to get object media stream
	objectHandle := common.GCS.Bucket(common.BUCKET).Object(objectName)

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

	// get object content and send it
	objectContent, err := objectHandle.NewReader(ctx)
	if err != nil {
		log.Error().Msgf("get: %v", err)
	}
	defer objectContent.Close()
	if len(pipeline) > 0 {
		// use a filter pipeline
		_, err = filter.PipelineCopy(ctx, response, objectContent, request, pipeline)
	} else {
		// unfiltered, simple copy
		_, err = io.Copy(response, objectContent)
	}
	if err != nil {
		log.Error().Msgf("get: %v", err)
	}
	return
}
