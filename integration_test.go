package gotrycatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	trycatcherrors "github.com/linkerlin/gotrycatch/errors"
)

// =============================================================================
// HTTP Service Integration Tests
// =============================================================================

// TestHTTPServer_ValidationErrorHandling tests HTTP handlers with validation errors
func TestHTTPServer_ValidationErrorHandling(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var caught bool
		var responseCode int
		var responseBody map[string]interface{}

		tb := Try(func() {
			username := r.URL.Query().Get("username")
			if len(username) < 3 {
				panic(trycatcherrors.NewValidationError("username", "must be at least 3 characters", 1001))
			}
			if len(username) > 20 {
				panic(trycatcherrors.NewValidationError("username", "must be at most 20 characters", 1002))
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "username": username})
		})

		tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
			caught = true
			responseCode = http.StatusBadRequest
			responseBody = err.ToMap()
		})

		tb = tb.CatchAny(func(err interface{}) {
			caught = true
			responseCode = http.StatusInternalServerError
			responseBody = map[string]interface{}{"error": fmt.Sprintf("%v", err)}
		})

		tb.Finally(func() {
			if caught {
				w.WriteHeader(responseCode)
				json.NewEncoder(w).Encode(responseBody)
			}
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	tests := []struct {
		name           string
		username       string
		expectStatus   int
		expectError    bool
		expectErrorFld string
	}{
		{"valid username", "john", http.StatusOK, false, ""},
		{"too short", "ab", http.StatusBadRequest, true, "username"},
		{"too long", "thisusernameiswaytoolongexceedinglimit", http.StatusBadRequest, true, "username"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s?username=%s", server.URL, tt.username))
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			json.Unmarshal(body, &result)

			if tt.expectError {
				if result["field"] != tt.expectErrorFld {
					t.Errorf("Expected field '%s', got '%v'", tt.expectErrorFld, result["field"])
				}
			} else {
				if result["status"] != "success" {
					t.Errorf("Expected success, got %v", result)
				}
			}
		})
	}
}

// TestHTTPServer_AuthErrorHandling tests HTTP handlers with authentication errors
func TestHTTPServer_AuthErrorHandling(t *testing.T) {
	validToken := "valid-token-12345"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var responseCode int
		var responseBody map[string]interface{}

		tb := Try(func() {
			token := r.Header.Get("Authorization")
			if token == "" {
				panic(trycatcherrors.NewAuthError("token_verify", "anonymous", "missing token"))
			}
			if token != "Bearer "+validToken {
				panic(trycatcherrors.NewAuthError("token_verify", "unknown", "invalid token"))
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
		})

		tb = Catch[trycatcherrors.AuthError](tb, func(err trycatcherrors.AuthError) {
			responseCode = http.StatusUnauthorized
			responseBody = err.ToMap()
		})

		tb = tb.CatchAny(func(err interface{}) {
			responseCode = http.StatusInternalServerError
			responseBody = map[string]interface{}{"error": fmt.Sprintf("%v", err)}
		})

		tb.Finally(func() {
			if responseCode != 0 {
				w.WriteHeader(responseCode)
				json.NewEncoder(w).Encode(responseBody)
			}
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	tests := []struct {
		name         string
		token        string
		expectStatus int
	}{
		{"valid token", "Bearer " + validToken, http.StatusOK},
		{"invalid token", "Bearer invalid-token", http.StatusUnauthorized},
		{"no token", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", server.URL, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, resp.StatusCode)
			}
		})
	}
}

// TestHTTPServer_RateLimitErrorHandling tests HTTP handlers with rate limiting
func TestHTTPServer_RateLimitErrorHandling(t *testing.T) {
	var mu sync.Mutex
	requestCounts := make(map[string]int)
	rateLimit := 3

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.Header.Get("X-Client-ID")
		if clientID == "" {
			clientID = "anonymous"
		}

		var responseCode int
		var responseBody map[string]interface{}

		tb := Try(func() {
			mu.Lock()
			requestCounts[clientID]++
			count := requestCounts[clientID]
			mu.Unlock()

			if count > rateLimit {
				panic(trycatcherrors.NewRateLimitError("api", rateLimit, count, 60))
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"request": count,
			})
		})

		tb = Catch[trycatcherrors.RateLimitError](tb, func(err trycatcherrors.RateLimitError) {
			responseCode = http.StatusTooManyRequests
			responseBody = err.ToMap()
		})

		tb.Finally(func() {
			if responseCode != 0 {
				w.WriteHeader(responseCode)
				json.NewEncoder(w).Encode(responseBody)
			}
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{}

	for i := 1; i <= 5; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		req.Header.Set("X-Client-ID", "test-client")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		defer resp.Body.Close()

		if i <= rateLimit {
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Request %d: expected 200, got %d", i, resp.StatusCode)
			}
		} else {
			if resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Request %d: expected 429, got %d", i, resp.StatusCode)
			}
		}
	}
}

