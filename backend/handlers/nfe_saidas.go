package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// ---------------------------------------------------------------------------
// Structs de parsing XML — nomes dos campos refletem as tags da NF-e (SEFAZ).
// Cópia seletiva de FB_APU04/backend/handlers/nfe_saidas.go — sem adaptação,
// o mapeamento SEFAZ é padrão de mercado, independe do projeto.
// ---------------------------------------------------------------------------

type nfeProc struct {
	XMLName xml.Name `xml:"nfeProc"`
	NFe     nfe      `xml:"NFe"`
	ProtNFe protNFe  `xml:"protNFe"`
}

type nfe struct {
	InfNFe infNFe `xml:"infNFe"`
}

type protNFe struct {
	InfProt infProt `xml:"infProt"`
}

type infProt struct {
	ChNFe string `xml:"chNFe"` // chave 44 dígitos
}

type infNFe struct {
	ID     string `xml:"Id,attr"` // "NFe" + 44 dígitos (fallback)
	Ide    ide    `xml:"ide"`
	Emit   emit   `xml:"emit"`
	Dest   dest   `xml:"dest"`
	Det    []det  `xml:"det"` // array de itens da nota
	Total  total  `xml:"total"`
	Transp transp `xml:"transp"`
}

// transp.modFrete: modalidade do frete (0=remetente/CIF, 1=destinatário/FOB,
// 2=terceiros, 3=próprio remetente, 4=próprio destinatário, 9=sem transporte).
type transp struct {
	ModFrete string `xml:"modFrete"`
}

type det struct {
	NItem   string     `xml:"nItem,attr"`
	Prod    prod       `xml:"prod"`
	Imposto detImposto `xml:"imposto"`
}

type prod struct {
	CProd string `xml:"cProd"`
	XProd string `xml:"xProd"`
	NCM   string `xml:"NCM"`
	CEST  string `xml:"CEST"`
	CFOP  string `xml:"CFOP"`
	VProd string `xml:"vProd"`
	VDesc string `xml:"vDesc"`
}

type detImposto struct {
	ICMS   detICMS   `xml:"ICMS"`
	PIS    detPIS    `xml:"PIS"`
	COFINS detCOFINS `xml:"COFINS"`
	IPI    detIPI    `xml:"IPI"`
}

// detICMSGrupo captura qualquer sub-grupo ICMS (CST ou CSOSN) sem mapear ~30 variantes.
// orig: código da Tabela A do CST (origem da mercadoria, 0-8).
type detICMSGrupo struct {
	Orig  string `xml:"orig"`
	CST   string `xml:"CST"`
	CSOSN string `xml:"CSOSN"`
	VBC   string `xml:"vBC"`
	VICMS string `xml:"vICMS"`
	VBCST string `xml:"vBCST"`   // base de cálculo do ICMS-ST por item
	VST   string `xml:"vICMSST"` // ICMS-ST retido pelo fornecedor por item
}

type detICMS struct {
	Grupos []detICMSGrupo `xml:",any"`
}

type detPIS struct {
	CST    string `xml:"PISAliq>CST"`
	VPIS   string `xml:"PISAliq>vPIS"`
	VBCPIS string `xml:"PISAliq>vBC"`
}

type detCOFINS struct {
	CST       string `xml:"COFINSAliq>CST"`
	VCOFINS   string `xml:"COFINSAliq>vCOFINS"`
	VBCCOFINS string `xml:"COFINSAliq>vBC"`
}

type detIPI struct {
	VIPI string `xml:"IPITrib>vIPI"`
}

type ide struct {
	Mod      string `xml:"mod"` // 55 ou 65
	Serie    string `xml:"serie"`
	NNF      string `xml:"nNF"`
	DhEmi    string `xml:"dhEmi"` // ISO8601 → data_emissao + mes_ano
	TpNF     string `xml:"tpNF"`  // 1 = saída (rejeitar se ≠ 1)
	NatOp    string `xml:"natOp"`
	IndFinal string `xml:"indFinal"` // "0"=B2B/normal, "1"=consumidor final; "" para NF-e antigas
}

