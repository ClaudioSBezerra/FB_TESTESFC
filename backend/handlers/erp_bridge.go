package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/sijms/go-ora/v2" // registro do driver "oracle"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type ERPBridgeConfig struct {
	CompanyID        string     `json:"company_id"`
	Ativo            bool       `json:"ativo"`
	Horario          string     `json:"horario"` // HH:MM
	DiasRetroativos  int        `json:"dias_retroativos"`
	UltimoRunEm      *time.Time `json:"ultimo_run_em"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ResetTracker     bool       `json:"reset_tracker"`
	ErpType          string     `json:"erp_type"`
	FBTaxEmail       string     `json:"fbtax_email"`
	FBTaxPasswordSet bool       `json:"fbtax_password_set"`
	OracleDsn        string     `json:"oracle_dsn"`
	OracleUsuario    string     `json:"oracle_usuario"`
	OracleSenhaSet   bool       `json:"oracle_senha_set"`
	APIKey           string     `json:"api_key"`
	DaemonLastSeen   *time.Time `json:"daemon_last_seen"`
	DaemonOnline     bool       `json:"daemon_online"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func erpBridgeGetCompany(db *sql.DB, r *http.Request) (string, error) {
	claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
	if !ok {
		return "", sql.ErrNoRows
	}
	userID := claims["user_id"].(string)
	return GetEffectiveCompanyID(db, userID, r.Header.Get("X-Company-ID"))
}

// ── GET/PATCH /api/erp-bridge/config ─────────────────────────────────────────

func ERPBridgeConfigHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		companyID, err := erpBridgeGetCompany(db, r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			var cfg ERPBridgeConfig
			var horario string
			var erpType, fbtaxEmail, fbtaxPassword, oracleDsn, oracleUsuario, oracleSenha, apiKey sql.NullString
			err := db.QueryRow(`
				SELECT company_id, ativo, TO_CHAR(horario, 'HH24:MI'), dias_retroativos,
				       ultimo_run_em, updated_at, reset_tracker,
				       COALESCE(erp_type, 'oracle_xml'),
				       fbtax_email, fbtax_password, oracle_dsn, oracle_usuario, oracle_senha, api_key,
				       daemon_last_seen
				FROM erp_bridge_config WHERE company_id = $1
			`, companyID).Scan(&cfg.CompanyID, &cfg.Ativo, &horario,
				&cfg.DiasRetroativos, &cfg.UltimoRunEm, &cfg.UpdatedAt, &cfg.ResetTracker,
				&erpType, &fbtaxEmail, &fbtaxPassword, &oracleDsn, &oracleUsuario, &oracleSenha, &apiKey,
				&cfg.DaemonLastSeen)
			if err == sql.ErrNoRows {
				cfg = ERPBridgeConfig{
					CompanyID:       companyID,
					Ativo:           false,
					Horario:         "02:00",
					DiasRetroativos: 1,
					UpdatedAt:       time.Now(),
					ResetTracker:    false,
					ErpType:         "oracle_xml",
				}
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else {
				cfg.Horario = horario
				if erpType.Valid {
					cfg.ErpType = erpType.String
				} else {
					cfg.ErpType = "oracle_xml"
				}
				if fbtaxEmail.Valid {
					cfg.FBTaxEmail = fbtaxEmail.String
				}
				cfg.FBTaxPasswordSet = fbtaxPassword.Valid && fbtaxPassword.String != ""
				if oracleDsn.Valid {
					cfg.OracleDsn = oracleDsn.String
				}
				if oracleUsuario.Valid {
					cfg.OracleUsuario = DecryptFieldWithFallback(oracleUsuario.String)
				}
				cfg.OracleSenhaSet = oracleSenha.Valid && oracleSenha.String != ""
				if apiKey.Valid && apiKey.String != "" {
					cfg.APIKey = DecryptFieldWithFallback(apiKey.String)
				}
				// Daemon está online se fez heartbeat nos últimos 3 minutos
				if cfg.DaemonLastSeen != nil {
					cfg.DaemonOnline = time.Since(*cfg.DaemonLastSeen) < 3*time.Minute
				}
			}
			json.NewEncoder(w).Encode(cfg)

		case http.MethodPatch:
			var req struct {
				Ativo           *bool   `json:"ativo"`
				Horario         *string `json:"horario"`
				DiasRetroativos *int    `json:"dias_retroativos"`
				ResetTracker    *bool   `json:"reset_tracker"`
				ErpType         *string `json:"erp_type"`
				FBTaxEmail      *string `json:"fbtax_email"`
				FBTaxPassword   *string `json:"fbtax_password"`
				OracleDsn       *string `json:"oracle_dsn"`
				OracleUsuario   *string `json:"oracle_usuario"`
				OracleSenha     *string `json:"oracle_senha"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "JSON inválido", http.StatusBadRequest)
				return
			}
			_, err := db.Exec(`
				INSERT INTO erp_bridge_config (company_id, ativo, horario, dias_retroativos, updated_at)
				VALUES ($1, COALESCE($2, false), COALESCE($3::TIME, '02:00'), COALESCE($4, 1), NOW())
				ON CONFLICT (company_id) DO UPDATE SET
				    ativo            = COALESCE($2, erp_bridge_config.ativo),
				    horario          = COALESCE($3::TIME, erp_bridge_config.horario),
				    dias_retroativos = COALESCE($4, erp_bridge_config.dias_retroativos),
				    reset_tracker    = COALESCE($5, erp_bridge_config.reset_tracker),
				    updated_at       = NOW()
			`, companyID, req.Ativo, req.Horario, req.DiasRetroativos, req.ResetTracker)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Atualiza credenciais individualmente se fornecidas
			if req.FBTaxEmail != nil {
				db.Exec(`UPDATE erp_bridge_config SET fbtax_email = $2 WHERE company_id = $1`, companyID, *req.FBTaxEmail)
			}
			if req.FBTaxPassword != nil && *req.FBTaxPassword != "" {
				if enc, encErr := EncryptField(*req.FBTaxPassword); encErr == nil {
					db.Exec(`UPDATE erp_bridge_config SET fbtax_password = $2 WHERE company_id = $1`, companyID, enc)
				}
			}
			if req.OracleUsuario != nil {
				if enc, encErr := EncryptField(*req.OracleUsuario); encErr == nil {
					db.Exec(`UPDATE erp_bridge_config SET oracle_usuario = $2 WHERE company_id = $1`, companyID, enc)
				}
			}
			if req.OracleSenha != nil && *req.OracleSenha != "" {
				if enc, encErr := EncryptField(*req.OracleSenha); encErr == nil {
					db.Exec(`UPDATE erp_bridge_config SET oracle_senha = $2 WHERE company_id = $1`, companyID, enc)
				}
			}
			if req.ErpType != nil {
				db.Exec(`UPDATE erp_bridge_config SET erp_type = $2 WHERE company_id = $1`, companyID, *req.ErpType)
			}
			if req.OracleDsn != nil {
				db.Exec(`UPDATE erp_bridge_config SET oracle_dsn = $2 WHERE company_id = $1`, companyID, *req.OracleDsn)
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ── POST /api/erp-bridge/config/generate-api-key ─────────────────────────────
// Gera uma nova API key para o daemon Bridge e a armazena criptografada.
// Retorna a chave em plaintext para copiar ao config.yaml — mostrada apenas uma vez.
// Restrito a admin (withAuth(..., "admin")).

func ERPBridgeGenerateAPIKeyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		companyID, err := erpBridgeGetCompany(db, r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Gera 32 bytes aleatórios → chave hex de 64 caracteres
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			http.Error(w, "erro ao gerar chave", http.StatusInternalServerError)
			return
		}
		key := hex.EncodeToString(raw)
		hash := sha256.Sum256([]byte(key))
		hashHex := hex.EncodeToString(hash[:])
		enc, encErr := EncryptField(key)
		if encErr != nil {
			http.Error(w, "erro ao criptografar chave", http.StatusInternalServerError)
			return
		}
		db.Exec(`
			INSERT INTO erp_bridge_config (company_id, api_key, api_key_hash)
			VALUES ($1, $2, $3)
			ON CONFLICT (company_id) DO UPDATE SET api_key = $2, api_key_hash = $3, updated_at = NOW()
		`, companyID, enc, hashHex)
		json.NewEncoder(w).Encode(map[string]string{"api_key": key})
	}
}

// ── GET /api/erp-bridge/credentials ──────────────────────────────────────────
// Endpoint público (sem JWT) — autenticado via X-API-Key.
// Usado pelo daemon Bridge para buscar credenciais criptografadas.

func ERPBridgeCredentialsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, "X-API-Key obrigatório", http.StatusUnauthorized)
			return
		}
		hash := sha256.Sum256([]byte(apiKey))
		hashHex := hex.EncodeToString(hash[:])
		var fbtaxEmail, fbtaxPassword, oracleUsuario, oracleSenha, erpType, oracleDsn sql.NullString
		err := db.QueryRow(`
			SELECT fbtax_email, fbtax_password, oracle_usuario, oracle_senha,
			       COALESCE(erp_type, 'oracle_xml'), COALESCE(oracle_dsn, '')
			FROM erp_bridge_config WHERE api_key_hash = $1
		`, hashHex).Scan(&fbtaxEmail, &fbtaxPassword, &oracleUsuario, &oracleSenha, &erpType, &oracleDsn)
		if err == sql.ErrNoRows {
			http.Error(w, "API key inválida", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result := map[string]string{
			"fbtax_email":    "",
			"fbtax_password": "",
			"oracle_usuario": "",
			"oracle_senha":   "",
			"erp_type":       "oracle_xml",
			"oracle_dsn":     "",
		}
		if fbtaxEmail.Valid {
			result["fbtax_email"] = fbtaxEmail.String
		}
		if fbtaxPassword.Valid && fbtaxPassword.String != "" {
			result["fbtax_password"] = DecryptFieldWithFallback(fbtaxPassword.String)
		}
		if oracleUsuario.Valid && oracleUsuario.String != "" {
			result["oracle_usuario"] = DecryptFieldWithFallback(oracleUsuario.String)
		}
		if oracleSenha.Valid && oracleSenha.String != "" {
			result["oracle_senha"] = DecryptFieldWithFallback(oracleSenha.String)
		}
		if erpType.Valid {
			result["erp_type"] = erpType.String
		}
		if oracleDsn.Valid && oracleDsn.String != "" {
			result["oracle_dsn"] = DecryptFieldWithFallback(oracleDsn.String)
		}
		json.NewEncoder(w).Encode(result)
	}
}

// ── POST /api/erp-bridge/test-connection ─────────────────────────────────────
// Abre e pinga a conexão Oracle via sijms/go-ora/v2.
// Exige autenticação (withAuth). Usa SOMENTE o DSN salvo em erp_bridge_config
// da empresa do usuário — não aceita host arbitrário no corpo (SSRF mitigado).
// Nunca retorna credenciais em claro ao frontend.

func ERPBridgeTestConnectionHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		companyID, err := erpBridgeGetCompany(db, r)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "empresa não encontrada"})
			return
		}

		// Ler credenciais Oracle armazenadas (criptografadas) em erp_bridge_config
		var oracleDsn, oracleUsuario, oracleSenha sql.NullString
		db.QueryRow(`
			SELECT oracle_dsn, oracle_usuario, oracle_senha
			FROM erp_bridge_config WHERE company_id = $1
		`, companyID).Scan(&oracleDsn, &oracleUsuario, &oracleSenha)

		if !oracleDsn.Valid || strings.TrimSpace(oracleDsn.String) == "" {
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "DSN Oracle não configurado"})
			return
		}

		// Descriptografar credenciais server-side; montar DSN go-ora.
		// Formato go-ora: "oracle://usuario:senha@host:port/service_name"
		// Ref: github.com/sijms/go-ora/v2 — suporta URL e Easy Connect.
		// Se o oracle_dsn já contém a URL completa ("oracle://..."), usá-lo diretamente.
		// Caso contrário (Easy Connect "host:port/service"), montar URL com usuario/senha.
		dsnPlain := DecryptFieldWithFallback(oracleDsn.String)
		usuarioPlain := DecryptFieldWithFallback(oracleUsuario.String)
		senhaPlain := DecryptFieldWithFallback(oracleSenha.String)

		var connStr string
		if strings.HasPrefix(dsnPlain, "oracle://") {
			// DSN já é uma URL Oracle completa — usar diretamente
			connStr = dsnPlain
		} else {
			// Easy Connect / host:port/service — montar URL
			connStr = fmt.Sprintf("oracle://%s:%s@%s",
				usuarioPlain,
				senhaPlain,
				dsnPlain,
			)
		}

		conn, err := sql.Open("oracle", connStr)
		if err != nil {
			log.Printf("ERPBridge test-connection sql.Open error (company %s): %v", companyID, err)
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if pingErr := conn.PingContext(ctx); pingErr != nil {
			log.Printf("ERPBridge test-connection ping error (company %s): %v", companyID, pingErr)
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": pingErr.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}