// TestHTTPServer_MultipleErrorTypes tests handlers handling multiple error types
func TestHTTPServer_MultipleErrorTypes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var responseCode int
		var responseBody map[string]interface{}

		tb := Try(func() {
			action := r.URL.Query().Get("action")

			switch action {
			case "validate":
				panic(trycatcherrors.NewValidationError("field", "invalid", 1000))
			case "auth":
				panic(trycatcherrors.NewAuthError("login", "user", "failed"))
			case "business":
				panic(trycatcherrors.NewBusinessLogicError("rule", "violation"))
			case "network":
				panic(trycatcherrors.NewNetworkError("http://example.com", 503))
			case "database":
				panic(trycatcherrors.NewDatabaseError("SELECT", "users", fmt.Errorf("connection failed")))
			case "success":
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				return
			}
		})

		tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
			responseCode = http.StatusBadRequest
			responseBody = err.ToMap()
		})

		tb = Catch[trycatcherrors.AuthError](tb, func(err trycatcherrors.AuthError) {
			responseCode = http.StatusUnauthorized
			responseBody = err.ToMap()
		})

		tb = Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
			responseCode = http.StatusUnprocessableEntity
			responseBody = err.ToMap()
		})

		tb = Catch[trycatcherrors.NetworkError](tb, func(err trycatcherrors.NetworkError) {
			responseCode = http.StatusBadGateway
			responseBody = err.ToMap()
		})

		tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
			responseCode = http.StatusServiceUnavailable
			responseBody = err.ToMap()
		})

		tb.Finally(func() {
			if responseCode != 0 {
				w.WriteHeader(responseCode)
				json.NewEncoder(w).Encode(responseBody)
			}
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	tests := []struct {
		action       string
		expectStatus int
		expectType   string
	}{
		{"success", http.StatusOK, ""},
		{"validate", http.StatusBadRequest, "ValidationError"},
		{"auth", http.StatusUnauthorized, "AuthError"},
		{"business", http.StatusUnprocessableEntity, "BusinessLogicError"},
		{"network", http.StatusBadGateway, "NetworkError"},
		{"database", http.StatusServiceUnavailable, "DatabaseError"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s?action=%s", server.URL, tt.action))
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, resp.StatusCode)
			}

			if tt.expectType != "" {
				body, _ := io.ReadAll(resp.Body)
				var result map[string]interface{}
				json.Unmarshal(body, &result)
				if result["type"] != tt.expectType {
					t.Errorf("Expected type %s, got %v", tt.expectType, result["type"])
				}
			}
		})
	}
}

// =============================================================================
// File Operation Integration Tests
// =============================================================================

// TestFileOperations_WriteWithCleanup tests file operations with cleanup in Finally
func TestFileOperations_WriteWithCleanup(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	var fileHandle *os.File
	var writeErr error

	tb := Try(func() {
		var err error
		fileHandle, err = os.Create(testFile)
		if err != nil {
			panic(trycatcherrors.NewConfigError("file", testFile, err.Error()))
		}

		_, writeErr = fileHandle.WriteString("Hello, World!")
		if writeErr != nil {
			panic(writeErr)
		}
	})

	tb = Catch[trycatcherrors.ConfigError](tb, func(err trycatcherrors.ConfigError) {
		t.Logf("Config error: %v", err)
	})

	tb.Finally(func() {
		if fileHandle != nil {
			fileHandle.Close()
		}
	})

	if writeErr != nil {
		t.Errorf("Write failed: %v", writeErr)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", content)
	}
}