type emit struct {
	CNPJ      string    `xml:"CNPJ"`
	XNome     string    `xml:"xNome"`
	CRT       string    `xml:"CRT"` // "1" = Simples Nacional
	EnderEmit enderEmit `xml:"enderEmit"`
}

type enderEmit struct {
	XMun string `xml:"xMun"`
	UF   string `xml:"UF"`
}

type dest struct {
	CNPJ      string    `xml:"CNPJ"`
	CPF       string    `xml:"CPF"`
	XNome     string    `xml:"xNome"`
	EnderDest enderDest `xml:"enderDest"`
}

type enderDest struct {
	CMun string `xml:"cMun"` // código IBGE 7 dígitos
	UF   string `xml:"UF"`
}

type total struct {
	ICMSTot   icmsTot   `xml:"ICMSTot"`
	IBSCBSTot ibsCbsTot `xml:"IBSCBSTot"`
}

type icmsTot struct {
	VBC        string `xml:"vBC"`
	VICMS      string `xml:"vICMS"`
	VICMSDeson string `xml:"vICMSDeson"`
	VFCP       string `xml:"vFCP"`
	VBCST      string `xml:"vBCST"`
	VST        string `xml:"vST"`
	VFcpST     string `xml:"vFCPST"`
	VFcpSTRet  string `xml:"vFCPSTRet"`
	VProd      string `xml:"vProd"`
	VFrete     string `xml:"vFrete"`
	VSeg       string `xml:"vSeg"`
	VDesc      string `xml:"vDesc"`
	VII        string `xml:"vII"`
	VIPI       string `xml:"vIPI"`
	VIPIDevol  string `xml:"vIPIDevol"`
	VPIS       string `xml:"vPIS"`
	VCOFINS    string `xml:"vCOFINS"`
	VOutro     string `xml:"vOutro"`
	VNF        string `xml:"vNF"`
}

type ibsCbsTot struct {
	VBCIBSCBS string `xml:"vBCIBSCBS"`
	GIBS      gIBS   `xml:"gIBS"`
	GCBS      gCBS   `xml:"gCBS"`
}

type gIBS struct {
	GIBSuf    gIBSuf  `xml:"gIBSUF"`
	GIBSMun   gIBSMun `xml:"gIBSMun"`
	VIBS      string  `xml:"vIBS"`
	VCredPres string  `xml:"vCredPres"`
}

type gIBSuf struct {
	VIBSuf string `xml:"vIBSUF"`
}
type gIBSMun struct {
	VIBSMun string `xml:"vIBSMun"`
}

type gCBS struct {
	VCBS      string `xml:"vCBS"`
	VCredPres string `xml:"vCredPres"`
}

// ---------------------------------------------------------------------------
// Helpers de conversão
// ---------------------------------------------------------------------------

func toDecimal(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func toNullDecimal(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// toNullSmallInt converte string "0"/"1"/"" para interface{} compatível com SMALLINT nullable.
// "" → nil (NULL — ausência de dado para NF-e históricas)
// "1" → 1 (consumidor final)
// qualquer outro valor → 0 (B2B/normal — fallback seguro)
func toNullSmallInt(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if s == "1" {
		return 1
	}
	return 0
}

// nfeCharsetReader converte encodings declarados no XML (ex: windows-1252) para UTF-8.
func nfeCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.ReplaceAll(charset, "-", "")) {
	case "windows1252", "cp1252":
		return transform.NewReader(input, charmap.Windows1252.NewDecoder()), nil
	case "iso88591", "latin1":
		return transform.NewReader(input, charmap.ISO8859_1.NewDecoder()), nil
	default:
		return nil, fmt.Errorf("encoding não suportado: %s", charset)
	}
}

