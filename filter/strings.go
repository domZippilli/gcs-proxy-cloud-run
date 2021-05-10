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
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
)

// ToLower applies bytes.ToLower to the media.
//
// This is an example of a streaming filter. This will use very little memory
// and add very little latency to responses.
func ToLower(ctx context.Context, handle MediaFilterHandle) error {
	defer handle.input.Close()
	defer handle.output.Close()
	buf := make([]byte, 4096)
	for {
		_, err := handle.input.Read(buf)
		buf = bytes.ToLower(buf)
		handle.output.Write(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			return FilterError(handle, http.StatusInternalServerError, "lower filter: %v", err)
		}
	}
	return nil
}

// Intercalate will split the media, insert a value between elements,
// and return the concatenated result.
//
// This function should be called from a lambda that applies desired values for
// intercalation, leaving only ctx and handle for use as a MediaFilter.
//
// For example:
//   func(ctx context.Context, handle MediaFilterHandle) error {
//   	return Intercalate(ctx, handle, "\n", "f")
//   },
//
// This is an example of a store-and-forward filter, in that it loads the
// entire response to perform its transformation, so it will use memory at least
// equal to the source, and add its processing time to latency.
func Intercalate(ctx context.Context, handle MediaFilterHandle, separator string, insertValue string) error {
	defer handle.input.Close()
	defer handle.output.Close()
	// load the media into memory
	media := new(bytes.Buffer)
	if _, err := io.Copy(media, handle.input); err != nil {
		return FilterError(handle, http.StatusInternalServerError, "intercalate filter: %v", err)
	}
	// make it a string
	mediaString := media.String()
	// split and intercalate
	inputStrings := strings.Split(mediaString, separator)
	outputStrings := []string{}
	for i, token := range inputStrings {
		outputStrings = append(outputStrings, token)
		if i <= len(inputStrings)-2 {
			outputStrings = append(outputStrings, separator+insertValue)
		}
	}
	for _, line := range outputStrings {
		handle.output.Write([]byte(string(line)))
	}
	return nil
}