// TestFileOperations_ReadNonExistent tests reading non-existent files
func TestFileOperations_ReadNonExistent(t *testing.T) {
	var caughtError trycatcherrors.ConfigError
	var caught bool

	tb := Try(func() {
		_, err := os.ReadFile("/nonexistent/path/to/file.txt")
		if err != nil {
			panic(trycatcherrors.NewConfigError("file", "/nonexistent/path/to/file.txt", err.Error()))
		}
	})

	tb = Catch[trycatcherrors.ConfigError](tb, func(err trycatcherrors.ConfigError) {
		caughtError = err
		caught = true
	})

	tb.Finally(func() {
	})

	if !caught {
		t.Error("Expected ConfigError to be caught")
	}

	if caughtError.Key != "file" {
		t.Errorf("Expected key 'file', got '%s'", caughtError.Key)
	}
}

// TestFileOperations_CreateDirectoryStructure tests directory creation with error handling
func TestFileOperations_CreateDirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()
	createdDirs := []string{}
	createdFiles := []string{}

	tb := Try(func() {
		dirs := []string{"src", "src/models", "src/controllers", "src/views"}
		for _, dir := range dirs {
			fullPath := filepath.Join(tempDir, dir)
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				panic(trycatcherrors.NewConfigError("directory", fullPath, err.Error()))
			}
			createdDirs = append(createdDirs, fullPath)
		}

		files := map[string]string{
			"src/main.go":        "package main",
			"src/models/user.go": "package models",
		}

		for filePath, content := range files {
			fullPath := filepath.Join(tempDir, filePath)
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				panic(trycatcherrors.NewConfigError("file", fullPath, err.Error()))
			}
			createdFiles = append(createdFiles, fullPath)
		}
	})

	tb = Catch[trycatcherrors.ConfigError](tb, func(err trycatcherrors.ConfigError) {
		t.Errorf("Config error: %v", err)
	})

	tb.Finally(func() {
		t.Logf("Created %d directories and %d files", len(createdDirs), len(createdFiles))
	})

	for _, dir := range createdDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory not created: %s", dir)
		}
	}

	for _, file := range createdFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("File not created: %s", file)
		}
	}
}

// TestFileOperations_TempFileCleanup tests temporary file cleanup in Finally
func TestFileOperations_TempFileCleanup(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "temp_data.txt")
	fileExists := true

	tb := Try(func() {
		if err := os.WriteFile(tempFile, []byte("temporary data"), 0644); err != nil {
			panic(err)
		}

		content, err := os.ReadFile(tempFile)
		if err != nil {
			panic(err)
		}

		if string(content) != "temporary data" {
			panic(fmt.Errorf("unexpected content: %s", content))
		}

		panic(trycatcherrors.NewBusinessLogicError("temp_check", "intentional error to test cleanup"))
	})

	tb = Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
	})

	tb.Finally(func() {
		if err := os.Remove(tempFile); err == nil {
			fileExists = false
		}
	})

	if fileExists {
		t.Error("Temp file should have been cleaned up")
	}
}

// =============================================================================
// Database Simulation Integration Tests (No Mocking)
// =============================================================================

// In-memory database simulation
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type InMemoryDB struct {
	users  map[int]*User
	nextID int
	mu     sync.RWMutex
}

func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		users:  make(map[int]*User),
		nextID: 1,
	}
}

func (db *InMemoryDB) Insert(user *User) (*User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if user.Name == "" {
		return nil, trycatcherrors.NewValidationError("name", "name is required", 2001)
	}
	if user.Email == "" {
		return nil, trycatcherrors.NewValidationError("email", "email is required", 2002)
	}

	user.ID = db.nextID
	db.users[user.ID] = user
	db.nextID++
	return user, nil
}

func (db *InMemoryDB) FindByID(id int) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user, exists := db.users[id]
	if !exists {
		return nil, trycatcherrors.NewDatabaseError("SELECT", "users", fmt.Errorf("user not found with id %d", id))
	}
	return user, nil
}

func (db *InMemoryDB) Update(user *User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.users[user.ID]; !exists {
		return trycatcherrors.NewDatabaseError("UPDATE", "users", fmt.Errorf("user not found with id %d", user.ID))
	}
	db.users[user.ID] = user
	return nil
}

func (db *InMemoryDB) Delete(id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.users[id]; !exists {
		return trycatcherrors.NewDatabaseError("DELETE", "users", fmt.Errorf("user not found with id %d", id))
	}
	delete(db.users, id)
	return nil
}

