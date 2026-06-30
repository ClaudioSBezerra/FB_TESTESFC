# Phase 1: Foundation & Inherited Stack - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-30
**Phase:** 1-Foundation & Inherited Stack
**Areas discussed:** Estratégia de cópia, Migrações Postgres, Multi-tenant, Identidade/nomes

---

## Estratégia de cópia

### Como trazer o código do FB_APU04?
| Option | Description | Selected |
|--------|-------------|----------|
| Cherry-pick por dependência | Copiar só o necessário, resolvendo imports sob demanda | ✓ |
| Copiar tudo e podar | Clonar inteiro e remover módulos fora de escopo | |
| Reescrever enxuto | Usar FB_APU04 só como referência e reescrever | |

### main.go (1441 linhas) e registro de rotas
| Option | Description | Selected |
|--------|-------------|----------|
| Novo main.go enxuto | Registrar só as rotas dos módulos copiados | ✓ |
| Copiar main.go e comentar rotas | Trazer inteiro e desativar rotas fora de escopo | |

### Dependências Go (vendor/)
| Option | Description | Selected |
|--------|-------------|----------|
| go.mod limpo + go mod tidy | Remover deps não usadas, manter Oracle/Postgres | ✓ |
| Copiar go.mod + vendor inteiros | Build idêntico, mas carrega deps fiscais | |

### Shell do frontend
| Option | Description | Selected |
|--------|-------------|----------|
| Shell + só páginas no escopo | Layout/router/AuthContext/api client + páginas relevantes | ✓ |
| Copiar src inteiro e remover rotas | Trazer tudo e desativar 40+ páginas | |
| Você decide na pesquisa | Researcher mapeia o corte mínimo | |

**Notes:** Confirma a decisão "cópia seletiva" já registrada no PROJECT.md.

---

## Migrações Postgres

### Schema a partir das 149 migrações
| Option | Description | Selected |
|--------|-------------|----------|
| Schema inicial consolidado | 1-3 migrações novas enxutas (auth+hierarquia+users) | ✓ |
| Reaproveitar subconjunto na ordem | Copiar só as relevantes mantendo numeração | |
| Trazer todas as 149 | Paridade total, mas pesado e frágil | |

### Escopo do schema da Fase 1
| Option | Description | Selected |
|--------|-------------|----------|
| Só auth + tenancy agora | NFe saída fica para a Fase 2 | ✓ |
| Já incluir tabelas de NFe | Banco pronto, mas estrutura sem uso ainda | |

### Mecanismo de migração
| Option | Description | Selected |
|--------|-------------|----------|
| Mesmo runner do FB_APU04 | .sql numerados; researcher confirma a lib | ✓ |
| Você decide na pesquisa | Researcher recomenda manter ou trocar | |

---

## Multi-tenant

### Hierarquia ambiente→grupo→empresa→usuário na v1
| Option | Description | Selected |
|--------|-------------|----------|
| Manter modelo + seed única | Preservar schema/handlers, semear só Ferreira Costa | ✓ |
| Reduzir schema p/ empresa única | Enxugar removendo níveis | |
| Hard-code Ferreira Costa | Fixar no código, sem tabela real | |

### Criação de Ferreira Costa + admin
| Option | Description | Selected |
|--------|-------------|----------|
| Migração de seed | SQL idempotente, sobe pronto no docker compose up | ✓ |
| Setup/onboarding na 1ª execução | Tela/comando de bootstrap | |

### Credenciais do admin inicial
| Option | Description | Selected |
|--------|-------------|----------|
| Via variáveis de ambiente (.env) | E-mail/senha do .env, hash na seed | |
| Senha padrão + troca no 1º login | Senha conhecida com flag de redefinição | |
| (free text) | Usuário padrão claudio_bezerra@hotmail.com; demais a partir dele | ✓ |

**User's choice:** Admin padrão fixo `claudio_bezerra@hotmail.com`, senha `123456`, a partir do qual os demais usuários são cadastrados.
**Notes:** Registrado como default de desenvolvimento para ferramenta interna somente-leitura; trocável depois pela tela de gestão.

### Contexto de empresa (TEN-03) com empresa única
| Option | Description | Selected |
|--------|-------------|----------|
| Manter resolução herdada, default único | Mecanismo do FB_APU04, Ferreira Costa sempre; sem CompanySwitcher | ✓ |
| Middleware fixa empresa única | Injeta ID fixo ignorando hierarquia | |

---

## Identidade/nomes

### Identidade do projeto (module path, nome do app)
| Option | Description | Selected |
|--------|-------------|----------|
| Renomear para fb_testesfc | Module path/imports/nome próprios | ✓ |
| Manter nomes do FB_APU04 | Colar sem ajustar imports | |

### Docker-compose, portas e Postgres
| Option | Description | Selected |
|--------|-------------|----------|
| Portas/DB próprios, sem conflito | Rodar em paralelo ao FB_APU04 | ✓ |
| Reusar as mesmas portas do FB_APU04 | Simples, mas impede rodar os dois | |
| Você decide na pesquisa | Researcher propõe o mapeamento | |

### Escopo do ERP_BRIDGE na Fase 1
| Option | Description | Selected |
|--------|-------------|----------|
| Infra + tela de config + teste de conexão | Conecta, sem consultar prod/PRODB ainda | ✓ |
| Só copiar a infra, sem validar conexão | Validação fica para Fase 2 | |
| Você decide na pesquisa | Researcher avalia viabilidade sem credenciais | |

---

## Claude's Discretion

- Lib/abordagem exata do runner de migração — researcher confirma inspecionando o FB_APU04.
- Detalhe fino do corte de dependências do shell do frontend — researcher mapeia o grafo de componentes se necessário.
- Mapeamento concreto de portas/serviços/volumes do docker-compose.

## Deferred Ideas

- Tabelas de NFe saída (cabeçalho/itens/impostos) → Fase 2.
- Lookup de grupo fiscal em prod/PRODB → Fase 2.
- CompanySwitcher / multi-tenant completo (MTN-01) → v2.
- Endurecer credenciais do admin (via .env / troca obrigatória no 1º login) → considerado, adiável.
