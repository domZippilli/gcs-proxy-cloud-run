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
package main

import (
	"context"
	"io"
	"log"
	"net/http"

	storage "cloud.google.com/go/storage"
)

// get handles GET requests.
func get(ctx context.Context, response http.ResponseWriter, request *http.Request, filters []MediaFilter) {
	// Do the request to get response content stream
	objectName := convertURLtoObject(request.URL.String())
	objectHandle := GCS.Bucket(BUCKET).Object(objectName)

	// get static-serving metadata and set headers
	err := setHeaders(ctx, objectHandle, response)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(response, "404 - Not Found", http.StatusNotFound)
			return
		} else {
			log.Fatal(err)
		}
	}

	// get object content and send it
	objectContent, err := objectHandle.NewReader(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer objectContent.Close()
	if len(filters) > 0 {
		// apply filter chain
		filtered, err := FilterChain(ctx, objectContent, request, response, filters)
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.Copy(response, filtered)
	} else {
		// unfiltered, simple copy
		_, err = io.Copy(response, objectContent)
	}
	if err != nil {
		log.Fatal(err)
	}
	return
}
