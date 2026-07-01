package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"fb_testesfc/services"
)

// ---------------------------------------------------------------------------
// Defaults de parâmetros do pacote fiscal sem fonte de dados persistida.
//
// ASSUNÇÕES PENDENTES DE VALIDAÇÃO NO CHECKPOINT HUMANO (02-02-PLAN.md):
// o XML de NF-e / schema atual do FB_TESTESFC não carrega alguns dos 23
// parâmetros de calcula_imposto_produto (ex.: indIEDest do destinatário,
// CRT/Simples Nacional do emitente, código interno de "centro fiscal"). Os
// valores abaixo são defaults conservadores documentados — a Fase 3
// (comparação esperado-vs-calculado) e o checkpoint humano desta fase vão
// expor rapidamente se algum default estiver incorreto para a Ferreira Costa.
// ---------------------------------------------------------------------------
const (
	defaultTipoContribuinte          = "N"     // não contribuinte — sem indIEDest persistido para refinar
	defaultTipoCentroFiscal          = "VRJNE" // valor do exemplo do script de teste do pacote fiscal (constante de config, não vem do XML)
	defaultTipoOperacao              = 1       // 1 = operação normal de venda (script de teste)
	defaultIndicadorServico          = "N"     // comércio, não serviço
	defaultFornecedorSimplesNacional = "N"     // CRT do emitente não persistido em nfe_saidas
)

// fiscalNotaContext agrega os dados de cabeçalho da nota necessários para
// montar o FiscalInput de cada item + o cod_empresa resolvido uma única vez
// por nota (T-02-08: nunca aceito do cliente, sempre derivado do CNPJ/UF do
// próprio XML já persistido e escopado por company_id).
type fiscalNotaContext struct {
	EmitCNPJ      string
	EmitUF        string
	DestUF        string
	DestCMun      string
	DataEmissao   time.Time
	CodEmpresa    int
	CodEmpresaErr error
}

type fiscalItemInput struct {
	ID    string
	CProd string
	CFOP  string
	VProd float64
	VDesc float64
	VIPI  float64
}

type fiscalExecutionSummary struct {
	Total          int `json:"total"`
	OK             int `json:"ok"`
	SemGrupoFiscal int `json:"sem_grupo_fiscal"`
	Error          int `json:"error"`
}

// ---------------------------------------------------------------------------
// POST /api/fiscal-execution/run
// Body: {"nfe_id": "<uuid>"}
// ---------------------------------------------------------------------------

func FiscalExecutionRunHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			jsonErr(w, http.StatusMethodNotAllowed, "Método não permitido")
			return
		}

		// T-02-08: company sempre resolvida via JWT — nunca aceita do corpo.
		companyID, err := erpBridgeGetCompany(db, r)
		if err != nil {
			jsonErr(w, http.StatusUnauthorized, "Não autenticado")
			return
		}

		var req struct {
			NfeID string `json:"nfe_id"`
		}
		if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil || strings.TrimSpace(req.NfeID) == "" {
			jsonErr(w, http.StatusBadRequest, "nfe_id é obrigatório")
			return
		}

		// Valida que a nota pertence à company do JWT (T-02-08) e carrega o
		// cabeçalho necessário para montar os parâmetros de entrada.
		var emitCNPJ, emitUF, destUF, destCMun string
		var dataEmissao time.Time
		err = db.QueryRow(`
			SELECT emit_cnpj, COALESCE(emit_uf,''), COALESCE(dest_uf,''), COALESCE(dest_c_mun,''), data_emissao
			FROM nfe_saidas
			WHERE id = $1 AND company_id = $2`, req.NfeID, companyID,
		).Scan(&emitCNPJ, &emitUF, &destUF, &destCMun, &dataEmissao)
		if err == sql.ErrNoRows {
			jsonErr(w, http.StatusNotFound, "Nota não encontrada")
			return
		}
		if err != nil {
			log.Printf("[FiscalExecution] load nfe error (nfe=%s): %v", req.NfeID, err)
			jsonErr(w, http.StatusInternalServerError, "Erro ao carregar nota")
			return
		}

		nfeCtx := fiscalNotaContext{
			EmitCNPJ:    emitCNPJ,
			EmitUF:      emitUF,
			DestUF:      destUF,
			DestCMun:    destCMun,
			DataEmissao: dataEmissao,
		}
		nfeCtx.CodEmpresa, nfeCtx.CodEmpresaErr = resolveCodEmpresa(emitCNPJ, emitUF)

		itemRows, err := db.Query(`
			SELECT id, COALESCE(c_prod,''), COALESCE(cfop,''), v_prod, COALESCE(v_desc,0), v_ipi
			FROM nfe_saidas_itens
			WHERE nfe_id = $1
			ORDER BY n_item ASC`, req.NfeID)
		if err != nil {
			log.Printf("[FiscalExecution] load itens error (nfe=%s): %v", req.NfeID, err)
			jsonErr(w, http.StatusInternalServerError, "Erro ao carregar itens da nota")
			return
		}
		var itens []fiscalItemInput
		for itemRows.Next() {
			var it fiscalItemInput
			if scanErr := itemRows.Scan(&it.ID, &it.CProd, &it.CFOP, &it.VProd, &it.VDesc, &it.VIPI); scanErr != nil {
				log.Printf("[FiscalExecution] item scan error (nfe=%s): %v", req.NfeID, scanErr)
				continue
			}
			itens = append(itens, it)
		}
		itemRows.Close()

		if len(itens) == 0 {
			json.NewEncoder(w).Encode(fiscalExecutionSummary{})
			return
		}

		// Conexão Oracle dedicada a este pipeline (Pitfall 5 — SetMaxOpenConns
		// distinto da conexão pontual de test-connection).
		oracleConn, err := openFiscalOracleConn(db, companyID)
		if err != nil {
			log.Printf("[FiscalExecution] openFiscalOracleConn error (company=%s): %v", companyID, err)
			jsonErr(w, http.StatusBadGateway, "Falha ao conectar ao Oracle. Verifique as credenciais ERP configuradas.")
			return
		}
		defer oracleConn.Close()

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()

		summary := processFiscalBatch(ctx, oracleConn, db, companyID, nfeCtx, itens)
		json.NewEncoder(w).Encode(summary)
	}
}

