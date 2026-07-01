-- 009_nfe_saidas_itens_desconto.sql
-- Adiciona v_desc (desconto por item) a nfe_saidas_itens.
--
-- DEVIATION (Rule 2 - Auto-add missing critical functionality, Plano 02-02):
-- o XML de NF-e já traz vDesc por item (prod.VDesc, ver nfe_saidas.go) e o
-- pacote fiscal (PKG_FISCAL_FCTAX.calcula_imposto_produto) exige pDesconto por
-- item para recalcular a base de ICMS corretamente — sem esse valor persistido,
-- a comparação esperado-vs-calculado da Fase 3 ficaria estruturalmente incorreta
-- (base de cálculo divergente por desconsiderar o desconto do item). O Plano
-- 02-01 parseava vDesc mas não o persistia (lacuna do schema original,
-- reproduzida do FB_APU04). Coluna aditiva, não quebra dados já importados
-- (DEFAULT 0, reimportação idempotente via ON CONFLICT já recalcula o valor).

ALTER TABLE nfe_saidas_itens ADD COLUMN IF NOT EXISTS v_desc NUMERIC(15,2) DEFAULT 0;
