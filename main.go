package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Endpoint struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	Response struct {
		Status int         `json:"status"`
		Body   interface{} `json:"body"`
	} `json:"response"`
}

func loadConfig(filePath string) ([]Endpoint, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var endpoints []Endpoint
	byteValue, _ := ioutil.ReadAll(file)
	err = json.Unmarshal(byteValue, &endpoints)
	if err != nil {
		return nil, err
	}

	return endpoints, nil
}

func main() {
	endpoints, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, endpoint := range endpoints {
			if r.URL.Path == endpoint.Endpoint && r.Method == endpoint.Method {
				w.WriteHeader(endpoint.Response.Status)
				json.NewEncoder(w).Encode(endpoint.Response.Body)
				return
			}
		}
		http.NotFound(w, r)
	})

	fmt.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
