-- 006_nfe_saidas.sql
-- Cabeçalho da NF-e de saída — valores ESPERADOS (gabarito) vindos do XML.
-- Analog: FB_APU04/backend/migrations/058_create_nfe_saidas.sql (adaptado ao
-- schema enxuto do FB_TESTESFC — sem colunas de fornecedor/CTe/SPED).

CREATE TABLE IF NOT EXISTS nfe_saidas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    chave_nfe       VARCHAR(44) NOT NULL,
    modelo          SMALLINT NOT NULL,
    serie           VARCHAR(3),
    numero_nfe      VARCHAR(9),
    data_emissao    DATE NOT NULL,
    mes_ano         VARCHAR(7) NOT NULL,
    nat_op          VARCHAR(60),

    -- Emitente (filial Ferreira Costa)
    emit_cnpj       VARCHAR(14) NOT NULL,
    emit_nome       VARCHAR(60),
    emit_uf         VARCHAR(2),
    emit_municipio  VARCHAR(60),

    -- Destinatário (cliente)
    dest_cnpj_cpf   VARCHAR(14),
    dest_nome       VARCHAR(60),
    dest_uf         VARCHAR(2),
    dest_c_mun      VARCHAR(7),

    -- ICMSTot — totais fiscais esperados (do XML)
    v_bc            NUMERIC(15,2) DEFAULT 0,
    v_icms          NUMERIC(15,2) DEFAULT 0,
    v_icms_deson    NUMERIC(15,2) DEFAULT 0,
    v_fcp           NUMERIC(15,2) DEFAULT 0,
    v_bc_st         NUMERIC(15,2) DEFAULT 0,
    v_st            NUMERIC(15,2) DEFAULT 0,
    v_fcp_st        NUMERIC(15,2) DEFAULT 0,
    v_fcp_st_ret    NUMERIC(15,2) DEFAULT 0,
    v_prod          NUMERIC(15,2) DEFAULT 0,
    v_frete         NUMERIC(15,2) DEFAULT 0,
    v_seg           NUMERIC(15,2) DEFAULT 0,
    v_desc          NUMERIC(15,2) DEFAULT 0,
    v_ii            NUMERIC(15,2) DEFAULT 0,
    v_ipi           NUMERIC(15,2) DEFAULT 0,
    v_ipi_devol     NUMERIC(15,2) DEFAULT 0,
    v_pis           NUMERIC(15,2) DEFAULT 0,
    v_cofins        NUMERIC(15,2) DEFAULT 0,
    v_outro         NUMERIC(15,2) DEFAULT 0,
    v_nf            NUMERIC(15,2) DEFAULT 0,

    -- IBSCBSTot — Reforma Tributária
    v_bc_ibs_cbs      NUMERIC(15,2),
    v_ibs_uf          NUMERIC(15,2),
    v_ibs_mun         NUMERIC(15,2),
    v_ibs             NUMERIC(15,2),
    v_cred_pres_ibs   NUMERIC(15,2),
    v_cbs             NUMERIC(15,2),
    v_cred_pres_cbs   NUMERIC(15,2),

    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_nfe_saidas_company_chave UNIQUE (company_id, chave_nfe)
);

CREATE INDEX IF NOT EXISTS idx_nfe_saidas_company_mes  ON nfe_saidas(company_id, mes_ano);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_company_data ON nfe_saidas(company_id, data_emissao);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_emit_cnpj    ON nfe_saidas(company_id, emit_cnpj);
CREATE INDEX IF NOT EXISTS idx_nfe_saidas_dest_c_mun   ON nfe_saidas(company_id, dest_c_mun);
