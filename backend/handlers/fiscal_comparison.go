package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ---------------------------------------------------------------------------
// FiscalComparisonListHandler — GET /api/fiscal-comparison
// Lista, para todos os itens de todas as notas da empresa, o par de valores
// ESPERADO (nfe_saidas_itens, gabarito do XML) vs. CALCULADO (fiscal_execution_items,
// resultado do pacote fiscal), escopado por company_id resolvido via JWT.
// Cobre CMP-01/CMP-02 no nível da lista (Plano 03-01).
// ---------------------------------------------------------------------------

type comparisonItemRow struct {
	// Identificação
	ItemID      string `json:"item_id"`
	NItem       int    `json:"n_item"`
	XProd       string `json:"x_prod"`
	NCM         string `json:"ncm"`
	CFOP        string `json:"cfop"`
	NfeID       string `json:"nfe_id"`
	NumeroNFe   string `json:"numero_nfe"`
	Serie       string `json:"serie"`
	DestNome    string `json:"dest_nome"`
	DestCNPJCPF string `json:"dest_cnpj_cpf"`
	DataEmissao string `json:"data_emissao"`

	// Esperado (XML)
	EspBcIcms   float64 `json:"esp_bc_icms"`
	EspIcms     float64 `json:"esp_icms"`
	EspBcSt     float64 `json:"esp_bc_st"`
	EspSt       float64 `json:"esp_st"`
	EspBcPis    float64 `json:"esp_bc_pis"`
	EspPis      float64 `json:"esp_pis"`
	EspBcCofins float64 `json:"esp_bc_cofins"`
	EspCofins   float64 `json:"esp_cofins"`

	// Calculado (pacote fiscal) — null quando o item ainda não foi processado
	CalcBcIcms   *float64 `json:"calc_bc_icms"`
	CalcIcms     *float64 `json:"calc_icms"`
	CalcBcSt     *float64 `json:"calc_bc_st"`
	CalcSt       *float64 `json:"calc_st"`
	CalcBcPis    *float64 `json:"calc_bc_pis"`
	CalcPis      *float64 `json:"calc_pis"`
	CalcBcCofins *float64 `json:"calc_bc_cofins"`
	CalcCofins   *float64 `json:"calc_cofins"`

	FiscalStatus string `json:"fiscal_status"`
	ErrorMessage string `json:"error_message"`
}

func FiscalComparisonListHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			jsonErr(w, http.StatusMethodNotAllowed, "Método não permitido")
			return
		}

		companyID, err := erpBridgeGetCompany(db, r)
		if err != nil {
			jsonErr(w, http.StatusUnauthorized, "Não autenticado")
			return
		}

		rows, err := db.Query(`
			SELECT
				i.id, i.n_item, i.x_prod, COALESCE(i.ncm,''), COALESCE(i.cfop,''),
				n.id, COALESCE(n.numero_nfe,''), COALESCE(n.serie,''),
				COALESCE(n.dest_nome,''), COALESCE(n.dest_cnpj_cpf,''),
				TO_CHAR(n.data_emissao,'DD/MM/YYYY'),
				i.v_bc_icms, i.v_icms, i.v_bc_st, i.v_st,
				i.v_bc_pis, i.v_pis, i.v_bc_cofins, i.v_cofins,
				f.base_calculo_icms, f.valor_icms, f.base_substituicao, f.valor_substituicao,
				f.base_calculo_pis, f.valor_pis, f.base_calculo_cofins, f.valor_cofins,
				COALESCE(f.status,''), COALESCE(f.error_message,'')
			FROM nfe_saidas_itens i
			JOIN nfe_saidas n ON n.id = i.nfe_id
			LEFT JOIN fiscal_execution_items f ON f.nfe_item_id = i.id
			WHERE i.company_id = $1
			ORDER BY n.data_emissao DESC, n.numero_nfe DESC, i.n_item ASC
			LIMIT 2000`, companyID)
		if err != nil {
			log.Printf("FiscalComparisonList query error (company %s): %v", companyID, err)
			jsonErr(w, http.StatusInternalServerError, "Erro ao consultar comparação")
			return
		}
		defer rows.Close()

		list := []comparisonItemRow{}
		for rows.Next() {
			var row comparisonItemRow
			var calcBcIcms, calcIcms, calcBcSt, calcSt sql.NullFloat64
			var calcBcPis, calcPis, calcBcCofins, calcCofins sql.NullFloat64

			if err := rows.Scan(
				&row.ItemID, &row.NItem, &row.XProd, &row.NCM, &row.CFOP,
				&row.NfeID, &row.NumeroNFe, &row.Serie,
				&row.DestNome, &row.DestCNPJCPF, &row.DataEmissao,
				&row.EspBcIcms, &row.EspIcms, &row.EspBcSt, &row.EspSt,
				&row.EspBcPis, &row.EspPis, &row.EspBcCofins, &row.EspCofins,
				&calcBcIcms, &calcIcms, &calcBcSt, &calcSt,
				&calcBcPis, &calcPis, &calcBcCofins, &calcCofins,
				&row.FiscalStatus, &row.ErrorMessage,
			); err != nil {
				log.Printf("FiscalComparisonList scan error: %v", err)
				continue
			}

			if calcBcIcms.Valid {
				row.CalcBcIcms = &calcBcIcms.Float64
			}
			if calcIcms.Valid {
				row.CalcIcms = &calcIcms.Float64
			}
			if calcBcSt.Valid {
				row.CalcBcSt = &calcBcSt.Float64
			}
			if calcSt.Valid {
				row.CalcSt = &calcSt.Float64
			}
			if calcBcPis.Valid {
				row.CalcBcPis = &calcBcPis.Float64
			}
			if calcPis.Valid {
				row.CalcPis = &calcPis.Float64
			}
			if calcBcCofins.Valid {
				row.CalcBcCofins = &calcBcCofins.Float64
			}
			if calcCofins.Valid {
				row.CalcCofins = &calcCofins.Float64
			}

			list = append(list, row)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"total": len(list),
			"items": list,
		})
	}
}