// parseNFeXML lê bytes de um XML de NF-e e retorna os dados estruturados.
func parseNFeXML(data []byte) (*nfeProc, error) {
	// Remove namespace para simplificar o parsing
	data = bytes.ReplaceAll(data,
		[]byte(` xmlns="http://www.portalfiscal.inf.br/nfe"`), []byte(""))
	data = bytes.ReplaceAll(data,
		[]byte(` xmlns='http://www.portalfiscal.inf.br/nfe'`), []byte(""))
	for _, ns := range [][]byte{
		[]byte(` xmlns:nfe="http://www.portalfiscal.inf.br/nfe"`),
		[]byte(` xmlns:nfe='http://www.portalfiscal.inf.br/nfe'`),
	} {
		data = bytes.ReplaceAll(data, ns, []byte(""))
	}
	data = bytes.ReplaceAll(data, []byte("<nfe:"), []byte("<"))
	data = bytes.ReplaceAll(data, []byte("</nfe:"), []byte("</"))

	// PITFALL: XMLs sem wrapper nfeProc — se a raiz for <NFe>, envolver
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("<NFe")) {
		data = append([]byte("<nfeProc>"), append(trimmed, []byte("</nfeProc>")...)...)
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = nfeCharsetReader

	var proc nfeProc
	if err := dec.Decode(&proc); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("arquivo truncado ou incompleto")
		}
		// Fallback: tenta reinterpretar como Windows-1252 (XMLs sem declaração de encoding)
		converted, convErr := convertWindows1252(data)
		if convErr != nil {
			return nil, fmt.Errorf("XML inválido: %w", err)
		}
		dec2 := xml.NewDecoder(bytes.NewReader(converted))
		dec2.CharsetReader = nfeCharsetReader
		var proc2 nfeProc
		if err2 := dec2.Decode(&proc2); err2 != nil {
			if err2 == io.EOF {
				return nil, fmt.Errorf("arquivo truncado ou incompleto")
			}
			return nil, fmt.Errorf("XML inválido (encoding não reconhecido): %w", err2)
		}
		return &proc2, nil
	}
	return &proc, nil
}

// convertWindows1252 converte bytes Windows-1252 para UTF-8.
func convertWindows1252(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), charmap.Windows1252.NewDecoder())
	return io.ReadAll(reader)
}

// extractChave retorna a chave de acesso de 44 dígitos.
func extractChave(proc *nfeProc) string {
	ch := strings.TrimSpace(proc.ProtNFe.InfProt.ChNFe)
	if len(ch) == 44 {
		return ch
	}
	id := strings.TrimSpace(proc.NFe.InfNFe.ID)
	if strings.HasPrefix(id, "NFe") && len(id) == 47 {
		return id[3:]
	}
	return ""
}

// parseDhEmi converte dhEmi ISO8601 em data e mes_ano.
func parseDhEmi(dhEmi string) (time.Time, string, error) {
	dhEmi = strings.TrimSpace(dhEmi)
	formats := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	var t time.Time
	var err error
	for _, f := range formats {
		t, err = time.Parse(f, dhEmi)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, "", fmt.Errorf("data inválida '%s'", dhEmi)
	}
	mesAno := fmt.Sprintf("%02d/%04d", t.Month(), t.Year())
	return t, mesAno, nil
}

// ---------------------------------------------------------------------------
// insertNFeItens insere os itens de uma nota em nfe_saidas_itens.
// Adaptação: tabela fixa (FB_TESTESFC só tem nfe_saidas_itens, sem entradas).
// ON CONFLICT (nfe_id, n_item) DO UPDATE garante reimportação idempotente.
// ---------------------------------------------------------------------------