// TestDatabaseSimulation_CRUDOperations tests CRUD operations with Try-Catch
func TestDatabaseSimulation_CRUDOperations(t *testing.T) {
	db := NewInMemoryDB()

	var createdUser *User
	var foundUserName string
	var updateErr error
	var deleted bool

	tb := Try(func() {
		var err error
		createdUser, err = db.Insert(&User{Name: "John Doe", Email: "john@example.com"})
		if err != nil {
			panic(err)
		}

		foundUser, err := db.FindByID(createdUser.ID)
		if err != nil {
			panic(err)
		}
		foundUserName = foundUser.Name

		foundUser.Name = "John Updated"
		updateErr = db.Update(foundUser)
		if updateErr != nil {
			panic(updateErr)
		}

		if err := db.Delete(createdUser.ID); err != nil {
			panic(err)
		}
		deleted = true
	})

	tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		t.Errorf("Validation error: %v", err)
	})

	tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		t.Errorf("Database error: %v", err)
	})

	tb.Finally(func() {
		t.Logf("User ID %d operations completed", createdUser.ID)
	})

	if createdUser == nil || createdUser.ID != 1 {
		t.Errorf("User creation failed")
	}
	if foundUserName != "John Doe" {
		t.Errorf("User find failed, expected 'John Doe', got '%s'", foundUserName)
	}
	if updateErr != nil {
		t.Errorf("User update failed")
	}
	if !deleted {
		t.Errorf("User deletion failed")
	}
}

// TestDatabaseSimulation_ValidationErrors tests validation error handling
func TestDatabaseSimulation_ValidationErrors(t *testing.T) {
	db := NewInMemoryDB()
	var caughtValidationError trycatcherrors.ValidationError
	var caught bool

	tb := Try(func() {
		_, err := db.Insert(&User{Name: "", Email: "test@example.com"})
		if err != nil {
			panic(err)
		}
	})

	tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		caughtValidationError = err
		caught = true
	})

	tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		t.Errorf("Unexpected database error: %v", err)
	})

	if !caught {
		t.Error("Expected ValidationError to be caught")
	}
	if caughtValidationError.Field != "name" {
		t.Errorf("Expected field 'name', got '%s'", caughtValidationError.Field)
	}
}

// TestDatabaseSimulation_NotFoundErrors tests not found error handling
func TestDatabaseSimulation_NotFoundErrors(t *testing.T) {
	db := NewInMemoryDB()
	var caughtDatabaseError trycatcherrors.DatabaseError
	var caught bool

	tb := Try(func() {
		_, err := db.FindByID(999)
		if err != nil {
			panic(err)
		}
	})

	tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		caughtDatabaseError = err
		caught = true
	})

	if !caught {
		t.Error("Expected DatabaseError to be caught")
	}
	if caughtDatabaseError.Operation != "SELECT" {
		t.Errorf("Expected operation 'SELECT', got '%s'", caughtDatabaseError.Operation)
	}
}

// =============================================================================
// Multi-Layer Application Integration Tests
// =============================================================================

// Repository layer
type UserRepository struct {
	db *InMemoryDB
}

func (r *UserRepository) Create(user *User) (*User, error) {
	return r.db.Insert(user)
}

func (r *UserRepository) GetByID(id int) (*User, error) {
	return r.db.FindByID(id)
}

// Service layer
type UserService struct {
	repo *UserRepository
}

func (s *UserService) CreateUser(name, email string) (*User, error) {
	if len(name) < 2 {
		return nil, trycatcherrors.NewValidationError("name", "name must be at least 2 characters", 3001)
	}

	user, err := s.repo.Create(&User{Name: name, Email: email})
	if err != nil {
		return nil, trycatcherrors.NewDatabaseError("INSERT", "users", err)
	}

	return user, nil
}

func (s *UserService) GetUser(id int) (*User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, trycatcherrors.NewDatabaseError("SELECT", "users", err)
	}

	if user.Email == "" {
		return nil, trycatcherrors.NewBusinessLogicError("email_required", "user must have valid email")
	}

	return user, nil
}

// Controller layer
type UserController struct {
	service *UserService
}

func (c *UserController) HandleCreateUser(name, email string) (int, map[string]interface{}) {
	var responseCode int
	var responseBody map[string]interface{}

	tb := Try(func() {
		user, err := c.service.CreateUser(name, email)
		if err != nil {
			panic(err)
		}
		responseCode = http.StatusCreated
		responseBody = map[string]interface{}{
			"status": "created",
			"user":   user,
		}
	})

	tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		responseCode = http.StatusBadRequest
		responseBody = err.ToMap()
	})

	tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		responseCode = http.StatusInternalServerError
		responseBody = err.ToMap()
	})

	tb.Finally(func() {
	})

	return responseCode, responseBody
}

