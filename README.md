# gcs-proxy-cloud-run

A Cloud Run service to proxy a GCS bucket. Useful for conditional serving logic, transcoding, certain security features, etc.

Many use cases can be satisfied by using [built-in static website hosting for GCS with a Cloud Load Balancer](https://cloud.google.com/storage/docs/hosting-static-website). If there are limitations of that feature that are blocking you, this proxy approach might be for you.

This contrasts with [gcs-proxy-cloud-function](http://github.com/domZippilli/gcs-proxy-cloud-function) in that it uses Cloud Run, which conveys an important advantage -- streaming responses, which greatly improve TTFB and reduce memory usage. If at all possible, use this.

Note, also, that since this is just a containerized proxy, you can also run it any other way you would run containers, not just in Cloud Run.

**DISCLAIMER:** This code is offered as a proof-of-concept only. It should not be used unmodified in production. Your use of this code is at your own risk. See `LICENSE` for more information.

## Quickstart Deployment

As a prerequisite, [enable the Cloud Build API](https://console.cloud.google.com/apis/library/cloudbuild.googleapis.com) for your project.

Also, if you haven't done so, ensure `gcloud` is using the correct credentials. Usually, a combination of `gcloud auth login`, `gcloud config set project`, and optionally `gcloud auth revoke` when you are finished will do the job.

Then, build and deploy using the provided shell scripts. The default arguments should work for trials. You will need to provide a bucket and a region in which to run the proxy to the `deploy.sh` script.

```shell
./build.sh && ./deploy.sh mybucket us-central1
```

Final output should include a `Service URL`, like this:

```
Service URL: https://gcs-mybucket-urqwoijds-uc.a.run.app
```

This is the location where you can access the proxy.

The default configuration makes a simple, read-only public endpoint with the proxy backed by the given GCS bucket. Users would do well to tune the runtime settings for the service to suit their needs.

## Configuration

Configuration for the HTTP behavior of the proxy is encoded in `main/config/config.go`. Rather than using a separate config file, the configuration can be expressed in simple Go code and then compiled into the service and deployed.

HTTP methods have a configuration function like this:

```go
// This function will be called in main.go for GET requests
func GET(ctx context.Context, output http.ResponseWriter, input *http.Request) {
    proxyhttp.Get(ctx, output, input, []filter.MediaFilter{})
}
```

In this case, `GET` is the function you configure; you can do all kinds of pre-processing like examining or rewriting request headers, etc. Then, `get` is the standard `proxyhttp/get.go` implementation which reads objects from GCS.

This configuration is referred to from `main/main.go` at the entrypoint for the function. In general, the idea is to keep that file pretty static and open ended, and push most customization into `config/config.go`.

## Media Filters

The fourth argument to `get` above is a slice of `main/filter/MediaFilter`. **Media filters allow you to execute arbitrary Go code against the media of requests**. A number of example and possibly useful filters can be found in `main/filter/filter.go`. Inserting filters into the processing of a response is simple; just add them to the slice in `config/config.go`:

```go
// This function will be called in main.go for GET requests
func GET(ctx context.Context, output http.ResponseWriter, input *http.Request) {
    proxyhttp.Get(ctx, output, input, []filter.MediaFilter{
        filter.LowerFilter,
    })
}
```

This will apply `LowerFilter` to all responses, converting all characters to lowercase.

Multiple filters can be chained together by adding them to the slice. Filters will be processed in the order they are listed in the slice.

For more information, check out the documentation in `main/filter/filter.go`.


## Copyright

``` text
Copyright 2021 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
