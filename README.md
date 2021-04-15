# gcs-proxy-cloud-run

A Cloud Run service to proxy a GCS bucket. Useful for conditional serving logic, transcoding, certain security features, etc.

This contrasts with [gcs-proxy-cloud-function](http://github.com/domZippilli/gcs-proxy-cloud-function) in that it uses Cloud Run, which conveys an important advantage -- streaming responses.

## Deployment

As a prerequisite, [enable the Cloud Build API](https://console.cloud.google.com/apis/library/cloudbuild.googleapis.com) for your project.

Also, if you haven't done so, ensure `gcloud` is using the correct credentials. Usually, a combination of `gcloud auth login`, `gcloud config set project`, and optionally `gcloud auth revoke` when you are finished will do the job.

Then, build and deploy using the provided shell scripts. The default arguments should work for trials. You will need to provide a bucket and a region in which to run the proxy to the `deploy.sh` script.

```shell
./build.sh && ./deploy.sh mybucket us-central1
```

Users would do well to tune the runtime settings for the service to suit their needs, or in response to benchmarks.
