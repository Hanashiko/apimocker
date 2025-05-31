package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	faker "github.com/bxcodec/faker/v3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type Endpoint struct {
	Path string `yaml:"path" json:"path"`
	Method string `yaml:"method" json:"method"`
	Data string `yaml:"data" json:"data"`
	Count int `yaml:"count" json:"count"`
	File string `yaml:"file" json:"file"`
	Status int `yaml:"status" json:"status"`
	Delay string `yaml:"delay" json:"delay"`
	Headers map[string]string `yaml:"headers" json:"headers"`
	Errors []ErrorConfig `yaml:"errors" json:"errors"`
}

type ErrorConfig struct {
	Probability float64 `yaml:"probability" json:"probability"`
	Status int `yaml:"status" json:"status"`
	Message string `yaml:"message" json:"message"`
}

type Config struct {
	Port int `yaml:"port" json:"port"`
	Endpoints []Endpoint `yaml:"endpoints" json:"endpoints"`
}

type model struct {
	messages []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("apimocker\n")
	b.WriteString("Running endpoints:\n")
	for _, msg := range m.messages {
		b.WriteString("- " + msg + "\n")
	}
	b.WriteString("\nSupported query parameters:\n")
	b.WriteString("- count: number of items to return\n")
	b.WriteString("- sort: field to sort by\n")
	b.WriteString("- order: asc/desc (default: asc)\n")
	b.WriteString("- filter: field:value to filter by\n")
	b.WriteString("- offset: number of items to skip\n")
	b.WriteString("- limit: alias for count\n")
	b.WriteString("\nPress q to quit.\n")
	return b.String()
}

func loadConfig(path string) (*Config, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		err = yaml.Unmarshal(file, config)
	} else {
		err = json.Unmarshal(file, config)
	}

	for i := range config.Endpoints {
		if config.Endpoints[i].Status == 0{
			config.Endpoints[i].Status = 200
		}
		if config.Endpoints[i].Count == 0 {
			config.Endpoints[i].Count = 1
		}
	}

	return config, err 
}

func parseDuration(delayStr string) time.Duration {
	if delayStr == "" {
		return 0
	}

	if strings.HasSuffix(delayStr, "ms") {
		if ms, err := strconv.Atoi(strings.TrimSuffix(delayStr, "ms")); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	if strings.HasSuffix(delayStr, "s") {
		if s, err := strconv.Atoi(strings.TrimSuffix(delayStr, "s")); err == nil {
			return time.Duration(s) * time.Second
		}
	}
	if strings.HasSuffix(delayStr, "m") {
		if m, err := strconv.Atoi(strings.TrimSuffix(delayStr, "m")); err == nil {
			return time.Duration(m) * time.Minute
		}
	}

	if duration, err := time.ParseDuration(delayStr); err == nil {
		return duration
	}

	return 0
}

func shouldTriggerError(errors []ErrorConfig) (bool, ErrorConfig) {
	if len(errors) == 0 {
		return false, ErrorConfig{}
	}

	for _, errorConfig := range errors {
		if rand.Float64() < errorConfig.Probability {
			return true, errorConfig
		}
	}

	return false, ErrorConfig{}
}

func generateFakeData(schema string, count int) ([]map[string]interface{}, error) {
	var template map[string]string
	if err := json.Unmarshal([]byte(schema), &template); err != nil {
		var fake []map[string]interface{}
		for i := 0; i < count; i++ {
			data := map[string]interface{}{}
			err := faker.FakeData(&data)
			if err != nil {
				return nil, err
			}
			fake = append(fake, data)
		}
		return fake,nil
	}

	supported := map[string]func() interface{}{
		"uuid": func() interface{} { return uuid.New().String() },
		"name": func() interface{} { return faker.Name() },
		"email": func() interface{} { return faker.Email() },
		"bool": func() interface{} { return rand.Intn(2) == 1 },
		"int": func() interface{} { return rand.Intn(1000) },
		"string": func() interface{} { return faker.Word() },
		"lat": func() interface{} { return faker.Latitude() },
		"lng": func() interface{} { return faker.Longitude() },
		"ipv4": func() interface{} { return faker.IPv4() },
		"url": func() interface{} { return faker.URL() },
		"username": func() interface{} { return faker.Username() },
		"password": func() interface{} { return faker.Password() },
		"phone": func() interface{} { return faker.Phonenumber() },
		"date": func() interface{} { return faker.Date() },
		"timestamp": func() interface{} { return time.Now().Unix() },
	}

	var result []map[string]interface{}
	for i := 0; i < count; i++ {
		row := make(map[string]interface{})
		for key, typ := range template {
			if fn, ok := supported[typ]; ok {
				row[key] = fn()
			} else {
				row[key] = nil
			}
		}
		result = append(result, row)
	}

	return result, nil
}

func applyQueryFilters(data []map[string]interface{}, params url.Values) []map[string]interface{} {
	result := data

	if filter := params.Get("filter"); filter != "" {
		parts := strings.SplitN(filter, ":", 2)
		if len(parts) == 2 {
			field := parts[0]
			value := parts[1]
			var filtered []map[string]interface{}
			for _, item := range result {
				if itemValue, exists := item[field]; exists {
					itemStr := fmt.Sprintf("%v", itemValue)
					if strings.Contains(strings.ToLower(itemStr), strings.ToLower(value)) {
						filtered = append(filtered, item)
					}
				}
			}
			result = filtered
		}
	}

	if sortField := params.Get("sort"); sortField != "" {
		order := params.Get("order")
		if order == "" {
			order = "asc"
		}

		sort.Slice(result, func(i, j int) bool {
			val1, exists1 := result[i][sortField]
			val2, exists2 := result[j][sortField]
			
			if !exists1 && !exists2 {
				return false
			}
			if !exists1 {
				return order == "desc"
			}
			if !exists2 {
				return order == "asc"
			}

			str1 := fmt.Sprintf("%v", val1)
			str2 := fmt.Sprintf("%v", val2)

			if order == "desc" {
				return str1 > str2
			}
			return str1 < str2
		})
	}

	offset := 0
	if offsetStr := params.Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	if offset >= len(result) {
		return []map[string]interface{}{}
	}
	if offset > 0 {
		result = result[offset:]
	}

	count := len(result)
	if countStr := params.Get("count"); countStr != "" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		} 
	} else if limitStr := params.Get("limit"); limitStr != "" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	if count < len(result){
		result = result[:count]
	}

	return result
}