// openFiscalOracleConn monta a conexão Oracle a partir das credenciais salvas
// em erp_bridge_config (mesmo padrão de erp_bridge.go), com SetMaxOpenConns(5)
// dedicado a este pipeline (Pitfall 5 — exaustão de sessões Oracle, T-02-07).
func openFiscalOracleConn(db *sql.DB, companyID string) (*sql.DB, error) {
	var oracleDsn, oracleUsuario, oracleSenha sql.NullString
	err := db.QueryRow(`
		SELECT oracle_dsn, oracle_usuario, oracle_senha
		FROM erp_bridge_config WHERE company_id = $1`, companyID,
	).Scan(&oracleDsn, &oracleUsuario, &oracleSenha)
	if err != nil {
		return nil, fmt.Errorf("credenciais Oracle não configuradas para a empresa")
	}
	if !oracleDsn.Valid || strings.TrimSpace(oracleDsn.String) == "" {
		return nil, fmt.Errorf("DSN Oracle não configurado")
	}

	dsnPlain := DecryptFieldWithFallback(oracleDsn.String)
	usuarioPlain := DecryptFieldWithFallback(oracleUsuario.String)
	senhaPlain := DecryptFieldWithFallback(oracleSenha.String)

	var connStr string
	if strings.HasPrefix(dsnPlain, "oracle://") {
		connStr = dsnPlain
	} else {
		connStr = fmt.Sprintf("oracle://%s:%s@%s", usuarioPlain, senhaPlain, dsnPlain)
	}

	conn, err := sql.Open("oracle", connStr)
	if err != nil {
		// Nunca propagar err.Error() ao cliente — pode conter DSN/senha (T-02-06).
		return nil, fmt.Errorf("falha ao inicializar conexão Oracle")
	}
	conn.SetMaxOpenConns(5)
	return conn, nil
}

// ---------------------------------------------------------------------------
// Isolamento de erro por item — nunca aborta o lote (ERP-03/FIS-03).
// Semáforo de concorrência limitado a 5 (Pitfall 5) + recover por item.
// ---------------------------------------------------------------------------

