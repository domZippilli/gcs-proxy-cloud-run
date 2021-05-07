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
package filter

import (
	"context"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

// LogRequest doesn't modify the request. It simply logs it.
func LogRequest(ctx context.Context, handle MediaFilterHandle) error {
	defer handle.input.Close()
	defer handle.output.Close()
	bytesSent, err := io.Copy(handle.output, handle.input)
	if err != nil {
		return FilterError(handle, http.StatusInternalServerError, "logrequest filter: %v", err)
	}
	log.Info().Msgf("%v %v %v sent %vB",
		handle.request.RemoteAddr,
		handle.request.Method,
		handle.request.URL,
		bytesSent)
	return err
}