func (c *UserController) HandleGetUser(id int) (int, map[string]interface{}) {
	var responseCode int
	var responseBody map[string]interface{}

	tb := Try(func() {
		user, err := c.service.GetUser(id)
		if err != nil {
			panic(err)
		}
		responseCode = http.StatusOK
		responseBody = map[string]interface{}{
			"status": "success",
			"user":   user,
		}
	})

	tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		responseCode = http.StatusNotFound
		responseBody = err.ToMap()
	})

	tb = Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
		responseCode = http.StatusUnprocessableEntity
		responseBody = err.ToMap()
	})

	return responseCode, responseBody
}

// TestMultiLayerApplication_CreateUser tests the full Controller -> Service -> Repository chain
func TestMultiLayerApplication_CreateUser(t *testing.T) {
	db := NewInMemoryDB()
	repo := &UserRepository{db: db}
	service := &UserService{repo: repo}
	controller := &UserController{service: service}

	tests := []struct {
		name         string
		userName     string
		email        string
		expectStatus int
	}{
		{"valid user", "John Doe", "john@example.com", http.StatusCreated},
		{"name too short", "J", "j@example.com", http.StatusBadRequest},
		{"empty name", "", "empty@example.com", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, body := controller.HandleCreateUser(tt.userName, tt.email)

			if code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, code)
			}

			if tt.expectStatus == http.StatusCreated {
				if body["status"] != "created" {
					t.Errorf("Expected status 'created', got %v", body["status"])
				}
			}
		})
	}
}

// TestMultiLayerApplication_GetUser tests user retrieval through all layers
func TestMultiLayerApplication_GetUser(t *testing.T) {
	db := NewInMemoryDB()
	repo := &UserRepository{db: db}
	service := &UserService{repo: repo}
	controller := &UserController{service: service}

	var createdID int
	tb := Try(func() {
		user, err := db.Insert(&User{Name: "Test User", Email: "test@example.com"})
		if err != nil {
			panic(err)
		}
		createdID = user.ID
	})
	tb.Finally(func() {})

	t.Run("get existing user", func(t *testing.T) {
		code, body := controller.HandleGetUser(createdID)

		if code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, code)
		}

		if body["status"] != "success" {
			t.Errorf("Expected status 'success', got %v", body["status"])
		}
	})

	t.Run("get non-existent user", func(t *testing.T) {
		code, body := controller.HandleGetUser(9999)

		if code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, code)
		}

		if body["type"] != "DatabaseError" {
			t.Errorf("Expected type 'DatabaseError', got %v", body["type"])
		}
	})
}

// =============================================================================
// Error Propagation Integration Tests
// =============================================================================

// TestErrorPropagation_ThroughMultipleLayers tests error propagation through layers
func TestErrorPropagation_ThroughMultipleLayers(t *testing.T) {
	db := NewInMemoryDB()

	var propagatedError interface{}
	var errorType string

	tb := Try(func() {
		_, err := db.Insert(&User{Name: "", Email: "test@example.com"})
		if err != nil {
			panic(err)
		}
	})

	tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
		propagatedError = err
		errorType = "ValidationError"
	})

	tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		propagatedError = err
		errorType = "DatabaseError"
	})

	tb.Finally(func() {})

	if propagatedError == nil {
		t.Error("Expected error to be propagated")
	}

	if errorType != "ValidationError" {
		t.Errorf("Expected ValidationError, got %s", errorType)
	}
}

// TestErrorPropagation_NestedTryCatch tests nested try-catch blocks
func TestErrorPropagation_NestedTryCatch(t *testing.T) {
	var outerCaught bool
	var innerCaught bool
	var finallyCalled bool

	tb := Try(func() {
		tbInner := Try(func() {
			panic(trycatcherrors.NewValidationError("inner", "inner error", 1))
		})

		tbInner = Catch[trycatcherrors.ValidationError](tbInner, func(err trycatcherrors.ValidationError) {
			innerCaught = true
			panic(trycatcherrors.NewBusinessLogicError("outer", "propagated from inner"))
		})

		tbInner.Finally(func() {})
	})

	tb = Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
		outerCaught = true
	})

	tb.Finally(func() {
		finallyCalled = true
	})

	if !innerCaught {
		t.Error("Inner catch not called")
	}
	if !outerCaught {
		t.Error("Outer catch not called")
	}
	if !finallyCalled {
		t.Error("Finally not called")
	}
}