func insertNFeItens(tx *sql.Tx, nfeID string, companyID string, dets []det) error {
	for _, d := range dets {
		nItem, _ := strconv.Atoi(d.NItem)
		if nItem == 0 {
			continue
		}

		// Extrai CST/CSOSN e origem (Tabela A) do primeiro grupo ICMS presente
		cstICMS := ""
		cstOrig := ""
		vBCICMS := 0.0
		vICMS := 0.0
		vBCST := 0.0
		vST := 0.0
		if len(d.Imposto.ICMS.Grupos) > 0 {
			g := d.Imposto.ICMS.Grupos[0]
			if g.CSOSN != "" {
				cstICMS = g.CSOSN
			} else {
				cstICMS = g.CST
			}
			cstOrig = strings.TrimSpace(g.Orig)
			vBCICMS = toDecimal(g.VBC)
			vICMS = toDecimal(g.VICMS)
			vBCST = toDecimal(g.VBCST)
			vST = toDecimal(g.VST)
		}

		_, err := tx.Exec(`
			INSERT INTO nfe_saidas_itens (
				nfe_id, company_id, n_item,
				c_prod, x_prod, ncm, cfop,
				cst_icms, cst_pis, cst_cofins,
				cst_orig, cest,
				v_prod, v_desc, v_bc_icms, v_icms,
				v_bc_pis, v_pis,
				v_bc_cofins, v_cofins,
				v_ipi,
				v_bc_st, v_st
			) VALUES (
				$1, $2, $3,
				$4, $5, $6, $7,
				$8, $9, $10,
				$11, $12,
				$13, $14, $15, $16,
				$17, $18,
				$19, $20,
				$21,
				$22, $23
			)
			ON CONFLICT (nfe_id, n_item) DO UPDATE SET
				c_prod       = EXCLUDED.c_prod,
				x_prod       = EXCLUDED.x_prod,
				ncm          = EXCLUDED.ncm,
				cfop         = EXCLUDED.cfop,
				cst_icms     = EXCLUDED.cst_icms,
				cst_pis      = EXCLUDED.cst_pis,
				cst_cofins   = EXCLUDED.cst_cofins,
				cst_orig     = EXCLUDED.cst_orig,
				cest         = EXCLUDED.cest,
				v_prod       = EXCLUDED.v_prod,
				v_desc       = EXCLUDED.v_desc,
				v_bc_icms    = EXCLUDED.v_bc_icms,
				v_icms       = EXCLUDED.v_icms,
				v_bc_pis     = EXCLUDED.v_bc_pis,
				v_pis        = EXCLUDED.v_pis,
				v_bc_cofins  = EXCLUDED.v_bc_cofins,
				v_cofins     = EXCLUDED.v_cofins,
				v_ipi        = EXCLUDED.v_ipi,
				v_bc_st      = EXCLUDED.v_bc_st,
				v_st         = EXCLUDED.v_st
		`,
			nfeID, companyID, nItem,
			d.Prod.CProd, d.Prod.XProd, d.Prod.NCM, d.Prod.CFOP,
			cstICMS, d.Imposto.PIS.CST, d.Imposto.COFINS.CST,
			cstOrig, d.Prod.CEST,
			toDecimal(d.Prod.VProd), toDecimal(d.Prod.VDesc), vBCICMS, vICMS,
			toDecimal(d.Imposto.PIS.VBCPIS), toDecimal(d.Imposto.PIS.VPIS),
			toDecimal(d.Imposto.COFINS.VBCCOFINS), toDecimal(d.Imposto.COFINS.VCOFINS),
			toDecimal(d.Imposto.IPI.VIPI),
			vBCST, vST,
		)
		if err != nil {
			return fmt.Errorf("item %d: %w", nItem, err)
		}
	}
	return nil
}

