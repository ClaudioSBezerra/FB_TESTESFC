package main

// FB_TESTESFC — Validador de Testes Unitários do Pacote Fiscal (Ferreira Costa)
// Backend Go enxuto: auth, hierarquia, ERP Bridge — sem workers SPED, sem Prometheus.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"fb_testesfc/handlers"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// ─── DB singleton ──────────────────────────────────────────────────────────────

var (
	db      *sql.DB
	dbMutex sync.RWMutex
	dbErr   error
)

func getDB() *sql.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return db
}

func initDBAsync() {
	go func() {
		var conn *sql.DB
		var err error
		connStr := os.Getenv("DATABASE_URL")
		if connStr == "" {
			connStr = "postgres://postgres:postgres@localhost:5432/fb_testesfc_db?sslmode=disable"
			fmt.Println("DATABASE_URL not set, using default local connection:", connStr)
		}

		attempt := 0
		for {
			attempt++
			conn, err = sql.Open("postgres", connStr)
			if err == nil {
				err = conn.Ping()
				if err == nil {
					conn.SetMaxOpenConns(25)
					conn.SetMaxIdleConns(10)
					conn.SetConnMaxLifetime(30 * time.Minute)

					dbMutex.Lock()
					db = conn
					dbErr = nil
					dbMutex.Unlock()

					fmt.Println("Successfully connected to the database!")
					onDBConnected()
					return
				}
			}

			dbMutex.Lock()
			dbErr = fmt.Errorf("attempt %d: %v", attempt, err)
			dbMutex.Unlock()

			fmt.Printf("Failed to connect to database (attempt %d): %v. Retrying in 5s...\n", attempt, err)
			time.Sleep(5 * time.Second)
		}
	}()
}

// ─── Migration runner ─────────────────────────────────────────────────────────

func onDBConnected() {
	database := getDB()

	migrationDir := "migrations"
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		if _, err := os.Stat("backend/migrations"); err == nil {
			migrationDir = "backend/migrations"
		}
	}

	fmt.Printf("Looking for migrations in: %s\n", migrationDir)
	files, err := filepath.Glob(filepath.Join(migrationDir, "*.sql"))
	if err != nil {
		log.Printf("Error finding migration files: %v", err)
		return
	}

	// Ensure schema_migrations table exists with correct schema
	var tableExists bool
	_ = database.QueryRow(`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='schema_migrations')`).Scan(&tableExists)
	if !tableExists {
		_, err = database.Exec(`CREATE TABLE schema_migrations (
			filename VARCHAR(255) PRIMARY KEY,
			executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			log.Printf("Warning: Failed to create schema_migrations table: %v", err)
		}
	}

	if len(files) == 0 {
		log.Println("Warning: No migration files found!")
	}

	for _, file := range files {
		baseName := filepath.Base(file)
		var alreadyExecuted bool
		errCheck := database.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)", baseName).Scan(&alreadyExecuted)
		if errCheck != nil {
			log.Printf("Warning: Could not check migration status for %s: %v", baseName, errCheck)
			continue
		}
		if alreadyExecuted {
			continue
		}

		fmt.Printf("Executing migration: %s\n", file)
		migration, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Could not read migration file %s: %v", file, err)
			continue
		}
		_, err = database.Exec(string(migration))
		if err != nil {
			log.Printf("ERROR: Migration %s failed: %v — will retry on next startup", file, err)
			// NÃO registrar migrações falhas — serão retentadas no próximo startup
			continue
		}
		fmt.Printf("Migration %s executed successfully.\n", file)
		_, insertErr := database.Exec("INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING", baseName)
		if insertErr != nil {
			log.Printf("Warning: Could not record migration %s: %v", baseName, insertErr)
		}
	}
	// PARAR AQUI — não há workers SPED, Prometheus ou goroutines de agendamento ERP/RFB neste projeto
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Database  string `json:"database"`
	DBError   string `json:"db_error,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dbStatus := "connecting..."
	var lastErr string

	database := getDB()
	if database != nil {
		if err := database.Ping(); err != nil {
			dbStatus = "error: " + err.Error()
		} else {
			dbStatus = "connected"
		}
	} else {
		dbMutex.RLock()
		if dbErr != nil {
			dbStatus = "error"
			lastErr = dbErr.Error()
		}
		dbMutex.RUnlock()
	}

	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "running",
		Timestamp: time.Now().Format(time.RFC3339),
		Service:   "FB_TESTESFC Validador Fiscal",
		Database:  dbStatus,
		DBError:   lastErr,
	})
}

func jsonServiceUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(`{"error":"service_unavailable","message":"Database initializing, please try again in a moment."}`))
}

// ─── Route helpers ─────────────────────────────────────────────────────────────

// withDB wraps a handler factory that needs a DB reference, delaying resolution until request time.
func withDB(handlerFactory func(*sql.DB) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		database := getDB()
		if database == nil {
			jsonServiceUnavailable(w)
			return
		}
		handlerFactory(database)(w, r)
	}
}

// withAuth wraps a handler factory with DB + JWT AuthMiddleware.
func withAuth(handlerFactory func(*sql.DB) http.HandlerFunc, role string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		database := getDB()
		if database == nil {
			jsonServiceUnavailable(w)
			return
		}
		h := handlerFactory(database)
		handlers.AuthMiddleware(h, role)(w, r)
	}
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	_ = godotenv.Load()

	// Validação JWT_SECRET — avisa em dev, fatal em prod
	handlers.ValidateJWTSecret()

	// Validação ENCRYPTION_KEY — avisa/fatal se ausente em prod (CR-02)
	handlers.ValidateEncryptionKey()

	initDBAsync()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	// ── Rotas de auth (sem autenticação) ──────────────────────────────────────
	http.HandleFunc("/api/auth/register", withDB(handlers.RegisterHandler))
	http.HandleFunc("/api/auth/login", withDB(handlers.LoginHandler))
	http.HandleFunc("/api/auth/forgot-password", withDB(handlers.ForgotPasswordHandler))
	http.HandleFunc("/api/auth/reset-password", withDB(handlers.ResetPasswordHandler))
	http.HandleFunc("/api/auth/refresh", withDB(handlers.RefreshHandler))
	http.HandleFunc("/api/auth/logout", withDB(handlers.LogoutHandler))

	// ── Rotas de auth (autenticadas) ──────────────────────────────────────────
	http.HandleFunc("/api/auth/me", withAuth(handlers.GetMeHandler, ""))
	http.HandleFunc("/api/auth/change-password", withAuth(handlers.ChangePasswordHandler, ""))
	http.HandleFunc("/api/auth/preferred-company", withAuth(handlers.SetPreferredCompanyHandler, ""))
	http.HandleFunc("/api/user/companies", withAuth(handlers.GetUserCompaniesHandler, ""))
	http.HandleFunc("/api/user/hierarchy", withAuth(handlers.GetUserHierarchyHandler, ""))

	// ── Rotas de admin (role=admin) ───────────────────────────────────────────
	http.HandleFunc("/api/admin/users", withAuth(handlers.ListUsersHandler, "admin"))
	http.HandleFunc("/api/admin/users/create", withAuth(handlers.CreateUserHandler, "admin"))
	http.HandleFunc("/api/admin/users/promote", withAuth(handlers.PromoteUserHandler, "admin"))
	http.HandleFunc("/api/admin/users/delete", withAuth(handlers.DeleteUserHandler, "admin"))
	http.HandleFunc("/api/admin/users/reassign", withAuth(handlers.ReassignUserHandler, "admin"))

	// ── Hierarquia ambiente/grupo/empresa (multi-method inline) ──────────────
	http.HandleFunc("/api/config/environments", withAuth(func(db *sql.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handlers.GetEnvironmentsHandler(db)(w, r)
			case http.MethodPost:
				handlers.CreateEnvironmentHandler(db)(w, r)
			case http.MethodPut:
				handlers.UpdateEnvironmentHandler(db)(w, r)
			case http.MethodDelete:
				handlers.DeleteEnvironmentHandler(db)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	}, ""))

	http.HandleFunc("/api/config/groups", withAuth(func(db *sql.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handlers.GetGroupsHandler(db)(w, r)
			case http.MethodPost:
				handlers.CreateGroupHandler(db)(w, r)
			case http.MethodPut:
				handlers.UpdateGroupHandler(db)(w, r)
			case http.MethodDelete:
				handlers.DeleteGroupHandler(db)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	}, ""))

	http.HandleFunc("/api/config/companies", withAuth(func(db *sql.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handlers.GetCompaniesHandler(db)(w, r)
			case http.MethodPost:
				handlers.CreateCompanyHandler(db)(w, r)
			case http.MethodPut:
				handlers.UpdateCompanyHandler(db)(w, r)
			case http.MethodDelete:
				handlers.DeleteCompanyHandler(db)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	}, ""))

	// ── Managers ──────────────────────────────────────────────────────────────
	http.HandleFunc("/api/managers", withAuth(handlers.ListManagersHandler, ""))
	http.HandleFunc("/api/managers/create", withAuth(handlers.CreateManagerHandler, ""))
	http.HandleFunc("/api/managers/", withAuth(func(db *sql.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut, http.MethodPatch:
				handlers.UpdateManagerHandler(db)(w, r)
			case http.MethodDelete:
				handlers.DeleteManagerHandler(db)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	}, ""))

	// ── ERP Bridge (Phase 1: config + credenciais + test-connection) ─────────
	// T-04-01: oracle_senha nunca retornada ao frontend (apenas *_set flag)
	// T-04-03: test-connection usa só o DSN salvo da empresa (não host do corpo) — SSRF mitigado
	// T-04-05: generate-api-key restrito a admin
	http.HandleFunc("/api/erp-bridge/config", withAuth(handlers.ERPBridgeConfigHandler, ""))
	http.HandleFunc("/api/erp-bridge/config/generate-api-key", withAuth(handlers.ERPBridgeGenerateAPIKeyHandler, "admin"))
	http.HandleFunc("/api/erp-bridge/test-connection", withAuth(handlers.ERPBridgeTestConnectionHandler, "")) // NOVO (D-14)
	// DESATIVADO (auditoria de segurança 2026-07-01, T-04-06): ERPBridgeCredentialsHandler
	// devolve credenciais Oracle em texto claro via X-API-Key, sem JWT e sem rate limit.
	// D-14 escopou a Fase 1 a "infra + testar conexão" — o daemon consumidor deste endpoint
	// ainda não existe. Reativar apenas quando uma fase futura construir esse daemon,
	// com rate limiting e revisão de ameaça dedicada.
	// http.HandleFunc("/api/erp-bridge/credentials", withDB(handlers.ERPBridgeCredentialsHandler))

	// ── Import Pipeline (Phase 2: XMLs de NF-e de saída) ─────────────────────
	// T-02-04: escopo por company_id resolvido via JWT (erpBridgeGetCompany) —
	// nenhum handler aceita company_id arbitrário do cliente.
	// T-02-05: todas as rotas exigem autenticação (withAuth, role vazia).
	http.HandleFunc("/api/xml/upload", withAuth(handlers.XMLUploadHandler, ""))
	http.HandleFunc("/api/nfe-saidas", withAuth(handlers.NFeSaidasListHandler, ""))
	http.HandleFunc("/api/nfe-saidas/", withAuth(handlers.NFeSaidaDetailHandler, ""))

	// ── Health ────────────────────────────────────────────────────────────────
	http.HandleFunc("/api/health", healthHandler)

	// ── SecurityMiddleware (CORS + security headers) envolvendo todo o mux ───
	// NÃO registrar /metrics (Prometheus removido — D-03)
	mux := handlers.SecurityMiddleware(http.DefaultServeMux, handlers.GetAllowedOrigins())

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("FB_TESTESFC backend iniciando na porta %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	<-quit
	log.Println("Encerrando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Erro no graceful shutdown: %v", err)
	}
	log.Println("Servidor encerrado.")
}