// =============================================================================
// Concurrent Integration Tests
// =============================================================================

// TestConcurrent_HTTPRequests tests concurrent HTTP request handling
func TestConcurrent_HTTPRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var responseCode int

		tb := Try(func() {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

		tb = tb.CatchAny(func(err interface{}) {
			responseCode = http.StatusInternalServerError
		})

		tb.Finally(func() {
			if responseCode != 0 {
				w.WriteHeader(responseCode)
			}
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := http.Get(server.URL)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("request %d: unexpected status %d", id, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent request error: %v", err)
	}
}

// TestConcurrent_DatabaseOperations tests concurrent database operations
func TestConcurrent_DatabaseOperations(t *testing.T) {
	db := NewInMemoryDB()
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errorCount := 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			tb := Try(func() {
				_, err := db.Insert(&User{
					Name:  fmt.Sprintf("User%d", id),
					Email: fmt.Sprintf("user%d@example.com", id),
				})
				if err != nil {
					panic(err)
				}
			})

			tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
				mu.Lock()
				errorCount++
				mu.Unlock()
			})

			tb.Finally(func() {
				if !tb.HasError() {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			})
		}(i)
	}

	wg.Wait()

	if successCount != 100 {
		t.Errorf("Expected 100 successful inserts, got %d", successCount)
	}
	if errorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}
}

// =============================================================================
// TryWithResult Integration Tests
// =============================================================================

