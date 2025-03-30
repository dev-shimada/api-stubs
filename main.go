package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"
)

type Endpoint struct {
	Request struct {
		URL             string `json:"url"`
		URLPattern      string `json:"urlPattern"`
		URLPath         string `json:"urlPath"`
		URLPathPattern  string `json:"urlPathPattern"`
		URLPathTemplate string `json:"urlPathTemplate"`

		Method          string            `json:"method"`
		QueryParameters map[string]string `json:"queryParameters"`
		// BodyPatterns    []map[string]string          `json:"bodyParameters"`
	} `json:"request"`
	Response struct {
		Status        int               `json:"status"`
		BodyFileName  string            `json:"bodyFileName"`
		Body          string            `json:"body"`
		Headers       map[string]string `json:"headers"`
		Transformaers []string          `json:"transformers"`
	} `json:"response"`
}

func loadConfig(filePath string) ([]Endpoint, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var endpoints []Endpoint
	byteValue, _ := io.ReadAll(file)
	err = json.Unmarshal(byteValue, &endpoints)
	if err != nil {
		return nil, err
	}

	return endpoints, nil
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		endpoints, err := loadConfig("config.json")
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
		for _, endpoint := range endpoints {
			var url string
			switch {
			case endpoint.Request.URL != "":
				url = endpoint.Request.URL
			case endpoint.Request.URLPattern != "":
				url = endpoint.Request.URLPattern
			case endpoint.Request.URLPath != "":
				url = endpoint.Request.URLPath
			case endpoint.Request.URLPathPattern != "":
				url = endpoint.Request.URLPathPattern
			case endpoint.Request.URLPathTemplate != "":
				url = endpoint.Request.URLPathTemplate
			default:
				url = endpoint.Request.URL
			}

			var responseBody string
			switch {
			case endpoint.Response.BodyFileName != "":
				file, err := os.Open(endpoint.Response.BodyFileName)
				if err != nil {
					http.Error(w, "Failed to open body file", http.StatusInternalServerError)
					return
				}
				defer file.Close()
			case endpoint.Response.Body != "":
				responseBody = endpoint.Response.Body
			default:
				slog.Error("Response body is empty")
				http.Error(w, "Response body is empty", http.StatusInternalServerError)
				return
			}

			if r.URL.Path == url && r.Method == endpoint.Request.Method {
				w.WriteHeader(endpoint.Response.Status)

				tpl, err := template.New("response").Parse(responseBody)
				if err != nil {
					slog.Error(fmt.Sprintf("Failed to parse response template: %s", err))
					http.Error(w, "Failed to parse response template", http.StatusInternalServerError)
					return
				}
				if err := tpl.Execute(w, endpoint.Request); err != nil {
					slog.Error(fmt.Sprintf("Failed to execute response template: %s", err))
					http.Error(w, "Failed to execute response template", http.StatusInternalServerError)
					return
				}
				// w.Write([]byte(responseBody))
				return
			}
		}
		http.NotFound(w, r)
	})

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	// defer stop()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	slog.Info("Server is running at :8080 Press CTRL-C to exit.")
	go srv.ListenAndServe()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Info(fmt.Sprintf("HTTP server Shutdown: %v", err))
	}
}