func processFiscalBatch(ctx context.Context, oracleDB *sql.DB, pgDB *sql.DB, companyID string, nfe fiscalNotaContext, itens []fiscalItemInput) fiscalExecutionSummary {
	summary := fiscalExecutionSummary{Total: len(itens)}
	var mu sync.Mutex
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, item := range itens {
		wg.Add(1)
		sem <- struct{}{}
		go func(it fiscalItemInput) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("[FiscalExecution] item=%s panic recuperado: %v", it.ID, rec)
					if perr := persistFiscalItemResult(pgDB, companyID, it.ID, "error",
						"Falha inesperada ao processar o item.", "", nil, nil); perr != nil {
						log.Printf("[FiscalExecution] item=%s persist error after panic: %v", it.ID, perr)
					}
					mu.Lock()
					summary.Error++
					mu.Unlock()
				}
			}()

			itemCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			status := processSingleFiscalItem(itemCtx, oracleDB, pgDB, companyID, nfe, it)
			mu.Lock()
			switch status {
			case "ok":
				summary.OK++
			case "sem_grupo_fiscal":
				summary.SemGrupoFiscal++
			default:
				summary.Error++
			}
			mu.Unlock()
		}(item)
	}
	wg.Wait()
	return summary
}

// processSingleFiscalItem executa o pipeline (lookup grupo fiscal → pacote
// fiscal → persistência) para um único item, isolando qualquer falha nesse
// item (log com prefixo "[FiscalExecution] item=%s err=%v", nunca aborta os
// demais itens do lote — ERP-03/FIS-03).
func processSingleFiscalItem(ctx context.Context, oracleDB *sql.DB, pgDB *sql.DB, companyID string, nfe fiscalNotaContext, it fiscalItemInput) string {
	if nfe.CodEmpresaErr != nil {
		log.Printf("[FiscalExecution] item=%s err=%v", it.ID, nfe.CodEmpresaErr)
		if perr := persistFiscalItemResult(pgDB, companyID, it.ID, "error",
			"Não foi possível determinar a filial (cod_empresa) do emitente para o lookup fiscal.", "", nil, nil); perr != nil {
			log.Printf("[FiscalExecution] item=%s persist error: %v", it.ID, perr)
		}
		return "error"
	}

	grupoFiscal, _, _, err := lookupGrupoFiscal(ctx, oracleDB, it.CProd, nfe.CodEmpresa)
	if err != nil {
		if errors.Is(err, errSemGrupoFiscal) {
			if perr := persistFiscalItemResult(pgDB, companyID, it.ID, "sem_grupo_fiscal",
				"Produto não encontrado em PROD/PRODB — grupo fiscal não pôde ser determinado.", "", nil, nil); perr != nil {
				log.Printf("[FiscalExecution] item=%s persist error: %v", it.ID, perr)
			}
			return "sem_grupo_fiscal"
		}
		log.Printf("[FiscalExecution] item=%s err=%v", it.ID, err)
		if perr := persistFiscalItemResult(pgDB, companyID, it.ID, "error",
			"Falha ao consultar o grupo fiscal no Oracle (prod/PRODB).", "", nil, nil); perr != nil {
			log.Printf("[FiscalExecution] item=%s persist error: %v", it.ID, perr)
		}
		return "error"
	}

	in := services.FiscalInput{
		PCnpjEmpresa:                 nfe.EmitCNPJ,
		PUFOrigem:                    nfe.EmitUF,
		PUFDestino:                   nfe.DestUF,
		PTipoContribuinte:            defaultTipoContribuinte,
		PTipoCentroFiscal:            defaultTipoCentroFiscal,
		PTipoOperacao:                defaultTipoOperacao,
		PEntradaSaida:                "S", // módulo cobre apenas NF-e de saída (XML-01)
		PProduto:                     it.CProd,
		PCodigoGrupoFiscal:           grupoFiscal,
		PCnpjExcecao:                 "",
		PIndicadorServico:            defaultIndicadorServico,
		PPrecoTotal:                  it.VProd,
		PDespesas:                    0, // NF-e não carrega despesas acessórias por item (só no cabeçalho, vOutro)
		PDesconto:                    it.VDesc,
		PIPI:                         it.VIPI,
		PAliquotaSimplesNacional:     0,
		FornecedorSimplesNacional:    defaultFornecedorSimplesNacional,
		PTipoIsencaoPedidoBonificado: "",
		PCFOPOperacao:                it.CFOP,
		PTipoContribuinteSecundario:  "",
		PSimulacaoCalculo:            "N",
		PDataReferenciaFiscal:        &nfe.DataEmissao,
		PCodigoIbge:                  nfe.DestCMun,
	}

	inputJSON, marshalErr := json.Marshal(in)
	if marshalErr != nil {
		inputJSON = []byte("{}")
	}

	result, callErr := services.CallFiscalPackage(ctx, oracleDB, in)
	if callErr != nil {
		log.Printf("[FiscalExecution] item=%s err=%v", it.ID, callErr)
		if perr := persistFiscalItemResult(pgDB, companyID, it.ID, "error",
			"Falha ao executar o pacote fiscal no Oracle (FCCORP_BKP).", grupoFiscal, inputJSON, nil); perr != nil {
			log.Printf("[FiscalExecution] item=%s persist error: %v", it.ID, perr)
		}
		return "error"
	}

	if perr := persistFiscalItemResult(pgDB, companyID, it.ID, "ok", "", grupoFiscal, inputJSON, &result); perr != nil {
		log.Printf("[FiscalExecution] item=%s persist error: %v", it.ID, perr)
		return "error"
	}
	return "ok"
}

