package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

const liaraAPIBase = "https://api.iran.liara.ir"

type contextKey string

const liaraTokenContextKey contextKey = "liaraToken"

type Project struct {
	ID        string `json:"_id"`
	ProjectID string `json:"project_id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Scale     int    `json:"scale"`
	PlanID    string `json:"planID"`
	CreatedAt string `json:"created_at"`
}

type ProjectsResponse struct {
	Projects []Project `json:"projects"`
}

type Database struct {
	DBId          string `json:"DBId"`
	Type          string `json:"type"`
	PlanID        string `json:"planID"`
	Status        string `json:"status"`
	Scale         int    `json:"scale"`
	Hostname      string `json:"hostname"`
	PublicNetwork bool   `json:"publicNetwork"`
	Version       string `json:"version"`
	VolumeSize    int    `json:"volumeSize"`
	CreatedAt     string `json:"created_at"`
	DBName        string `json:"dbName"`
	Node          struct {
		ID   string `json:"_id"`
		Host string `json:"host"`
	} `json:"node"`
	Port         int    `json:"port"`
	RootPassword string `json:"root_password"`
	InternalPort int    `json:"internalPort"`
	ID           string `json:"id"`
	HourlyPrice  int    `json:"hourlyPrice"`
	MetaData     struct {
		StandaloneReplicaSet bool `json:"standaloneReplicaSet"`
		PrivateNetwork       bool `json:"privateNetwork"`
	} `json:"metaData"`
	Username string `json:"username"`
}

type DatabasesResponse struct {
	Databases []Database `json:"databases"`
}

type Schedule struct {
	ServiceName string       `json:"ServiceName"`
	ServiceType string       `json:"ServiceType"` // "project" or "database"
	Action      string       `json:"Action"`
	CronSpec    string       `json:"CronSpec"`
	JobID       cron.EntryID `json:"JobID"`
	NextRun     *time.Time   `json:"NextRun,omitempty"`
	LastRun     *time.Time   `json:"LastRun,omitempty"`
}

type SchedulesResponse struct {
	CurrentTime time.Time  `json:"currentTime"`
	Schedules   []Schedule `json:"schedules"`
}

var (
	scheduler = cron.New()
	schedules = make([]Schedule, 0)
	mu        sync.Mutex // For thread-safety for schedules

	// Log storage
	tokenLogs       = make(map[string]*bytes.Buffer)
	logsMu          sync.Mutex // For thread-safety for tokenLogs
	serverStartTime = time.Now()
)

// Custom log writer to store logs in memory per token
type TokenLogWriter struct {
	token string
}

func (writer *TokenLogWriter) Write(p []byte) (n int, err error) {
	logsMu.Lock()
	defer logsMu.Unlock()

	if _, ok := tokenLogs[writer.token]; !ok {
		tokenLogs[writer.token] = new(bytes.Buffer)
	}
	return tokenLogs[writer.token].Write(p)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		token := parts[1]

		ctx := context.WithValue(r.Context(), liaraTokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func getTokenFromContext(ctx context.Context) (string, error) {
	token, ok := ctx.Value(liaraTokenContextKey).(string)
	if !ok || token == "" {
		return "", fmt.Errorf("Liara token not found in context")
	}
	return token, nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

type LoginRequest struct {
	Token string `json:"token"`
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := getProjects(req.Token)
	if err != nil {
		log.Printf("Login failed for token: %v", err)
		http.Error(w, `{"error": "Invalid Liara API Token or API error"}`, http.StatusUnauthorized)
		return
	}

	// Set up log output for this specific token
	logsMu.Lock()
	tokenLogs[req.Token] = new(bytes.Buffer) // Clear previous logs for this token if any
	logsMu.Unlock()
	// This will direct subsequent logs for this token to its specific buffer
	// Note: This approach means the global log output is changed per request.
	// A more robust solution for production would involve passing a logger instance
	// through the context or using a more sophisticated logging library.
	// For this task, we'll use the simpler approach of setting the global output
	// within the handler, understanding its limitations.
	log.SetOutput(&TokenLogWriter{token: req.Token})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}

type ScheduleRequest struct {
	Service     string `json:"service"`
	ServiceType string `json:"serviceType"` // "project" or "database"
	Action      string `json:"action"`
	Cron        string `json:"cron"`
}

func scheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token, err := getTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error": "Authentication required"}`, http.StatusUnauthorized)
		return
	}

	var req ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	serviceName := req.Service
	serviceType := req.ServiceType
	action := req.Action
	cronSpec := req.Cron

	if serviceName == "" || (serviceType != "project" && serviceType != "database") || (action != "on" && action != "off") || cronSpec == "" {
		http.Error(w, `{"error": "Invalid input: service, serviceType, action, and cron are required"}`, http.StatusBadRequest)
		return
	}

	capturedToken := token

	jobID, err := scheduler.AddFunc(cronSpec, func() {
		if serviceType == "project" {
			scaleProject(serviceName, action == "on", capturedToken)
		} else if serviceType == "database" {
			scaleDatabase(serviceName, action == "on", capturedToken)
		}
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Invalid cron expression: %v"}`, err), http.StatusBadRequest)
		return
	}

	mu.Lock()
	schedules = append(schedules, Schedule{ServiceName: serviceName, ServiceType: serviceType, Action: action, CronSpec: cronSpec, JobID: jobID})
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Schedule added successfully"})
}