// TestTryWithResult_HTTPClient tests TryWithResult with HTTP client
func TestTryWithResult_HTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "hello"}`))
	}))
	defer server.Close()

	var result string
	var caught bool

	tb := TryWithResult(func() string {
		resp, err := http.Get(server.URL)
		if err != nil {
			panic(trycatcherrors.NewNetworkError(server.URL, 0))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			panic(trycatcherrors.NewNetworkError(server.URL, resp.StatusCode))
		}

		body, _ := io.ReadAll(resp.Body)
		return string(body)
	})

	tb = CatchWithResult[string, trycatcherrors.NetworkError](tb, func(err trycatcherrors.NetworkError) {
		caught = true
	})

	tb.Finally(func() {})

	if caught {
		t.Error("Should not have caught an error")
	}

	result = tb.GetResult()
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

// TestTryWithResult_FileReading tests TryWithResult with file reading
func TestTryWithResult_FileReading(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "data.txt")
	os.WriteFile(testFile, []byte("file content"), 0644)

	var result string
	var caughtError trycatcherrors.ConfigError

	tb := TryWithResult(func() string {
		content, err := os.ReadFile(testFile)
		if err != nil {
			panic(trycatcherrors.NewConfigError("file", testFile, err.Error()))
		}
		return string(content)
	})

	tb = CatchWithResult[string, trycatcherrors.ConfigError](tb, func(err trycatcherrors.ConfigError) {
		caughtError = err
	})

	tb.Finally(func() {})

	if caughtError.Key != "" {
		t.Errorf("Should not have error, got: %v", caughtError)
	}

	result = tb.GetResult()
	if result != "file content" {
		t.Errorf("Expected 'file content', got '%s'", result)
	}
}

// TestTryWithResult_DatabaseQuery tests TryWithResult with database query
func TestTryWithResult_DatabaseQuery(t *testing.T) {
	db := NewInMemoryDB()
	db.Insert(&User{Name: "Alice", Email: "alice@example.com"})
	db.Insert(&User{Name: "Bob", Email: "bob@example.com"})

	var users []*User
	var caught bool

	tb := TryWithResult(func() []*User {
		var result []*User
		for i := 1; i <= 2; i++ {
			user, err := db.FindByID(i)
			if err != nil {
				panic(err)
			}
			result = append(result, user)
		}
		return result
	})

	tb = CatchWithResult[[]*User, trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
		caught = true
	})

	tb.Finally(func() {})

	if caught {
		t.Error("Should not have caught an error")
	}

	users = tb.GetResult()
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

// =============================================================================
// Real-World Scenario Tests
// =============================================================================

// TestRealWorld_APIEndpoint tests a complete API endpoint scenario
func TestRealWorld_APIEndpoint(t *testing.T) {
	db := NewInMemoryDB()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var responseCode int
		var responseBody map[string]interface{}
		var startTime = time.Now()

		tb := Try(func() {
			if r.Method != http.MethodPost {
				panic(trycatcherrors.NewValidationError("method", "only POST allowed", 4000))
			}

			var input struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}

			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				panic(trycatcherrors.NewValidationError("body", "invalid JSON", 4001))
			}

			user, err := db.Insert(&User{Name: input.Name, Email: input.Email})
			if err != nil {
				panic(err)
			}

			responseCode = http.StatusCreated
			responseBody = map[string]interface{}{
				"status": "created",
				"user":   user,
				"meta": map[string]interface{}{
					"duration_ms": time.Since(startTime).Milliseconds(),
				},
			}
		})

		tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
			responseCode = http.StatusBadRequest
			responseBody = err.ToMap()
		})

		tb = Catch[trycatcherrors.DatabaseError](tb, func(err trycatcherrors.DatabaseError) {
			responseCode = http.StatusInternalServerError
			responseBody = map[string]interface{}{
				"type":  "DatabaseError",
				"error": err.Error(),
			}
		})

		tb = tb.CatchAny(func(err interface{}) {
			responseCode = http.StatusInternalServerError
			responseBody = map[string]interface{}{
				"type":  "UnknownError",
				"error": fmt.Sprintf("%v", err),
			}
		})

		tb.Finally(func() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(responseCode)
			json.NewEncoder(w).Encode(responseBody)
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("valid user creation", func(t *testing.T) {
		resp, err := http.Post(server.URL, "application/json",
			bytes.NewReader([]byte(`{"name":"John","email":"john@example.com"}`)))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		resp, err := http.Post(server.URL, "application/json",
			bytes.NewReader([]byte(`invalid json`)))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})
}

// TestRealWorld_BatchProcessing tests batch processing with error recovery
func TestRealWorld_BatchProcessing(t *testing.T) {
	db := NewInMemoryDB()

	users := []struct {
		Name  string
		Email string
	}{
		{"User1", "user1@example.com"},
		{"", "user2@example.com"},
		{"User3", "user3@example.com"},
		{"User4", ""},
		{"User5", "user5@example.com"},
	}

	var successCount int
	var validationErrors []trycatcherrors.ValidationError
	var mu sync.Mutex

	for _, u := range users {
		tb := Try(func() {
			_, err := db.Insert(&User{Name: u.Name, Email: u.Email})
			if err != nil {
				panic(err)
			}
		})

		tb = Catch[trycatcherrors.ValidationError](tb, func(err trycatcherrors.ValidationError) {
			mu.Lock()
			validationErrors = append(validationErrors, err)
			mu.Unlock()
		})

		tb.Finally(func() {
			if !tb.HasError() {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		})
	}

	if successCount != 3 {
		t.Errorf("Expected 3 successful inserts, got %d", successCount)
	}

	if len(validationErrors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(validationErrors))
	}
}

// TestRealWorld_GracefulShutdown tests graceful shutdown scenario
func TestRealWorld_GracefulShutdown(t *testing.T) {
	var resources []string
	var cleanedUp []string
	var mu sync.Mutex

	cleanupResource := func(name string) {
		mu.Lock()
		cleanedUp = append(cleanedUp, name)
		mu.Unlock()
	}

	tb := Try(func() {
		resources = append(resources, "db_connection")
		resources = append(resources, "cache_client")
		resources = append(resources, "file_handle")

		panic(trycatcherrors.NewBusinessLogicError("shutdown", "simulated shutdown signal"))
	})

	tb = Catch[trycatcherrors.BusinessLogicError](tb, func(err trycatcherrors.BusinessLogicError) {
	})

	tb.Finally(func() {
		for i := len(resources) - 1; i >= 0; i-- {
			cleanupResource(resources[i])
		}
	})

	expectedOrder := []string{"file_handle", "cache_client", "db_connection"}
	for i, expected := range expectedOrder {
		if cleanedUp[i] != expected {
			t.Errorf("Cleanup order wrong: expected %s at position %d, got %s", expected, i, cleanedUp[i])
		}
	}
}
