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
package proxy

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/DomZippilli/gcs-proxy-cloud-function/filter"

	"github.com/rs/zerolog/log"
)

// SendOptions sends an HTTP OPTIONS response for this proxy.
// Pipelines will not be able to modify the response (other than to add
// body contents, which is in error), but logging is still ok.
func SendOptions(ctx context.Context, response http.ResponseWriter,
	request *http.Request, pipeline filter.Pipeline) {
	response.Header().Add("Allow", "OPTIONS, GET, HEAD")
	response.WriteHeader(http.StatusNoContent)
	err := error(nil)
	emptyMedia := &common.ReadFFwder{
		Media: strings.NewReader(""),
		Size:  0,
	}
	if len(pipeline) > 0 {
		// use a filter pipeline
		filterReader := filter.PipelineCopy(ctx, response, emptyMedia, request, pipeline)
		finalMedia := &common.ReadFFwder{
			Media: filterReader,
			// This proxy doesn't support Range on OPTIONS. 0 here will effectively block any seeks.
			Size: 0,
		}
		http.ServeContent(response, request, "", time.Now(), finalMedia)
	} else {
		// unfiltered, simple copy
		http.ServeContent(response, request, "", time.Now(), emptyMedia)
	}
	if err != nil {
		log.Error().Msgf("SendOptions: %v", err)
	}
}
