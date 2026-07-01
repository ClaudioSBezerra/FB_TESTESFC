package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// errSemGrupoFiscal sinaliza que o produto não foi encontrado em prod/PRODB —
// não é um erro fatal (ERP-03): o item deve ser marcado como "sem_grupo_fiscal"
// e o processamento dos demais itens deve seguir normalmente.
var errSemGrupoFiscal = errors.New("produto não encontrado em prod/PRODB")

// codEmpresaPorCNPJRaiz resolve o cod_empresa fixo de PRODB a partir da raiz
// (8 primeiros dígitos) do CNPJ do emitente (Assumption A6 / Open Question #5
// do 02-RESEARCH.md — cod_empresa é um valor pequeno e estático por filial,
// sem tabela nova; duas filiais compartilham a UF "PE").
//
// PENDENTE (checkpoint humano desta fase, passo 9 do <how-to-verify> do
// 02-02-PLAN.md): esta execução não tem acesso a um XML real nem ao Oracle da
// Ferreira Costa para confirmar a raiz de CNPJ de cada filial. A raiz abaixo
// (10230480) é a única confirmada nesta rodada — é exatamente o
// pCnpjEmpresa usado no exemplo do script de teste do pacote fiscal
// (/tmp/11_Script_Teste_..., pUFOrigem=PE), tratada aqui como Recife/PE
// (cod_empresa=2, conforme RESEARCH.md A6). A raiz da filial Garanhuns/PE
// (cod_empresa=1) NÃO está confirmada — adicionar ao mapa assim que
// confirmada contra o Oracle real. Até lá, notas emitidas por uma filial não
// mapeada retornam erro explícito por item (nunca um cod_empresa adivinhado).
var codEmpresaPorCNPJRaiz = map[string]int{
	"10230480": 2, // Ferreira Costa — Recife/PE (fonte: script de teste do pacote fiscal)
}

// resolveCodEmpresa deriva o cod_empresa de PRODB a partir do CNPJ do
// emitente da nota. Retorna erro explícito (nunca um valor adivinhado) quando
// a raiz do CNPJ não está mapeada — cada item da nota herda esse erro e é
// marcado como "error" sem abortar os demais itens (isolamento por item).
func resolveCodEmpresa(emitCNPJ, emitUF string) (int, error) {
	digits := onlyDigits(emitCNPJ)
	if len(digits) < 8 {
		return 0, fmt.Errorf("CNPJ do emitente inválido para resolução de cod_empresa")
	}
	raiz := digits[:8]
	if cod, ok := codEmpresaPorCNPJRaiz[raiz]; ok {
		return cod, nil
	}
	return 0, fmt.Errorf("cod_empresa não mapeado para a filial do emitente (CNPJ raiz %s, UF %s) — atualizar codEmpresaPorCNPJRaiz em fiscal_group_lookup.go", raiz, emitUF)
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// lookupGrupoFiscal consulta prod/PRODB (mesma instância Oracle do
// FCCORP_BKP) e retorna o grupo fiscal, a origem e o NCM do produto.
// Query confirmada pelo usuário (02-RESEARCH.md, Assumption A1):
//
//	SELECT pb.grupo_fiscal, p.especial AS origem, p.ncm
//	FROM prodb pb, prod p
//	WHERE p.codigo = pb.codigo AND pb.codigo = :codigoProduto AND pb.cod_empresa = :codEmpresa
//
// O filtro por cod_empresa é obrigatório (A6) — sem ele, o mesmo código de
// produto pode existir em mais de uma filial com grupo fiscal diferente.
// sql.ErrNoRows é traduzido para errSemGrupoFiscal (não fatal — ERP-03).
func lookupGrupoFiscal(ctx context.Context, oracleDB *sql.DB, codigoProduto string, codEmpresa int) (grupoFiscal, origem, ncm string, err error) {
	const query = `
		SELECT pb.grupo_fiscal, p.especial AS origem, p.ncm
		FROM prodb pb, prod p
		WHERE p.codigo = pb.codigo
		  AND pb.codigo = :codigoProduto
		  AND pb.cod_empresa = :codEmpresa`

	var grupoFiscalNS, origemNS, ncmNS sql.NullString
	row := oracleDB.QueryRowContext(ctx, query,
		sql.Named("codigoProduto", codigoProduto),
		sql.Named("codEmpresa", codEmpresa),
	)
	if scanErr := row.Scan(&grupoFiscalNS, &origemNS, &ncmNS); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return "", "", "", errSemGrupoFiscal
		}
		// Nunca propagar scanErr.Error() bruto ao cliente — o driver go-ora
		// pode incluir detalhes de conexão na mensagem (T-02-06). O chamador
		// (fiscal_execution.go) sanitiza antes de persistir/expor.
		return "", "", "", scanErr
	}
	return grupoFiscalNS.String, origemNS.String, ncmNS.String, nil
}
