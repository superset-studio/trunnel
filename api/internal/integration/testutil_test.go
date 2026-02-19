//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/superset-studio/kapstan/api/internal/controllers"
	"github.com/superset-studio/kapstan/api/internal/platform/database"
)

// Shared state set by TestMain.
var (
	testDB            *sqlx.DB
	testRouter        *echo.Echo
	jwtSecret         = []byte("integration-test-secret-key-1234")
	testEncryptionKey = []byte("01234567890123456789012345678901") // 32 bytes
)

// Response types matching API JSON.

type authResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         userResponse `json:"user"`
	Organization orgResponse  `json:"organization"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type userResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	EmailVerified bool   `json:"emailVerified"`
}

type orgResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type memberResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	UserID         string `json:"userId"`
	Role           string `json:"role"`
	Email          string `json:"email,omitempty"`
	Name           string `json:"name,omitempty"`
}

type apiKeyCreateResponse struct {
	APIKey apiKeyResponse `json:"apiKey"`
	Key    string         `json:"key"`
}

type apiKeyResponse struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantId"`
	Name        string `json:"name"`
	KeyPrefix   string `json:"keyPrefix"`
	AccessLevel string `json:"accessLevel"`
}

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// migrationsDir returns the absolute path to api/migrations/.
func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = .../api/internal/integration/testutil_test.go
	apiDir := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(apiDir, "migrations")
}

// TestMain creates a temporary database, runs migrations, and sets up the router.
func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL not set — skipping integration tests")
		os.Exit(0)
	}

	// Connect to the management database to create/drop the temp DB.
	mgmtDB, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to management DB: %v\n", err)
		os.Exit(1)
	}

	tempDBName := fmt.Sprintf("kapstan_integ_%d", rand.IntN(900000)+100000)
	if _, err := mgmtDB.Exec(fmt.Sprintf("CREATE DATABASE %s", tempDBName)); err != nil {
		fmt.Fprintf(os.Stderr, "cannot create temp DB %s: %v\n", tempDBName, err)
		os.Exit(1)
	}

	// Build the connection string for the temp DB.
	tempDBURL := replaceDBName(dbURL, tempDBName)

	testDB, err = sqlx.Connect("pgx", tempDBURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to temp DB: %v\n", err)
		mgmtDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tempDBName))
		mgmtDB.Close()
		os.Exit(1)
	}

	// Run migrations.
	if err := database.RunMigrations(tempDBURL, migrationsDir()); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		testDB.Close()
		mgmtDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tempDBName))
		mgmtDB.Close()
		os.Exit(1)
	}

	// Build the HTTP router.
	testRouter = controllers.NewRouter(testDB, jwtSecret, testEncryptionKey)

	// Run tests.
	code := m.Run()

	// Cleanup.
	testDB.Close()
	mgmtDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tempDBName))
	mgmtDB.Close()

	os.Exit(code)
}

// replaceDBName swaps the database name in a PostgreSQL URL.
func replaceDBName(connStr, newDB string) string {
	qIdx := len(connStr)
	for i, c := range connStr {
		if c == '?' {
			qIdx = i
			break
		}
	}

	slashIdx := -1
	for i := qIdx - 1; i >= 0; i-- {
		if connStr[i] == '/' {
			slashIdx = i
			break
		}
	}

	if slashIdx == -1 {
		return connStr + "/" + newDB
	}

	return connStr[:slashIdx+1] + newDB + connStr[qIdx:]
}

// truncateTables clears all data between tests.
func truncateTables(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec("TRUNCATE connections, refresh_tokens, api_keys, organization_members, organizations, users CASCADE")
	require.NoError(t, err, "truncating tables")
}

// ---------- HTTP test client ----------

type testClient struct {
	token string
}

func newClient() *testClient {
	return &testClient{}
}

func (tc *testClient) withToken(token string) *testClient {
	return &testClient{token: token}
}

func (tc *testClient) do(method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if tc.token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+tc.token)
	}

	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	return rec
}

func (tc *testClient) get(path string) *httptest.ResponseRecorder {
	return tc.do(http.MethodGet, path, nil)
}

func (tc *testClient) post(path string, body interface{}) *httptest.ResponseRecorder {
	return tc.do(http.MethodPost, path, body)
}

func (tc *testClient) put(path string, body interface{}) *httptest.ResponseRecorder {
	return tc.do(http.MethodPut, path, body)
}

func (tc *testClient) delete(path string) *httptest.ResponseRecorder {
	return tc.do(http.MethodDelete, path, nil)
}

// ---------- High-level helpers ----------

// register creates a new user+org and returns the parsed auth response.
func register(t *testing.T, email, password, name, orgName string) authResponse {
	t.Helper()
	c := newClient()
	rec := c.post("/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
		"orgName":  orgName,
	})
	require.Equal(t, http.StatusCreated, rec.Code, "register %s: %s", email, rec.Body.String())

	var resp authResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

// login authenticates and returns the parsed auth response.
func login(t *testing.T, email, password string) authResponse {
	t.Helper()
	c := newClient()
	rec := c.post("/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	require.Equal(t, http.StatusOK, rec.Code, "login %s: %s", email, rec.Body.String())

	var resp authResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

// decode is a generic JSON decode helper.
func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
	return v
}
