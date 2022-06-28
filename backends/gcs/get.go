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
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/DomZippilli/gcs-proxy-cloud-function/filter"

	storage "cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
)

// Read returns objects from a GCS bucket, mapping the URL to object names.
// Media caching is bypassed.
func Read(ctx context.Context, response http.ResponseWriter,
	request *http.Request, pipeline filter.Pipeline) {
	noCache := func(s string) ([]byte, bool) {
		return nil, false
	}
	ReadWithCache(ctx, response, request, pipeline, noCache, filter.Pipeline{})
}

// CacheGet defines how CachedGet will try to get media from the cache.
type CacheGet func(string) ([]byte, bool)

// ReadWithCache returns objects from a GCS bucket, mapping the URL to object names.
// Cached media may be served, sparing a trip to GCS.
//
// Filters in missPipeline will be applied on cache misses. A cache fill
// filter is a good idea here.
//
// Filters in hitPipeline will be applied on cache hits. Reducing the pipeline
// to not repeat steps done on fill (e.g., compression, transcoding) is a good
// idea here.
func ReadWithCache(ctx context.Context, response http.ResponseWriter,
	request *http.Request, missPipeline filter.Pipeline, cacheGet CacheGet,
	hitPipeline filter.Pipeline) {
	// normalize path
	objectName := common.NormalizePath(request.URL.Path)

	// get the object handle and headers. Headers are always cached and obey
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

	// try the media cache
	gcsMedia := &common.ReadFFwder{
		Size: objectAttrs.Size,
	}
	var pipeline filter.Pipeline
	maybeMedia, hit := cacheGet(objectName)
	if hit {
		log.Debug().Msgf("gcs ReadWithCache: HIT")
		gcsMedia.Media = bytes.NewReader(maybeMedia)
		// transformations may be cached; use cached content length
		response.Header().Set("Content-Length", fmt.Sprint(len(maybeMedia)))
		pipeline = hitPipeline
	} else {
		log.Debug().Msgf("gcs ReadWithCache: MISS")
		// get object content and send it
		// TODO(domz): need an aggressive reader
		objectContent, err := objectHandle.NewReader(ctx)
		if err != nil {
			log.Error().Msgf("get: %v", err)
		}
		defer objectContent.Close()
		gcsMedia.Media = objectContent
		pipeline = missPipeline
	}

	// serve the media
	if len(pipeline) > 0 {
		// use a filter pipeline
		filterReader := filter.PipelineCopy(ctx, response, gcsMedia, request, pipeline)
		finalMedia := &common.ReadFFwder{
			Media: filterReader,
			// As a compromise, we are going to use the object's size here.
			// It doesn't matter if the request doesn't have a range header.
			// And, it works with a range header if your pipeline doesn't modify the body.
			// But, if you try to use a range header on a modified body, behavior is undefined.
			// TODO: Consider blocking range requests in some circumstances.
			Size: objectAttrs.Size,
		}
		http.ServeContent(response, request, objectHandle.ObjectName(), objectAttrs.Created, finalMedia)
	} else {
		// unfiltered, simple copy
		http.ServeContent(response, request, objectHandle.ObjectName(), objectAttrs.Created, gcsMedia)
	}
	if err != nil {
		log.Error().Msgf("ReadWithCache: %v", err)
	}
}
