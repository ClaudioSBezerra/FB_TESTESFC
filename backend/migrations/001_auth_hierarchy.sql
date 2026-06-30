-- 001_auth_hierarchy.sql
-- Consolida: 013_create_environment_hierarchy + 015_create_auth_system
--            + 017_add_owner_to_companies + 018_add_role_to_users + 025_add_indexes_auth
-- NÃO inclui coluna removida em 023 do projeto base; schema começa sem ela

CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS enterprise_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES enterprise_groups(id) ON DELETE CASCADE,
    owner_id UUID,               -- adicionado direto (017); FK adicionada abaixo após users
    name VARCHAR(255) NOT NULL,
    trade_name VARCHAR(255),
    regime_tributario VARCHAR(50),
    inscricao_estadual VARCHAR(50),
    cnae_principal VARCHAR(10),
    segmento_economico VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
    -- SEM colunas removidas no projeto base; schema parte do estado final
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'admin',       -- adicionado direto (018)
    is_verified BOOLEAN DEFAULT FALSE,
    trial_ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- FK de companies.owner_id → users (adicionada depois que users existe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'companies_owner_id_fkey'
          AND table_name = 'companies'
    ) THEN
        ALTER TABLE companies
            ADD CONSTRAINT companies_owner_id_fkey
            FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS user_environments (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'admin',
    preferred_company_id UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, environment_id)
);

CREATE TABLE IF NOT EXISTS verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Índices de performance (025)
CREATE INDEX IF NOT EXISTS idx_enterprise_groups_env ON enterprise_groups(environment_id);
CREATE INDEX IF NOT EXISTS idx_companies_group ON companies(group_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_token ON verification_tokens(token);
CREATE INDEX IF NOT EXISTS idx_user_environments_user ON user_environments(user_id);
