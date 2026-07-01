-- 008_fiscal_execution_items.sql
-- Resultado calculado pelo pacote fiscal (PKG_FISCAL_FCTAX.calcula_imposto_produto),
-- um registro por item de nfe_saidas_itens. Modelo híbrido: colunas dedicadas para
-- os campos usados na comparação visual da Fase 3 + full_result JSONB para o
-- retorno completo (~88 campos, incluindo o bloco da Reforma Tributária).
-- Fonte do schema: 02-RESEARCH.md (linhas 442-481) / 02-PATTERNS.md (linhas 228-259).
--
-- Status distinto de erro (ERP-03 vs FIS-03):
--   sem_grupo_fiscal = lookup prod/PRODB não encontrou o produto (ERP-03)
--   error            = falha na chamada do pacote fiscal em si (FIS-03)

CREATE TABLE IF NOT EXISTS fiscal_execution_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id          UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    nfe_item_id         UUID NOT NULL REFERENCES nfe_saidas_itens(id) ON DELETE CASCADE,

    -- Status de execução (isolamento de erro por item — FIS-03/ERP-03)
    status              TEXT NOT NULL DEFAULT 'pending', -- pending | ok | error | sem_grupo_fiscal
    error_message       TEXT,
    executed_at         TIMESTAMPTZ,

    -- Parâmetros de entrada efetivamente usados (auditoria — o que foi enviado ao pacote)
    grupo_fiscal_codigo TEXT,             -- pCodigoGrupoFiscal resolvido via prod/PRODB
    input_params        JSONB,            -- snapshot dos 23 parâmetros de entrada enviados

    -- Campos usados na comparação visual da Fase 3 (colunas dedicadas — acesso rápido/indexável)
    base_calculo_icms          NUMERIC(15,2),  -- result.BaseCalculo
    valor_icms                 NUMERIC(15,2),  -- result.ValorImposto (quando TipoImposto = ICMS)
    base_substituicao          NUMERIC(15,2),  -- result.BaseSubstituicao
    valor_substituicao         NUMERIC(15,2),  -- result.ValorSubstituicao
    base_calculo_pis           NUMERIC(15,2),  -- result.BaseCalculoPIS
    valor_pis                  NUMERIC(15,2),  -- result.ValorPIS
    base_calculo_cofins        NUMERIC(15,2),  -- result.BaseCalculoCOFINS
    valor_cofins                NUMERIC(15,2), -- result.ValorCOFINS
    percentual_difal           NUMERIC(7,4),   -- result.PercentualDifal
    valor_icms_partilha_destino NUMERIC(15,2), -- result.ValorIcmsPartilhaDestino (DIFAL)
    valor_icms_pobreza         NUMERIC(15,2),  -- result.ValorIcmsPobreza (FCP)

    -- Retorno completo (~88 campos) para auditoria/depuração e campos da Reforma
    -- Tributária (IBS UF, IBS Município, CBS) ainda não mapeados para colunas dedicadas
    full_result         JSONB NOT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_fiscal_execution_item UNIQUE (nfe_item_id)
);

CREATE INDEX IF NOT EXISTS idx_fiscal_execution_status ON fiscal_execution_items(company_id, status);
CREATE INDEX IF NOT EXISTS idx_fiscal_execution_nfe_item ON fiscal_execution_items(nfe_item_id);
