package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

type AuthConfig struct {
	Type string `yaml:"type" json:"type"`
	Token string `yaml:"token" json:"token"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

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
	Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`
}

type ErrorConfig struct {
	Probability float64 `yaml:"probability" json:"probability"`
	Status int `yaml:"status" json:"status"`
	Message string `yaml:"message" json:"message"`
}

type LogConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Format string `yaml:"format" json:"format"`
	Output string `yaml:"output" json:"output"`
}

type Config struct {
	Port int `yaml:"port" json:"port"`
	Endpoints []Endpoint `yaml:"endpoints" json:"endpoints"`
	Logging LogConfig `yaml:"logging" json:"logging"`
}

type RequestLog struct {
	Timestamp string `json:"timestamp"`
	Method string `json:"method"`
	Path string `json:"path"`
	Query string `json:"query,omitempty"`
	StatusCode int `json:"status_code"`
	ResponseTime string `json:"response_time"`
	UserAgent string `json:"user_agent,omitempty"`
	RemoteAddr string `json:"remote_addr"`
	ContentLength int64 `json:"content_length"`
	AuthType string `json:"auth_type,omitempty"`
	AuthResult string `json:"auth_result,omitempty"`
}

type Logger struct {
	writer io.Writer
	format string
}

func NewLogger(config LogConfig) (*Logger, error) {
	if !config.Enabled {
		return &Logger{writer: io.Discard, format: config.Format}, nil
	}

	var writer io.Writer
	if config.Output == "stdout" || config.Output == "" {
		writer = os.Stdout
	} else {
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v",err)
		}
		writer = file
	}

	format := config.Format
	if format != "json" && format != "plain" {
		format = "plain"
	}
	
	return &Logger{writer: writer, format: format}, nil
}

