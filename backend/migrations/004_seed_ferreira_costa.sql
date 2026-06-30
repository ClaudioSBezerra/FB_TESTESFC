-- 004_seed_ferreira_costa.sql
-- Seed idempotente: Ambiente + Grupo + Empresa Ferreira Costa + Admin claudio_bezerra@hotmail.com
-- Adaptado de FB_APU04: 016_seed_default_environment + 021_ensure_admin_user + 024_ensure_master_link
--
-- NOTA: NÃO inclui INSERT em erp_bridge_config aqui — essa tabela é criada na migração 002
--       (Plan 04). Após a migração 002 ser executada, o seed do erp_bridge_config é feito
--       na migração correspondente do Plan 04.

DO $$
DECLARE
    v_env_id     UUID;
    v_group_id   UUID;
    v_company_id UUID;
    v_user_id    UUID;
BEGIN
    -- 1. Ambiente Ferreira Costa
    SELECT id INTO v_env_id FROM environments WHERE name = 'Ferreira Costa';
    IF v_env_id IS NULL THEN
        INSERT INTO environments (name, description)
        VALUES ('Ferreira Costa', 'Ambiente principal da Ferreira Costa')
        RETURNING id INTO v_env_id;
    END IF;

    -- 2. Grupo Ferreira Costa
    SELECT id INTO v_group_id FROM enterprise_groups
    WHERE name = 'Ferreira Costa' AND environment_id = v_env_id;
    IF v_group_id IS NULL THEN
        INSERT INTO enterprise_groups (environment_id, name)
        VALUES (v_env_id, 'Ferreira Costa')
        RETURNING id INTO v_group_id;
    END IF;

    -- 3. Empresa Ferreira Costa (sem cnpj — coluna não existe neste schema; ver 001_auth_hierarchy.sql)
    SELECT id INTO v_company_id FROM companies
    WHERE name = 'Ferreira Costa' AND group_id = v_group_id;
    IF v_company_id IS NULL THEN
        INSERT INTO companies (group_id, name, trade_name)
        VALUES (v_group_id, 'Ferreira Costa', 'Ferreira Costa')
        RETURNING id INTO v_company_id;
    END IF;

    -- 4. Admin claudio_bezerra@hotmail.com / senha 123456 (D-10)
    -- Hash bcrypt cost=14 retirado de 021_ensure_admin_user.sql do FB_APU04
    -- O hash '$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC' corresponde a '123456'
    SELECT id INTO v_user_id FROM users WHERE email = 'claudio_bezerra@hotmail.com';
    IF v_user_id IS NULL THEN
        INSERT INTO users (email, password_hash, full_name, role, is_verified)
        VALUES (
            'claudio_bezerra@hotmail.com',
            '$2a$14$Opb3Wt02JbSQbMLm.OQF8ObYr4UZh5h7S8KzCj1PfwLyjes6vFluC',
            'Claudio Bezerra (Admin)',
            'admin',
            true
        )
        RETURNING id INTO v_user_id;
    ELSE
        -- NÃO atualizar password_hash para não sobrescrever senha já trocada (CR-03)
        UPDATE users SET
            role        = 'admin',
            is_verified = true,
            full_name   = 'Claudio Bezerra (Admin)'
        WHERE email = 'claudio_bezerra@hotmail.com';
    END IF;

    -- 5. Vínculo user ↔ ambiente (idempotente via ON CONFLICT)
    INSERT INTO user_environments (user_id, environment_id, role)
    VALUES (v_user_id, v_env_id, 'admin')
    ON CONFLICT DO NOTHING;

    -- 6. Owner da empresa → admin (apenas se ainda não definido)
    UPDATE companies
    SET owner_id = v_user_id
    WHERE id = v_company_id AND owner_id IS NULL;

END $$;
