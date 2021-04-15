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
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"time"

	storage "cloud.google.com/go/storage"
	cache "github.com/patrickmn/go-cache"
)

var BUCKET string
var GCS *storage.Client
var contentHeaderCache = cache.New(90*time.Second, 10*time.Minute)

func setup() {
	// set the bucket name from environment variable
	BUCKET = os.Getenv("BUCKET_NAME")

	// initialize the client
	c, err := storage.NewClient(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	GCS = c
}

func main() {
	log.Print("starting server...")
	setup()
	http.HandleFunc("/", ProxyGCS)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// ProxyGCS is the entry point for the cloud function, providing a proxy that
// permits HTTP protocol usage of a GCS bucket's contents.
func ProxyGCS(output http.ResponseWriter, input *http.Request) {
	ctx := context.Background()

	// route HTTP methods to appropriate handlers
	switch input.Method {
	case http.MethodGet:
		get(ctx, output, input)
	default:
		http.Error(output, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
	}
	return
}

// convertURLtoObject converts a URL to an appropriate object path. This also
// includes redirecting root requests "/" to index.html.
func convertURLtoObject(url string) (object string) {
	switch url {
	case "/":
		return "index.html"
	default:
		return url[1:]
	}
}

// get handles GET requests.
func get(ctx context.Context, output http.ResponseWriter, input *http.Request) {
	// Do the request to get response content stream
	objectName := convertURLtoObject(input.URL.String())
	objectHandle := GCS.Bucket(BUCKET).Object(objectName)

	// get static-serving metadata and set headers
	err := setHeaders(ctx, objectHandle, output)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(output, "404 - Not Found", http.StatusNotFound)
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

	_, err = io.Copy(output, objectContent)
	if err != nil {
		log.Fatal(err)
	}
	return
}

// setHeaders will transfer HTTP headers from GCS metadata to the response.
func setHeaders(ctx context.Context, objectHandle *storage.ObjectHandle,
	output http.ResponseWriter) (err error) {

	// get object metadata. Use a cache to speed up TTFB.
	var objectAttrs *storage.ObjectAttrs
	maybeAttrs, hit := contentHeaderCache.Get(objectHandle.ObjectName())
	if hit {
		objectAttrs = maybeAttrs.(*storage.ObjectAttrs)
	} else {
		objectAttrs, err = objectHandle.Attrs(ctx)
		if err != nil {
			return err
		}
		// cache the result, honoring Cache-Control: max-age
		expiry := cache.DefaultExpiration
		cacheControl := objectAttrs.CacheControl
		if cacheControl != "" &&
			strings.HasPrefix(cacheControl, "max-age") {
			ccSecs, err := strconv.Atoi(strings.Split(cacheControl, "=")[1])
			if err != nil {
				return err
			}
			expiry = time.Second * time.Duration(ccSecs)
		}
		contentHeaderCache.Set(objectHandle.ObjectName(), objectAttrs, expiry)
	}

	// set all headers
	if objectAttrs.CacheControl != "" {
		output.Header().Set("Cache-Control", objectAttrs.CacheControl)
	}
	if objectAttrs.ContentEncoding != "" {
		output.Header().Set("Content-Encoding", objectAttrs.ContentEncoding)
	}
	if objectAttrs.ContentLanguage != "" {
		output.Header().Set("Content-Language", objectAttrs.ContentLanguage)
	}
	if objectAttrs.ContentType != "" {
		output.Header().Set("Content-Type", objectAttrs.ContentType)
	}
	output.Header().Set("Content-Length", fmt.Sprint(objectAttrs.Size))
	return
}
