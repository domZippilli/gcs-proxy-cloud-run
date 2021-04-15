package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	storage "cloud.google.com/go/storage"
)

var BUCKET string
var GCS *storage.Client

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

// get handles GET requests.
func get(ctx context.Context, output http.ResponseWriter, input *http.Request) {
	// Do the request to get response content stream
	objectName := convertURLtoObject(input.URL.String())
	responseContent, err := GCS.Bucket(BUCKET).Object(objectName).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(output, "404 - Not Found", http.StatusNotFound)
			return
		} else {
			log.Fatal(err)
		}
	}
	defer responseContent.Close()

	// Copy with a 16MB buffer (aligns well with optimal chunk size for GCS)
	_, err = io.CopyBuffer(output, responseContent, make([]byte, 16*1024^2))
	if err != nil {
		log.Fatal(err)
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