func serveFileHandler(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ext := filepath.Ext(path)
		switch ext := strings.ToLower(ext); ext {
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		case ".mp4":
			w.Header().Set("Content-Type", "video/mp4")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		http.ServeFile(w, r, path)
	}
}

func startServer(config *Config) []string {
	var messages []string
	for _, ep := range config.Endpoints {
		path := ep.Path
		method := ep.Method
		msg := fmt.Sprintf("[%s] http://localhost:%d%s", method, config.Port, path)

		if ep.Status != 200 {
			msg += fmt.Sprintf(" (status: %d)", ep.Status)
		}
		if ep.Delay != "" {
			msg += fmt.Sprintf(" (delay: %s)", ep.Delay)
		}
		if len(ep.Errors) > 0 {
			msg += " (with errors)"
		}

		messages = append(messages, msg)

		if ep.File != "" {
			http.HandleFunc(path, serveFileHandler(ep.File))
			continue
		}

		endpoint := ep
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			if endpoint.Delay != "" {
				delay := parseDuration(endpoint.Delay)
				if delay > 0 {
					time.Sleep(delay)
				}
			}

			if shouldError, errorConfig := shouldTriggerError(endpoint.Errors); shouldError {
				w.WriteHeader(errorConfig.Status)
				if errorConfig.Message != "" {
					w.Header().Set("Content-Type", "application/json")
					errorResponse := map[string]string{
						"error": errorConfig.Message,
					}
					json.NewEncoder(w).Encode(errorResponse)
				}
				return
			}

			for key, value := range endpoint.Headers {
				w.Header().Set(key,value)
			}

			params := r.URL.Query()

			count := endpoint.Count
			if countStr := params.Get("count"); countStr != "" {
				if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
					count = parsedCount
				}
			} else if limitStr := params.Get("limit"); limitStr != "" {
				if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
					count = parsedLimit
				}
			}


			generateCount := count
	
			data, err := generateFakeData(endpoint.Data, generateCount)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type","application/json")
				errorResponse := map[string]string{
					"error": "Failed to generate data",
				}
				json.NewEncoder(w).Encode(errorResponse)
				return
			}

			filteredData := applyQueryFilters(data, params)

			w.WriteHeader(endpoint.Status)

			if params.Get("meta") == "true" {
				response := map[string]interface{}{
					"data": filteredData,
					"meta": map[string]interface{}{
						"count": len(filteredData),
						"total": len(data),
						"offset": params.Get("offset"),
						"limit": params.Get("count"),
						"sort": params.Get("sort"),
						"order": params.Get("order"),
						"filter": params.Get("filter"),
						"status": endpoint.Status,
					},
				}
				w.Header().Set("Content-Type","application/json")
				json.NewEncoder(w).Encode(response)
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(filteredData)
			}
		})
	}

	go func() {
		log.Printf("Starting mock server on :%d\n", config.Port)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d",config.Port),nil))
	}()
		return messages
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var configPath string

	var rootCmd = &cobra.Command{
		Use: "apimocker",
		Short: "Lightweight TUI/mock REST API server with query parameter support",
		Long: `apimocker - A lightweight mock REST API server with TUI interface.

Supports dynamic query parameters:
 - count/limit: number of items to return
 - sort: field to sort by
 - order: asc/desc (default: asc)
 - filter: field:value to filter by
 - offset: number of items to skip
 - meta: include metadata in response (true/false)

Additional features:
 - Custom status codes
 - Response delays (ms, s, m or Go duration format)
 - Custom headers
 - Error simulation with probability

Example config:
port: 5050
endpoints:
  - path: /users
    method: GET
    status: 200
    delay: 500ms
    headers:
        X-Test-Mode: "true"
        X-API-Version: "v1"
    data: |
        {
            "id": "uuid",
            "name": "name",
            "email": "email"
        }
    errors:
      - probability: 0.1
        status: 500
        message: "Internal server error"

Examples:
 - GET /users?count=10
 - GET /users?sort=name&order=desc
 - GET /users?filter=name:john&count=5
 - GET /users?offset=10&limit=20&meta=true`,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := loadConfig(configPath)
			if err != nil {
				log.Fatalf("Failed to load config: %v", err)
			}
			messages := startServer(config)
			p := tea.NewProgram(model{messages: messages})
			if err := p.Start(); err != nil {
				log.Fatalf("Error running TUI: %v", err)
			}
		},
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "mock.yaml", "Path to mock config file")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