func (l *Logger) LogRequest(reqLog RequestLog) {
	if l.writer == io.Discard {
		return
	}

	if l.format == "json" {
		data, err := json.Marshal(reqLog)
		if err != nil {
			return
		}
		fmt.Fprintln(l.writer, string(data))
	} else {
		query := ""
		if reqLog.Query != "" {
			query = "?" + reqLog.Query
		}
		authInfo := ""
		if reqLog.AuthType != "" {
			authInfo = fmt.Sprintf(" - Auth: %s (%s)", reqLog.AuthType, reqLog.AuthResult)
		}
		fmt.Fprintf(l.writer, "[%s] %s %s%s - %d - %s - %s - %d bytes%s\r\n",
			reqLog.Timestamp,
			reqLog.Method,
			reqLog.Path,
			query,
			reqLog.StatusCode,
			reqLog.ResponseTime,
			reqLog.RemoteAddr,
			reqLog.ContentLength,
			authInfo,
			)
	}
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
	b.WriteString("\nAuthentication types supported:\n")
	b.WriteString("- Basic Auth: Authorization: Basic <base64(username:password)>\n")
	b.WriteString("- Bearer Token: Authorization: Bearer <token>\n")
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

	if config.Logging.Format == "" {
		config.Logging.Format = "plain"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stdout"
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

func authenticateRequest(r *http.Request, authConfig *AuthConfig) (bool, string, string){
	if authConfig == nil {
		return true, "", "no-auth"
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false, authConfig.Type, "missing-auth"
	}

	switch strings.ToLower(authConfig.Type) {
	case "basic":
		return authenticateBasic(authHeader, authConfig)
	case "bearer":
		return authenticateBearer(authHeader, authConfig)
	default:
		return false, authConfig.Type, "invalid-auth-type"
	}
}

func authenticateBasic(authHeader string, authConfig *AuthConfig) (bool, string, string) {
	if !strings.HasPrefix(authHeader, "Basic "){
		return false, "basic", "invalid-basic-format"
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false, "basic", "invalid-base64"
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return false, "basic", "invalid-credentials-format"
	}

	username, password := credentials[0], credentials[1]
	if username == authConfig.Username && password == authConfig.Password {
		return true, "basic", "success"
	}
	return false, "basic", "invalid-credentials"
}

func authenticateBearer(authHeader string, authConfig *AuthConfig) (bool, string, string) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false, "bearer", "invalid-bearer-format"
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authConfig.Token {
		return true, "bearer", "success"
	}

	return false, "bearer", "invalid-token"
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
		if parsedCount, err := strconv.Atoi(limitStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	if count < len(result){
		result = result[:count]
	}

	return result
}

func serveFileHandler(path string, endpoint Endpoint, logger *Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		statusCode := 200

		authSuccess, authType, authResult := authenticateRequest(r, endpoint.Auth)
		if !authSuccess {
			statusCode = http.StatusUnauthorized
			w.WriteHeader(statusCode)
			w.Header().Set("Content-Type", "application/json")
			errorResponse := map[string]string{
				"error": "Authentication required",
			}
			data, _ := json.Marshal(errorResponse)
			w.Write(data)

			duration := time.Since(start)
			reqLog := RequestLog{
				Timestamp: start.Format(time.RFC3339),
				Method: r.Method,
				Path: r.URL.Path,
				Query: r.URL.RawQuery,
				StatusCode: statusCode,
				ResponseTime: duration.String(),
				UserAgent: r.Header.Get("User-Agent"),
				RemoteAddr: r.RemoteAddr,
				ContentLength: int64(len(data)),
				AuthType: authType,
				AuthResult: authResult,
			}
			logger.LogRequest(reqLog)
			return
		}

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

		duration := time.Since(start)
		fileInfo, _ := os.Stat(path)
		contentLength := int64(0)
		if fileInfo != nil {
			contentLength = fileInfo.Size()
		}

		reqLog := RequestLog{
			Timestamp: start.Format(time.RFC3339),
			Method: r.Method,
			Path: r.URL.Path,
			Query: r.URL.RawQuery,
			StatusCode: 200,
			ResponseTime: duration.String(),
			UserAgent: r.Header.Get("User-Agent"),
			RemoteAddr: r.RemoteAddr,
			ContentLength: contentLength,
			AuthType: authType,
			AuthResult: authResult,
		}
		logger.LogRequest(reqLog)
	}
}

func createLoggingHandler(endpoint Endpoint, logger *Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		statusCode := endpoint.Status

		if r.Method != endpoint.Method {
			statusCode = http.StatusMethodNotAllowed
			http.Error(w, "Method Not Allowed", statusCode)

			duration := time.Since(start)
			reqLog := RequestLog{
				Timestamp: start.Format(time.RFC3339),
				Method: r.Method,
				Path: r.URL.Path,
				Query: r.URL.RawQuery,
				StatusCode: statusCode,
				ResponseTime: duration.String(),
				UserAgent: r.Header.Get("User-Agent"),
				RemoteAddr: r.RemoteAddr,
				ContentLength: 0,
			}
			logger.LogRequest(reqLog)
			return
		}

		authSuccess, authType, authResult := authenticateRequest(r, endpoint.Auth)
		if !authSuccess {
			statusCode = http.StatusUnauthorized
			w.WriteHeader(statusCode)
			w.Header().Set("Content-Type", "application/json")
			errorResponse := map[string]string{
				"error": "Authentication required",
			}

			data, _ := json.Marshal(errorResponse)
			contentLength := int64(len(data))
			w.Write(data)

			duration := time.Since(start)
			reqLog := RequestLog{
				Timestamp: start.Format(time.RFC3339),
				Method: r.Method,
				Path: r.URL.Path,
				Query: r.URL.RawQuery,
				StatusCode: statusCode,
				ResponseTime: duration.String(),
				UserAgent: r.Header.Get("User-Agent"),
				RemoteAddr: r.RemoteAddr,
				ContentLength: contentLength,
				AuthType: authType,
				AuthResult: authResult,
			}
			logger.LogRequest(reqLog)
			return
		}
		
		if endpoint.Delay != "" {
			delay := parseDuration(endpoint.Delay)
			if delay > 0 {
				time.Sleep(delay)
			}
		}

	if shouldError, errorConfig := shouldTriggerError(endpoint.Errors); shouldError {
			statusCode = errorConfig.Status
			w.WriteHeader(statusCode)
			contentLength := int64(0)
			if errorConfig.Message != "" {
				w.Header().Set("Content-Type", "application/json")
				errorResponse := map[string]string{
					"error": errorConfig.Message,
				}
				data, _ := json.Marshal(errorResponse)
				contentLength = int64(len(data))
				w.Write(data)
			}

			duration := time.Since(start)
			reqLog := RequestLog{
				Timestamp: start.Format(time.RFC3339),
				Method: r.Method,
				Path: r.URL.Path,
				Query: r.URL.RawQuery,
				StatusCode: statusCode,
				ResponseTime: duration.String(),
				UserAgent: r.Header.Get("User-Agent"),
				RemoteAddr: r.RemoteAddr,
				ContentLength: contentLength,
				AuthType: authType,
				AuthResult: authResult,
			}
			logger.LogRequest(reqLog)
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
			statusCode = http.StatusInternalServerError
			w.WriteHeader(statusCode)
			w.Header().Set("Content-Type","application/json")
			errorResponse := map[string]string{
				"error": "Failed to generate data",
			}
			responseData, _ := json.Marshal(errorResponse)
			w.Write(responseData)
			
			duration := time.Since(start)
			reqLog := RequestLog{
				Timestamp: start.Format(time.RFC3339),
				Method: r.Method,
				Path: r.URL.Path,
				Query: r.URL.RawQuery,
				StatusCode: statusCode,
				ResponseTime: duration.String(),
				UserAgent: r.Header.Get("User-Agent"),
				RemoteAddr: r.RemoteAddr,
				ContentLength: int64(len(responseData)),
				AuthType: authType,
				AuthResult: authResult,
			}
			logger.LogRequest(reqLog)
			return
		}

		filteredData := applyQueryFilters(data, params)

		w.WriteHeader(statusCode)

		var responseData []byte
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
			responseData, _ = json.Marshal(response)
		} else {
			w.Header().Set("Content-Type", "application/json")
			responseData, _ = json.Marshal(filteredData)
		}
		
		w.Write(responseData)

		duration := time.Since(start)
		reqLog := RequestLog{
			Timestamp: start.Format(time.RFC3339),
			Method: r.Method,
			Path: r.URL.Path,
			Query: r.URL.RawQuery,
			StatusCode: statusCode,
			ResponseTime: duration.String(),
			UserAgent: r.Header.Get("User-Agent"),
			RemoteAddr: r.RemoteAddr,
			ContentLength: int64(len(responseData)),
			AuthType: authType,
			AuthResult: authResult,
		}
		logger.LogRequest(reqLog)
	}
}

