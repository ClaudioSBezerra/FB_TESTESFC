-- 007_nfe_saidas_itens.sql
-- Itens da NF-e de saída — valores ESPERADOS (gabarito) por item, vindos do XML.
-- Analog: FB_APU04/backend/migrations/075_create_nfe_itens_tables.sql (seção
-- nfe_saidas_itens), com adição de v_bc_st/v_st a nível de item (decisão do
-- planner — RESEARCH.md aponta lacuna para comparação ICMS-ST item a item na Fase 3).

CREATE TABLE IF NOT EXISTS nfe_saidas_itens (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    nfe_id          UUID        NOT NULL REFERENCES nfe_saidas(id) ON DELETE CASCADE,
    company_id      UUID        NOT NULL,
    n_item          SMALLINT    NOT NULL,
    c_prod          VARCHAR(60),
    x_prod          VARCHAR(120) NOT NULL,
    ncm             VARCHAR(8),
    cest            VARCHAR(7),
    cfop            VARCHAR(4),
    cst_icms        VARCHAR(3),
    cst_orig        VARCHAR(1),
    cst_pis         VARCHAR(2),
    cst_cofins      VARCHAR(2),
    v_prod          NUMERIC(15,2) DEFAULT 0,
    v_total_item    NUMERIC(15,2) DEFAULT 0,
    v_bc_icms       NUMERIC(15,2) DEFAULT 0,
    v_icms          NUMERIC(15,2) DEFAULT 0,
    v_bc_st         NUMERIC(15,2) DEFAULT 0,
    v_st            NUMERIC(15,2) DEFAULT 0,
    v_ipi           NUMERIC(15,2) DEFAULT 0,
    v_bc_pis        NUMERIC(15,2) DEFAULT 0,
    v_pis           NUMERIC(15,2) DEFAULT 0,
    v_bc_cofins     NUMERIC(15,2) DEFAULT 0,
    v_cofins        NUMERIC(15,2) DEFAULT 0,
    v_ibs           NUMERIC(15,2) DEFAULT 0,
    v_cbs           NUMERIC(15,2) DEFAULT 0,
    cclasstrib      VARCHAR(20),

    CONSTRAINT uq_nfe_saidas_itens_nfe_item UNIQUE (nfe_id, n_item)
);

CREATE INDEX IF NOT EXISTS idx_nfe_saidas_itens_company_ncm ON nfe_saidas_itens(company_id, ncm);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_itens_nfe_id      ON nfe_saidas_itens(nfe_id);