// persistFiscalItemResult grava (ou atualiza, se já existir) o resultado do
// item em fiscal_execution_items. Cada item é sua própria unidade de
// trabalho — nunca uma transação única para o lote inteiro (RESEARCH.md,
// Anti-Pattern "assumir que erro de um item derruba o lote inteiro").
func persistFiscalItemResult(pgDB *sql.DB, companyID, nfeItemID, status, errMsg, grupoFiscalCodigo string, inputParams []byte, result *services.FiscalResult) error {
	fullResultJSON := []byte(`{}`)
	var baseICMS, valorICMS, baseST, valorST *float64
	var basePIS, valorPIS, baseCOFINS, valorCOFINS *float64
	var percDifal, valorPartilhaDest, valorPobreza *float64

	if result != nil {
		if b, mErr := json.Marshal(result); mErr == nil {
			fullResultJSON = b
		}
		baseICMS = &result.BaseCalculo
		valorICMS = &result.ValorImposto
		baseST = &result.BaseSubstituicao
		valorST = &result.ValorSubstituicao
		basePIS = &result.BaseCalculoPIS
		valorPIS = &result.ValorPIS
		baseCOFINS = &result.BaseCalculoCOFINS
		valorCOFINS = &result.ValorCOFINS
		percDifal = &result.PercentualDifal
		valorPartilhaDest = &result.ValorIcmsPartilhaDestino
		valorPobreza = &result.ValorIcmsPobreza
	}

	var errMsgSQL, grupoFiscalSQL interface{}
	if errMsg != "" {
		errMsgSQL = errMsg
	}
	if grupoFiscalCodigo != "" {
		grupoFiscalSQL = grupoFiscalCodigo
	}
	var inputParamsSQL interface{}
	if len(inputParams) > 0 {
		inputParamsSQL = inputParams
	}

	_, err := pgDB.Exec(`
		INSERT INTO fiscal_execution_items (
			company_id, nfe_item_id, status, error_message, executed_at,
			grupo_fiscal_codigo, input_params,
			base_calculo_icms, valor_icms, base_substituicao, valor_substituicao,
			base_calculo_pis, valor_pis, base_calculo_cofins, valor_cofins,
			percentual_difal, valor_icms_partilha_destino, valor_icms_pobreza,
			full_result
		) VALUES (
			$1, $2, $3, $4, NOW(),
			$5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17,
			$18
		)
		ON CONFLICT (nfe_item_id) DO UPDATE SET
			status                      = EXCLUDED.status,
			error_message               = EXCLUDED.error_message,
			executed_at                 = EXCLUDED.executed_at,
			grupo_fiscal_codigo         = EXCLUDED.grupo_fiscal_codigo,
			input_params                = EXCLUDED.input_params,
			base_calculo_icms           = EXCLUDED.base_calculo_icms,
			valor_icms                  = EXCLUDED.valor_icms,
			base_substituicao           = EXCLUDED.base_substituicao,
			valor_substituicao          = EXCLUDED.valor_substituicao,
			base_calculo_pis            = EXCLUDED.base_calculo_pis,
			valor_pis                   = EXCLUDED.valor_pis,
			base_calculo_cofins         = EXCLUDED.base_calculo_cofins,
			valor_cofins                = EXCLUDED.valor_cofins,
			percentual_difal            = EXCLUDED.percentual_difal,
			valor_icms_partilha_destino = EXCLUDED.valor_icms_partilha_destino,
			valor_icms_pobreza          = EXCLUDED.valor_icms_pobreza,
			full_result                 = EXCLUDED.full_result
	`,
		companyID, nfeItemID, status, errMsgSQL,
		grupoFiscalSQL, inputParamsSQL,
		baseICMS, valorICMS, baseST, valorST,
		basePIS, valorPIS, baseCOFINS, valorCOFINS,
		percDifal, valorPartilhaDest, valorPobreza,
		fullResultJSON,
	)
	return err
}