// insertNFeSaidaHeader persiste (ou atualiza, se já existir) o cabeçalho da
// nota em nfe_saidas. Retorna o id da nota (novo ou existente).
func insertNFeSaidaHeader(tx *sql.Tx, companyID, chave string, inf infNFe) (string, error) {
	modInt, _ := strconv.Atoi(strings.TrimSpace(inf.Ide.Mod))
	ic := inf.Total.ICMSTot
	ib := inf.Total.IBSCBSTot

	destCNPJCPF := strings.TrimSpace(inf.Dest.CNPJ)
	if destCNPJCPF == "" {
		destCNPJCPF = strings.TrimSpace(inf.Dest.CPF)
	}

	dataEmissao, mesAno, err := parseDhEmi(inf.Ide.DhEmi)
	if err != nil {
		return "", err
	}

	var nfeID string
	err = tx.QueryRow(`
		INSERT INTO nfe_saidas (
			company_id, chave_nfe, modelo, serie, numero_nfe,
			data_emissao, mes_ano, nat_op,
			emit_cnpj, emit_nome, emit_uf, emit_municipio,
			dest_cnpj_cpf, dest_nome, dest_uf, dest_c_mun,
			v_bc, v_icms, v_icms_deson, v_fcp,
			v_bc_st, v_st, v_fcp_st, v_fcp_st_ret,
			v_prod, v_frete, v_seg, v_desc,
			v_ii, v_ipi, v_ipi_devol, v_pis, v_cofins, v_outro, v_nf,
			v_bc_ibs_cbs, v_ibs_uf, v_ibs_mun, v_ibs, v_cred_pres_ibs,
			v_cbs, v_cred_pres_cbs
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,
			$9,$10,$11,$12,
			$13,$14,$15,$16,
			$17,$18,$19,$20,
			$21,$22,$23,$24,
			$25,$26,$27,$28,
			$29,$30,$31,$32,$33,$34,$35,
			$36,$37,$38,$39,$40,
			$41,$42
		)
		ON CONFLICT (company_id, chave_nfe) DO UPDATE SET
			mes_ano        = EXCLUDED.mes_ano,
			emit_cnpj      = EXCLUDED.emit_cnpj,
			emit_nome      = EXCLUDED.emit_nome,
			emit_uf        = EXCLUDED.emit_uf,
			emit_municipio = EXCLUDED.emit_municipio,
			dest_cnpj_cpf  = EXCLUDED.dest_cnpj_cpf,
			dest_nome      = EXCLUDED.dest_nome,
			dest_uf        = EXCLUDED.dest_uf,
			dest_c_mun     = EXCLUDED.dest_c_mun,
			v_bc           = EXCLUDED.v_bc,
			v_icms         = EXCLUDED.v_icms,
			v_icms_deson   = EXCLUDED.v_icms_deson,
			v_fcp          = EXCLUDED.v_fcp,
			v_bc_st        = EXCLUDED.v_bc_st,
			v_st           = EXCLUDED.v_st,
			v_fcp_st       = EXCLUDED.v_fcp_st,
			v_fcp_st_ret   = EXCLUDED.v_fcp_st_ret,
			v_prod         = EXCLUDED.v_prod,
			v_frete        = EXCLUDED.v_frete,
			v_seg          = EXCLUDED.v_seg,
			v_desc         = EXCLUDED.v_desc,
			v_ii           = EXCLUDED.v_ii,
			v_ipi          = EXCLUDED.v_ipi,
			v_ipi_devol    = EXCLUDED.v_ipi_devol,
			v_pis          = EXCLUDED.v_pis,
			v_cofins       = EXCLUDED.v_cofins,
			v_outro        = EXCLUDED.v_outro,
			v_nf           = EXCLUDED.v_nf,
			v_bc_ibs_cbs   = EXCLUDED.v_bc_ibs_cbs,
			v_ibs_uf       = EXCLUDED.v_ibs_uf,
			v_ibs_mun      = EXCLUDED.v_ibs_mun,
			v_ibs          = EXCLUDED.v_ibs,
			v_cred_pres_ibs = EXCLUDED.v_cred_pres_ibs,
			v_cbs          = EXCLUDED.v_cbs,
			v_cred_pres_cbs = EXCLUDED.v_cred_pres_cbs
		RETURNING id`,
		companyID, chave, modInt, inf.Ide.Serie, inf.Ide.NNF,
		dataEmissao, mesAno, inf.Ide.NatOp,
		inf.Emit.CNPJ, inf.Emit.XNome, inf.Emit.EnderEmit.UF, inf.Emit.EnderEmit.XMun,
		destCNPJCPF, inf.Dest.XNome, inf.Dest.EnderDest.UF, inf.Dest.EnderDest.CMun,
		toDecimal(ic.VBC), toDecimal(ic.VICMS), toDecimal(ic.VICMSDeson), toDecimal(ic.VFCP),
		toDecimal(ic.VBCST), toDecimal(ic.VST), toDecimal(ic.VFcpST), toDecimal(ic.VFcpSTRet),
		toDecimal(ic.VProd), toDecimal(ic.VFrete), toDecimal(ic.VSeg), toDecimal(ic.VDesc),
		toDecimal(ic.VII), toDecimal(ic.VIPI), toDecimal(ic.VIPIDevol), toDecimal(ic.VPIS), toDecimal(ic.VCOFINS), toDecimal(ic.VOutro), toDecimal(ic.VNF),
		toNullDecimal(ib.VBCIBSCBS), toNullDecimal(ib.GIBS.GIBSuf.VIBSuf), toNullDecimal(ib.GIBS.GIBSMun.VIBSMun),
		toNullDecimal(ib.GIBS.VIBS), toNullDecimal(ib.GIBS.VCredPres),
		toNullDecimal(ib.GCBS.VCBS), toNullDecimal(ib.GCBS.VCredPres),
	).Scan(&nfeID)
	if err != nil {
		return "", err
	}
	return nfeID, nil
}

