-- Migration 005: Seed — linha erp_bridge_config para a empresa Ferreira Costa
-- Depende de: 002_erp_bridge.sql (tabela erp_bridge_config) e 004_seed_ferreira_costa.sql (empresa Ferreira Costa)
-- Ordenação alfabética garante: 002 < 004 < 005 — tabela e empresa já existem quando esta roda.
-- INSERT idempotente: ON CONFLICT (company_id) DO NOTHING

INSERT INTO erp_bridge_config (company_id)
SELECT id FROM companies WHERE name = 'Ferreira Costa'
ON CONFLICT (company_id) DO NOTHING;
