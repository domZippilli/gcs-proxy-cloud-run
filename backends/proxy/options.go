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
package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"

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
	media := strings.NewReader("")
	err := error(nil)
	if len(pipeline) > 0 {
		// use a filter pipeline
		_, err = filter.PipelineCopy(ctx, response, media, request, pipeline)
	} else {
		// unfiltered, simple copy
		_, err = io.Copy(response, media)
	}
	if err != nil {
		log.Error().Msgf("SendOptions: %v", err)
	}
}