// ---------------------------------------------------------------------------
// NFeSaidasListHandler — GET /api/nfe-saidas
// Lista as notas da empresa do JWT (escopo por company_id).
// ---------------------------------------------------------------------------

type nfeSaidaRow struct {
	ID          string `json:"id"`
	ChaveNFe    string `json:"chave_nfe"`
	Modelo      int    `json:"modelo"`
	Serie       string `json:"serie"`
	NumeroNFe   string `json:"numero_nfe"`
	DataEmissao string `json:"data_emissao"`
	MesAno      string `json:"mes_ano"`
	NatOp       string `json:"nat_op"`

	EmitCNPJ      string `json:"emit_cnpj"`
	EmitNome      string `json:"emit_nome"`
	EmitUF        string `json:"emit_uf"`
	EmitMunicipio string `json:"emit_municipio"`

	DestCNPJCPF string `json:"dest_cnpj_cpf"`
	DestNome    string `json:"dest_nome"`
	DestUF      string `json:"dest_uf"`
	DestCMun    string `json:"dest_c_mun"`

	VBC        float64 `json:"v_bc"`
	VICMS      float64 `json:"v_icms"`
	VICMSDeson float64 `json:"v_icms_deson"`
	VFCP       float64 `json:"v_fcp"`
	VBcST      float64 `json:"v_bc_st"`
	VST        float64 `json:"v_st"`
	VFcpST     float64 `json:"v_fcp_st"`
	VFcpSTRet  float64 `json:"v_fcp_st_ret"`
	VProd      float64 `json:"v_prod"`
	VFrete     float64 `json:"v_frete"`
	VSeg       float64 `json:"v_seg"`
	VDesc      float64 `json:"v_desc"`
	VII        float64 `json:"v_ii"`
	VIPI       float64 `json:"v_ipi"`
	VIPIDevol  float64 `json:"v_ipi_devol"`
	VPIS       float64 `json:"v_pis"`
	VCOFINS    float64 `json:"v_cofins"`
	VOutro     float64 `json:"v_outro"`
	VNF        float64 `json:"v_nf"`

	VBCIbsCbs    *float64 `json:"v_bc_ibs_cbs"`
	VIBSuf       *float64 `json:"v_ibs_uf"`
	VIBSMun      *float64 `json:"v_ibs_mun"`
	VIBS         *float64 `json:"v_ibs"`
	VCredPresIBS *float64 `json:"v_cred_pres_ibs"`
	VCBS         *float64 `json:"v_cbs"`
	VCredPresCBS *float64 `json:"v_cred_pres_cbs"`
}