func scaleProject(projectName string, turnOn bool, token string) {
	scaleValue := 0
	if turnOn {
		scaleValue = 1
	}

	body := map[string]int{"scale": scaleValue}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/projects/%s/actions/scale", liaraAPIBase, projectName), bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("API error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API failed with status: %d", resp.StatusCode)
	} else {
		actionText := "turned off"
		if turnOn {
			actionText = "turned on"
		}
		log.Printf("Successfully %s project %s", actionText, projectName)
	}
}

func getProjects(token string) ([]Project, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/projects", liaraAPIBase), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API failed with status: %d", resp.StatusCode)
	}

	var projectsResponse ProjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&projectsResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return projectsResponse.Projects, nil
}

func projectsHandler(w http.ResponseWriter, r *http.Request) {
	token, err := getTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error": "Authentication required"}`, http.StatusUnauthorized)
		return
	}

	projects, err := getProjects(token)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch projects: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func getDatabases(token string) ([]Database, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/databases", liaraAPIBase), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API failed with status: %d", resp.StatusCode)
	}

	var databasesResponse DatabasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&databasesResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return databasesResponse.Databases, nil
}

func databasesHandler(w http.ResponseWriter, r *http.Request) {
	token, err := getTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error": "Authentication required"}`, http.StatusUnauthorized)
		return
	}

	databases, err := getDatabases(token)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch databases: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(databases)
}

func scaleDatabase(databaseID string, turnOn bool, token string) {
	scaleValue := 0
	if turnOn {
		scaleValue = 1
	}

	body := map[string]int{"scale": scaleValue}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/databases/%s/actions/scale", liaraAPIBase, databaseID), bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("API error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API failed with status: %d", resp.StatusCode)
	} else {
		actionText := "turned off"
		if turnOn {
			actionText = "turned on"
		}
		log.Printf("Successfully %s database %s", actionText, databaseID)
	}
}

func schedulesHandler(w http.ResponseWriter, r *http.Request) {
	_, err := getTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error": "Authentication required"}`, http.StatusUnauthorized)
		return
	}

	mu.Lock()
	currentSchedules := make([]Schedule, len(schedules))
	copy(currentSchedules, schedules)
	mu.Unlock()

	for i := range currentSchedules {
		entry := scheduler.Entry(currentSchedules[i].JobID)
		next := entry.Next
		prev := entry.Prev

		if !next.IsZero() {
			currentSchedules[i].NextRun = &next
		}
		if !prev.IsZero() {
			currentSchedules[i].LastRun = &prev
		}
	}

	response := SchedulesResponse{
		CurrentTime: time.Now(),
		Schedules:   currentSchedules,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 || pathParts[3] == "" {
		http.Error(w, `{"error": "Invalid request: JobID missing"}`, http.StatusBadRequest)
		return
	}

	jobIDStr := pathParts[3]
	id, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Invalid JobID format: %v"}`, err), http.StatusBadRequest)
		return
	}
	jobID := cron.EntryID(id)

	mu.Lock()
	defer mu.Unlock()

	scheduler.Remove(jobID)

	found := false
	for i, s := range schedules {
		if s.JobID == jobID {
			schedules = append(schedules[:i], schedules[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		http.Error(w, `{"error": "Schedule not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Schedule deleted successfully"})
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	token, err := getTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error": "Authentication required"}`, http.StatusUnauthorized)
		return
	}

	logsMu.Lock()
	defer logsMu.Unlock()

	logBuffer, ok := tokenLogs[token]
	if !ok {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("No logs available for this token."))
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(logBuffer.Bytes())
}

func uptimeHandler(w http.ResponseWriter, r *http.Request) {
	_, err := getTokenFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error": "Authentication required"}`, http.StatusUnauthorized)
		return
	}

	uptime := time.Since(serverStartTime)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"uptime": uptime.String()})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Set up a default log writer for general server logs
	log.SetOutput(os.Stdout)

	scheduler.Start()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/projects", authMiddleware(projectsHandler))
	http.HandleFunc("/databases", authMiddleware(databasesHandler))
	http.HandleFunc("/schedule", authMiddleware(scheduleHandler))
	http.HandleFunc("/schedules", authMiddleware(schedulesHandler))
	http.HandleFunc("/schedule/delete/", authMiddleware(deleteScheduleHandler))
	http.HandleFunc("/logs", authMiddleware(logsHandler))     // New endpoint for logs
	http.HandleFunc("/uptime", authMiddleware(uptimeHandler)) // New endpoint for uptime

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