func startServer(config *Config) ([]string, error) {
	logger, err := NewLogger(config.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %v", err)
	}

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
			http.HandleFunc(path, serveFileHandler(ep.File, ep, logger))
			continue
		}

		http.HandleFunc(path, createLoggingHandler(ep, logger))

	}

	if config.Logging.Enabled {
		logMsg := fmt.Sprintf("Logging: %s format", config.Logging.Format)
		if config.Logging.Output == "stdout" {
			logMsg += " to stdout"
		} else {
			logMsg += fmt.Sprintf(" to %s", config.Logging.Output)
		}
		messages = append(messages, logMsg)
	}

	go func() {
		log.Printf("Starting mock server on :%d\n", config.Port)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d",config.Port),nil))
	}()
		return messages, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var configPath string

	var rootCmd = &cobra.Command{
		Use: "apimocker",
		Short: "Lightweight TUI/mock REST API server with authentication and query parameter support",
		Long: `apimocker - A lightweight mock REST API server with TUI interface and authentication.

Supports dynamic query parameters:
 - count/limit: number of items to return
 - sort: field to sort by
 - order: asc/desc (default: asc)
 - filter: field:value to filter by
 - offset: number of items to skip
 - meta: include metadata in response (true/false)

Authentication types:
 - Basic Auth: username and password
 - Bearer Token: token-based authenticationn

Additional features:
 - Custom status codes
 - Response delays (ms, s, m or Go duration format)
 - Custom headers
 - Error simulation with probability

Example config:
port: 5050
logging:
  enabled: true
  format: json # or "plain"
  output: stdout # or file path like "requests.log"
endpoints:
  - path: /users
    method: GET
    status: 200
    delay: 500ms
    headers:
        X-Test-Mode: "true"
        X-API-Version: "v1"
	auth:
		type: bearer
		token: mysecrettoken
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
  - path: /admin
	method: GET
	auth:
		type: basic
		username: admin
		password: secret123
	data: |
		{
			"id": "uuid"
		}

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
			messages, err := startServer(config)
			if err != nil {
				log.Fatalf("Failed to start server: %v", err)
			}
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