// ---------------------------------------------------------------------------
// FiscalComparisonDetailHandler — GET /api/fiscal-comparison/{id}
// Detalhe de um único item: os mesmos pares esperado x calculado da lista,
// mais os campos "só calculado" (DIFAL, FCP, full_result) para a seção de
// auditoria do Dialog (D-07). Escopado por company_id — item de outra
// empresa retorna 404, nunca 200 com dado vazado (T-03-06/IDOR).
// Cobre CMP-03/CMP-04 no nível do detalhe (Plano 03-02).
// ---------------------------------------------------------------------------

type comparisonItemDetail struct {
	// Identificação
	ItemID    string `json:"item_id"`
	NItem     int    `json:"n_item"`
	XProd     string `json:"x_prod"`
	NCM       string `json:"ncm"`
	CFOP      string `json:"cfop"`
	NfeID     string `json:"nfe_id"`
	NumeroNFe string `json:"numero_nfe"`
	Serie     string `json:"serie"`
	DestNome  string `json:"dest_nome"`

	// Esperado (XML)
	EspBcIcms   float64 `json:"esp_bc_icms"`
	EspIcms     float64 `json:"esp_icms"`
	EspBcSt     float64 `json:"esp_bc_st"`
	EspSt       float64 `json:"esp_st"`
	EspBcPis    float64 `json:"esp_bc_pis"`
	EspPis      float64 `json:"esp_pis"`
	EspBcCofins float64 `json:"esp_bc_cofins"`
	EspCofins   float64 `json:"esp_cofins"`

	// Calculado (pacote fiscal) — null quando o item ainda não foi processado
	CalcBcIcms   *float64 `json:"calc_bc_icms"`
	CalcIcms     *float64 `json:"calc_icms"`
	CalcBcSt     *float64 `json:"calc_bc_st"`
	CalcSt       *float64 `json:"calc_st"`
	CalcBcPis    *float64 `json:"calc_bc_pis"`
	CalcPis      *float64 `json:"calc_pis"`
	CalcBcCofins *float64 `json:"calc_bc_cofins"`
	CalcCofins   *float64 `json:"calc_cofins"`

	// Só calculado (sem par no XML, D-07) — DIFAL/FCP
	PercentualDifal          *float64 `json:"percentual_difal"`
	ValorIcmsPartilhaDestino *float64 `json:"valor_icms_partilha_destino"` // DIFAL
	ValorIcmsPobreza         *float64 `json:"valor_icms_pobreza"`          // FCP

	GrupoFiscalCodigo string          `json:"grupo_fiscal_codigo"`
	FiscalStatus      string          `json:"fiscal_status"`
	ErrorMessage      string          `json:"error_message"`
	FullResult        json.RawMessage `json:"full_result"` // ~88 campos do pacote fiscal (auditoria)
}

func FiscalComparisonDetailHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			jsonErr(w, http.StatusMethodNotAllowed, "Método não permitido")
			return
		}

		companyID, err := erpBridgeGetCompany(db, r)
		if err != nil {
			jsonErr(w, http.StatusUnauthorized, "Não autenticado")
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/api/fiscal-comparison/")
		id = strings.TrimSuffix(id, "/")
		if id == "" {
			jsonErr(w, http.StatusBadRequest, "ID do item não informado")
			return
		}

		var row comparisonItemDetail
		var calcBcIcms, calcIcms, calcBcSt, calcSt sql.NullFloat64
		var calcBcPis, calcPis, calcBcCofins, calcCofins sql.NullFloat64
		var percentualDifal, valorIcmsPartilhaDestino, valorIcmsPobreza sql.NullFloat64
		var fullResult []byte

		err = db.QueryRow(`
			SELECT
				i.id, i.n_item, i.x_prod, COALESCE(i.ncm,''), COALESCE(i.cfop,''),
				n.id AS nfe_id, COALESCE(n.numero_nfe,''), COALESCE(n.serie,''), COALESCE(n.dest_nome,''),
				i.v_bc_icms, i.v_icms, i.v_bc_st, i.v_st,
				i.v_bc_pis, i.v_pis, i.v_bc_cofins, i.v_cofins,
				f.base_calculo_icms, f.valor_icms, f.base_substituicao, f.valor_substituicao,
				f.base_calculo_pis, f.valor_pis, f.base_calculo_cofins, f.valor_cofins,
				f.percentual_difal, f.valor_icms_partilha_destino, f.valor_icms_pobreza,
				COALESCE(f.grupo_fiscal_codigo,''), COALESCE(f.status,''), COALESCE(f.error_message,''),
				COALESCE(f.full_result, '{}'::jsonb)
			FROM nfe_saidas_itens i
			JOIN nfe_saidas n ON n.id = i.nfe_id
			LEFT JOIN fiscal_execution_items f ON f.nfe_item_id = i.id
			WHERE i.id = $1 AND i.company_id = $2`, id, companyID,
		).Scan(
			&row.ItemID, &row.NItem, &row.XProd, &row.NCM, &row.CFOP,
			&row.NfeID, &row.NumeroNFe, &row.Serie, &row.DestNome,
			&row.EspBcIcms, &row.EspIcms, &row.EspBcSt, &row.EspSt,
			&row.EspBcPis, &row.EspPis, &row.EspBcCofins, &row.EspCofins,
			&calcBcIcms, &calcIcms, &calcBcSt, &calcSt,
			&calcBcPis, &calcPis, &calcBcCofins, &calcCofins,
			&percentualDifal, &valorIcmsPartilhaDestino, &valorIcmsPobreza,
			&row.GrupoFiscalCodigo, &row.FiscalStatus, &row.ErrorMessage,
			&fullResult,
		)
		if err == sql.ErrNoRows {
			jsonErr(w, http.StatusNotFound, "Item não encontrado")
			return
		}
		if err != nil {
			log.Printf("FiscalComparisonDetail query error (id %s, company %s): %v", id, companyID, err)
			jsonErr(w, http.StatusInternalServerError, "Erro ao consultar item")
			return
		}

		if calcBcIcms.Valid {
			row.CalcBcIcms = &calcBcIcms.Float64
		}
		if calcIcms.Valid {
			row.CalcIcms = &calcIcms.Float64
		}
		if calcBcSt.Valid {
			row.CalcBcSt = &calcBcSt.Float64
		}
		if calcSt.Valid {
			row.CalcSt = &calcSt.Float64
		}
		if calcBcPis.Valid {
			row.CalcBcPis = &calcBcPis.Float64
		}
		if calcPis.Valid {
			row.CalcPis = &calcPis.Float64
		}
		if calcBcCofins.Valid {
			row.CalcBcCofins = &calcBcCofins.Float64
		}
		if calcCofins.Valid {
			row.CalcCofins = &calcCofins.Float64
		}
		if percentualDifal.Valid {
			row.PercentualDifal = &percentualDifal.Float64
		}
		if valorIcmsPartilhaDestino.Valid {
			row.ValorIcmsPartilhaDestino = &valorIcmsPartilhaDestino.Float64
		}
		if valorIcmsPobreza.Valid {
			row.ValorIcmsPobreza = &valorIcmsPobreza.Float64
		}
		row.FullResult = json.RawMessage(fullResult)

		json.NewEncoder(w).Encode(row)
	}
}