func NFeSaidasListHandler(db *sql.DB) http.HandlerFunc {
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
				id, chave_nfe, modelo, serie, COALESCE(numero_nfe,''),
				TO_CHAR(data_emissao, 'DD/MM/YYYY'), mes_ano, COALESCE(nat_op,''),
				emit_cnpj, COALESCE(emit_nome,''), COALESCE(emit_uf,''), COALESCE(emit_municipio,''),
				COALESCE(dest_cnpj_cpf,''), COALESCE(dest_nome,''), COALESCE(dest_uf,''), COALESCE(dest_c_mun,''),
				v_bc, v_icms, v_icms_deson, v_fcp,
				v_bc_st, v_st, v_fcp_st, v_fcp_st_ret,
				v_prod, v_frete, v_seg, v_desc,
				v_ii, v_ipi, v_ipi_devol, v_pis, v_cofins, v_outro, v_nf,
				v_bc_ibs_cbs, v_ibs_uf, v_ibs_mun, v_ibs, v_cred_pres_ibs,
				v_cbs, v_cred_pres_cbs
			FROM nfe_saidas
			WHERE company_id = $1
			ORDER BY data_emissao DESC, numero_nfe DESC
			LIMIT 500`, companyID)
		if err != nil {
			log.Printf("NFeSaidasList query error (company %s): %v", companyID, err)
			jsonErr(w, http.StatusInternalServerError, "Erro ao consultar notas")
			return
		}
		defer rows.Close()

		list := []nfeSaidaRow{}
		for rows.Next() {
			var row nfeSaidaRow
			if err := rows.Scan(
				&row.ID, &row.ChaveNFe, &row.Modelo, &row.Serie, &row.NumeroNFe,
				&row.DataEmissao, &row.MesAno, &row.NatOp,
				&row.EmitCNPJ, &row.EmitNome, &row.EmitUF, &row.EmitMunicipio,
				&row.DestCNPJCPF, &row.DestNome, &row.DestUF, &row.DestCMun,
				&row.VBC, &row.VICMS, &row.VICMSDeson, &row.VFCP,
				&row.VBcST, &row.VST, &row.VFcpST, &row.VFcpSTRet,
				&row.VProd, &row.VFrete, &row.VSeg, &row.VDesc,
				&row.VII, &row.VIPI, &row.VIPIDevol, &row.VPIS, &row.VCOFINS, &row.VOutro, &row.VNF,
				&row.VBCIbsCbs, &row.VIBSuf, &row.VIBSMun, &row.VIBS, &row.VCredPresIBS,
				&row.VCBS, &row.VCredPresCBS,
			); err != nil {
				log.Printf("NFeSaidasList scan error: %v", err)
				continue
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
// NFeSaidaDetailHandler — GET /api/nfe-saidas/{id}
// Retorna o cabeçalho + itens da nota, validando que ela pertence à company do JWT.
// ---------------------------------------------------------------------------

type nfeSaidaItemRow struct {
	ID         string  `json:"id"`
	NItem      int     `json:"n_item"`
	CProd      string  `json:"c_prod"`
	XProd      string  `json:"x_prod"`
	NCM        string  `json:"ncm"`
	CEST       string  `json:"cest"`
	CFOP       string  `json:"cfop"`
	CSTICMS    string  `json:"cst_icms"`
	CSTOrig    string  `json:"cst_orig"`
	CSTPIS     string  `json:"cst_pis"`
	CSTCOFINS  string  `json:"cst_cofins"`
	VProd      float64 `json:"v_prod"`
	VBCICMS    float64 `json:"v_bc_icms"`
	VICMS      float64 `json:"v_icms"`
	VBCST      float64 `json:"v_bc_st"`
	VST        float64 `json:"v_st"`
	VIPI       float64 `json:"v_ipi"`
	VBCPIS     float64 `json:"v_bc_pis"`
	VPIS       float64 `json:"v_pis"`
	VBCCOFINS  float64 `json:"v_bc_cofins"`
	VCOFINS    float64 `json:"v_cofins"`
	VIBS       float64 `json:"v_ibs"`
	VCBS       float64 `json:"v_cbs"`
	CClassTrib string  `json:"cclasstrib"`
}

func NFeSaidaDetailHandler(db *sql.DB) http.HandlerFunc {
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

		id := strings.TrimPrefix(r.URL.Path, "/api/nfe-saidas/")
		id = strings.TrimSuffix(id, "/")
		if id == "" {
			jsonErr(w, http.StatusBadRequest, "ID da nota não informado")
			return
		}

		var row nfeSaidaRow
		err = db.QueryRow(`
			SELECT
				id, chave_nfe, modelo, serie, COALESCE(numero_nfe,''),
				TO_CHAR(data_emissao, 'DD/MM/YYYY'), mes_ano, COALESCE(nat_op,''),
				emit_cnpj, COALESCE(emit_nome,''), COALESCE(emit_uf,''), COALESCE(emit_municipio,''),
				COALESCE(dest_cnpj_cpf,''), COALESCE(dest_nome,''), COALESCE(dest_uf,''), COALESCE(dest_c_mun,''),
				v_bc, v_icms, v_icms_deson, v_fcp,
				v_bc_st, v_st, v_fcp_st, v_fcp_st_ret,
				v_prod, v_frete, v_seg, v_desc,
				v_ii, v_ipi, v_ipi_devol, v_pis, v_cofins, v_outro, v_nf,
				v_bc_ibs_cbs, v_ibs_uf, v_ibs_mun, v_ibs, v_cred_pres_ibs,
				v_cbs, v_cred_pres_cbs
			FROM nfe_saidas
			WHERE id = $1 AND company_id = $2`, id, companyID,
		).Scan(
			&row.ID, &row.ChaveNFe, &row.Modelo, &row.Serie, &row.NumeroNFe,
			&row.DataEmissao, &row.MesAno, &row.NatOp,
			&row.EmitCNPJ, &row.EmitNome, &row.EmitUF, &row.EmitMunicipio,
			&row.DestCNPJCPF, &row.DestNome, &row.DestUF, &row.DestCMun,
			&row.VBC, &row.VICMS, &row.VICMSDeson, &row.VFCP,
			&row.VBcST, &row.VST, &row.VFcpST, &row.VFcpSTRet,
			&row.VProd, &row.VFrete, &row.VSeg, &row.VDesc,
			&row.VII, &row.VIPI, &row.VIPIDevol, &row.VPIS, &row.VCOFINS, &row.VOutro, &row.VNF,
			&row.VBCIbsCbs, &row.VIBSuf, &row.VIBSMun, &row.VIBS, &row.VCredPresIBS,
			&row.VCBS, &row.VCredPresCBS,
		)
		if err == sql.ErrNoRows {
			jsonErr(w, http.StatusNotFound, "Nota não encontrada")
			return
		}
		if err != nil {
			log.Printf("NFeSaidaDetail query error (id %s, company %s): %v", id, companyID, err)
			jsonErr(w, http.StatusInternalServerError, "Erro ao consultar nota")
			return
		}

		itemRows, err := db.Query(`
			SELECT
				id, n_item, COALESCE(c_prod,''), x_prod, COALESCE(ncm,''), COALESCE(cest,''),
				COALESCE(cfop,''), COALESCE(cst_icms,''), COALESCE(cst_orig,''),
				COALESCE(cst_pis,''), COALESCE(cst_cofins,''),
				v_prod, v_bc_icms, v_icms, v_bc_st, v_st, v_ipi,
				v_bc_pis, v_pis, v_bc_cofins, v_cofins, v_ibs, v_cbs, COALESCE(cclasstrib,'')
			FROM nfe_saidas_itens
			WHERE nfe_id = $1
			ORDER BY n_item ASC`, row.ID)
		if err != nil {
			log.Printf("NFeSaidaDetail itens query error (nfe %s): %v", row.ID, err)
			jsonErr(w, http.StatusInternalServerError, "Erro ao consultar itens da nota")
			return
		}
		defer itemRows.Close()

		itens := []nfeSaidaItemRow{}
		for itemRows.Next() {
			var it nfeSaidaItemRow
			if err := itemRows.Scan(
				&it.ID, &it.NItem, &it.CProd, &it.XProd, &it.NCM, &it.CEST,
				&it.CFOP, &it.CSTICMS, &it.CSTOrig, &it.CSTPIS, &it.CSTCOFINS,
				&it.VProd, &it.VBCICMS, &it.VICMS, &it.VBCST, &it.VST, &it.VIPI,
				&it.VBCPIS, &it.VPIS, &it.VBCCOFINS, &it.VCOFINS, &it.VIBS, &it.VCBS, &it.CClassTrib,
			); err != nil {
				log.Printf("NFeSaidaDetail item scan error: %v", err)
				continue
			}
			itens = append(itens, it)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"nfe":   row,
			"itens": itens,
		})
	}
}
