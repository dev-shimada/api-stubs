package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"text/template"
	"time"
)

// define the structure of the JSON configuration file
type Matcher struct {
	EqualTo        any `json:"equalTo"`
	Matches        any `json:"matches"`
	DoesNotMatch   any `json:"doesNotMatch"`
	Contains       any `json:"contains"`
	DoesNotContain any `json:"doesNotContain"`
}
type Request struct {
	URL             string `json:"url"`             // パスパラメータ、クエリパラメータを含む完全一致
	URLPattern      string `json:"urlPattern"`      // パスパラメータ、クエリパラメータを含む正規表現での完全一致
	URLPath         string `json:"urlPath"`         // パスパラメータを含む完全一致
	URLPathPattern  string `json:"urlPathPattern"`  // パスパラメータを含む正規表現での完全一致
	URLPathTemplate string `json:"urlPathTemplate"` // パスパラメータを含むテンプレートでの完全一致

	Method          string             `json:"method"`
	QueryParameters map[string]Matcher `json:"queryParameters"`
	PathParameters  map[string]Matcher `json:"pathParameters"`
	// BodyPatterns    []map[string]string          `json:"bodyParameters"`
}
type Response struct {
	Status        int               `json:"status"`
	BodyFileName  string            `json:"bodyFileName"` // bodyFileNameが指定されている場合は、bodyは無視される
	Body          string            `json:"body"`         // bodyFileNameが指定されていない場合は、bodyを使用する
	Headers       map[string]string `json:"headers"`
	Transformaers []string          `json:"transformers"`
}
type Endpoint struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		endpoints, err := loadConfig("config.json")
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		}
		for _, endpoint := range endpoints {
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

			isMatchPath, pathMap := pathMatcher(endpoint, r.URL.RawPath, r.URL.Path)
			isMatchQuery := queryMatcher(endpoint, r.URL.Query())
			if r.Method == endpoint.Request.Method && isMatchPath && isMatchQuery {
				w.WriteHeader(endpoint.Response.Status)

				tpl, err := template.New("response").Parse(responseBody)
				if err != nil {
					slog.Error(fmt.Sprintf("Failed to parse response template: %s", err))
					http.Error(w, "Failed to parse response template", http.StatusInternalServerError)
					return
				}

				type gotParams struct {
					Path  map[string]string
					Query map[string]string
				}
				q := make(map[string]string)
				for k, v := range r.URL.Query() {
					q[k] = v[0]
				}
				gp := gotParams{
					Query: q,
					Path:  pathMap,
				}

				if err := tpl.Execute(w, gp); err != nil {
					slog.Error(fmt.Sprintf("Failed to execute response template: %s", err))
					http.Error(w, "Failed to execute response template", http.StatusInternalServerError)
					return
				}

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
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				slog.Info("Server closed.")
			} else {
				slog.Error(fmt.Sprintf("ListenAndServe: %v", err))
			}
		}
	}()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Info(fmt.Sprintf("HTTP server Shutdown: %v", err))
	}
}

func pathMatcher(endpoint Endpoint, gotRawPath, gotPath string) (bool, map[string]string) {
	// trim trailing slashes
	gotPath = strings.TrimRight(gotPath, "/")
	gotRawPath = strings.TrimRight(gotRawPath, "/")

	var url string
	switch {
	case endpoint.Request.URL != "":
		url = strings.TrimRight(endpoint.Request.URL, "/")
		if gotRawPath != url {
			return false, nil
		}
		return true, nil
	case endpoint.Request.URLPattern != "":
		url = strings.TrimRight(endpoint.Request.URLPattern, "/")
		if !regexp.MustCompile(url).MatchString(gotRawPath) {
			return false, nil
		}
		return true, nil
	case endpoint.Request.URLPath != "":
		url = strings.TrimRight(endpoint.Request.URLPath, "/")
		if gotPath != url {
			return false, nil
		}
		return true, nil
	case endpoint.Request.URLPathPattern != "":
		url = strings.TrimRight(endpoint.Request.URLPathPattern, "/")
		if !regexp.MustCompile(url).MatchString(gotPath) {
			return false, nil
		}
		return true, nil
	case endpoint.Request.URLPathTemplate != "":
		url = strings.TrimRight(endpoint.Request.URLPathTemplate, "/")
	default:
		return false, nil
	}

	// check if the path parameters match
	requredPathUnits := strings.Split(url, "/")
	gotPathUnits := strings.Split(gotPath, "/")
	if len(requredPathUnits) != len(gotPathUnits) {
		return false, nil
	}

	// placeholder->position
	posMap := make(map[string]int)
	for k := range endpoint.Request.PathParameters {
		placeHolder := fmt.Sprintf("{%s}", k)
		if i := slices.Index(requredPathUnits, placeHolder); i == -1 {
			slog.Error(fmt.Sprintf("Path parameter %s not found in path %s", k, gotPath))
			return false, nil
		} else {
			posMap[k] = i
		}
	}

	for k, v := range endpoint.Request.PathParameters {
		if v.EqualTo != nil {
			if gotPathUnits[posMap[k]] != v.EqualTo {
				return false, nil
			}
		}
		if v.Matches != nil {
			if !regexp.MustCompile(v.Matches.(string)).MatchString(gotPathUnits[posMap[k]]) {
				return false, nil
			}
		}
		if v.DoesNotMatch != nil {
			if regexp.MustCompile(v.DoesNotMatch.(string)).MatchString(gotPathUnits[posMap[k]]) {
				return false, nil
			}
		}
		if v.Contains != nil {
			if !strings.Contains(gotPathUnits[posMap[k]], v.Contains.(string)) {
				return false, nil
			}
		}
		if v.DoesNotContain != nil {
			if strings.Contains(gotPathUnits[posMap[k]], v.DoesNotContain.(string)) {
				return false, nil
			}
		}
	}
	ret := make(map[string]string)
	for k, v := range posMap {
		ret[k] = gotPathUnits[v]
	}
	return true, ret
}

func queryMatcher(endpoint Endpoint, gotQuery url.Values) bool {
	for k, v := range endpoint.Request.QueryParameters {
		if v.EqualTo != nil {
			if gotQuery.Get(k) != v.EqualTo {
				return false
			}
		}
		if v.Matches != nil {
			if !regexp.MustCompile(v.Matches.(string)).MatchString(gotQuery.Get(k)) {
				return false
			}
		}
		if v.DoesNotMatch != nil {
			if regexp.MustCompile(v.DoesNotMatch.(string)).MatchString(gotQuery.Get(k)) {
				return false
			}
		}
		if v.Contains != nil {
			if !strings.Contains(gotQuery.Get(k), v.Contains.(string)) {
				return false
			}
		}
		if v.DoesNotContain != nil {
			if strings.Contains(gotQuery.Get(k), v.DoesNotContain.(string)) {
				return false
			}
		}
	}
	return true
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
