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
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	storage "cloud.google.com/go/storage"
	cache "github.com/patrickmn/go-cache"
)

// setHeaders will transfer HTTP headers from GCS metadata to the response.
func setHeaders(ctx context.Context, objectHandle *storage.ObjectHandle,
	output http.ResponseWriter) (err error) {

	// get object metadata. Use a cache to speed up TTFB.
	objectAttrs, err := getAttrs(ctx, objectHandle)
	if err != nil {
		return err
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

func getAttrs(ctx context.Context, objectHandle *storage.ObjectHandle) (
	objectAttrs *storage.ObjectAttrs, err error) {
	// get object metadata. Use a cache to speed up TTFB.
	maybeAttrs, hit := contentHeaderCache.Get(objectHandle.ObjectName())
	if hit {
		objectAttrs = maybeAttrs.(*storage.ObjectAttrs)
	} else {
		objectAttrs, err = objectHandle.Attrs(ctx)
		if err != nil {
			return
		}
		// cache the result, honoring Cache-Control: max-age
		expiry := cache.DefaultExpiration
		cacheControl := objectAttrs.CacheControl
		if cacheControl != "" &&
			strings.HasPrefix(cacheControl, "max-age") {
			ccSecs, err := strconv.Atoi(strings.Split(cacheControl, "=")[1])
			if err != nil {
				log.Println(err)
			} else {
				expiry = time.Second * time.Duration(ccSecs)
			}
		}
		contentHeaderCache.Set(objectHandle.ObjectName(), objectAttrs, expiry)
	}
	return
}
