# Fase 2: Import Pipeline & Fiscal Execution — Pesquisa

**Pesquisado em:** 2026-07-01 (atualizado no mesmo dia com esclarecimentos do usuário sobre schema Oracle real)
**Domínio:** Parsing de XML NFe (SEFAZ) + driver Oracle puro-Go (go-ora) para PL/SQL com Object Types + modelagem Postgres para comparação fiscal
**Confiança:** HIGH (alto para stack herdada do FB_APU04; MEDIUM-HIGH para a chamada Oracle via go-ora — padrão claro, mas ainda não testado contra a instância real; ALTO para o lookup `prod`/`PRODB`, com schema real confirmado pelo usuário — ver Assumptions Log)

## Summary

Esta fase tem duas partes de risco muito diferentes. A primeira — importação de XML de NFe de saída e persistência no Postgres — é **baixo risco**: o FB_APU04 já tem um parser maduro (`encoding/xml` com structs Go simples, sem biblioteca externa de NFe) e um schema de referência completo em `nfe_saidas` / `nfe_saidas_itens`, prontos para reaproveitar quase 1:1. A segunda — obter o grupo fiscal via Oracle (`prod`/`PRODB`) e executar `PKG_FISCAL_FCTAX.calcula_imposto_produto` no `FCCORP_BKP` — é **risco técnico real e não resolvido em nenhum dos dois repos**: nem o FB_APU04 (apesar de "maduro" e de ter um módulo inteiro chamado ERP_BRIDGE) nem a Fase 1 do FB_TESTESFC têm qualquer código Go que efetivamente dispare uma query ou um bloco PL/SQL contra Oracle além de um `PingContext` de teste de conexão. O ERP_BRIDGE do FB_APU04 é, na verdade, um **padrão de integração via daemon externo que envia XMLs por HTTP** — não um cliente Go que consulta Oracle diretamente. Portanto, a Fase 2 não tem "código de referência" a copiar para a parte Oracle: ela precisa ser construída do zero, usando o driver `github.com/sijms/go-ora/v2` já presente no `go.mod` desde a Fase 1.

O ponto decisivo descoberto nesta pesquisa: a função `PKG_FISCAL_FCTAX.calcula_imposto_produto` usa **notação de parâmetro nomeado** (`pCnpjEmpresa => :pCnpjEmpresa`, etc.) — e notação nomeada **não pode ser usada ao chamar uma function a partir de SQL puro** (`SELECT ... FROM dual`), apenas dentro de um bloco PL/SQL (anônimo, procedure ou function). Isso é confirmado pelo próprio script de teste fornecido pelo usuário, que usa exatamente um bloco `declare...begin...end;` — nunca um `SELECT`. Isso elimina a rota "mais simples" (registrar o Object Type via `go_ora.RegisterType` e fazer `db.QueryRow("SELECT fn(...) FROM dual").Scan(&struct)`) como opção viável para esta função específica, e aponta para o padrão que o próprio script já demonstra: **bloco PL/SQL anônimo com bind variables**, com os ~65 campos de saída do objeto **"achatados" em parâmetros OUT escalares individuais** (`sql.Out`/`go_ora.Out`) dentro do bloco, em vez de tentar mapear o Object Type Oracle inteiro como uma única struct Go.

**Primary recommendation:** Não tentar mapear `PKG_FISCAL_FCTAX.RDADOS_FISCAIS_PRODUTO` como um Oracle Object Type registrado via `go_ora.RegisterType`. Em vez disso, montar em Go uma string de bloco PL/SQL anônimo idêntica em estrutura ao script de teste (`declare result <tipo>; begin result := PKG_FISCAL_FCTAX.calcula_imposto_produto(<23 params nomeados>); :out1 := result.Campo1; :out2 := result.Campo2; ... end;`), com todos os ~65 campos de saída atribuídos a variáveis bind individuais de tipo escalar (NUMBER/VARCHAR2), passadas como `sql.Out`/`go_ora.Out` na chamada Go via `database/sql`. Isso evita a complexidade e as limitações documentadas de Object Types com go-ora, ao custo de uma string SQL grande e verbosa — gerada uma vez a partir da lista de campos do script de teste, não escrita à mão repetidamente.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Upload e parse de XML NFe | API/Backend (Go) | — | Parsing síncrono no handler de upload, reaproveitado do FB_APU04 |
| Persistência de nota/itens esperados (XML) | Database (Postgres) | API/Backend | Fonte da verdade "gabarito"; schema já modelado no FB_APU04 |
| Lookup de grupo fiscal (`prod`/`PRODB`) | API/Backend (Go, via go-ora) | Database (Oracle, somente leitura) | Consulta pontual por item, feita a partir do backend Go, nunca do frontend |
| Execução do pacote fiscal (`FCCORP_BKP`) | API/Backend (Go, via go-ora) | Database (Oracle, somente leitura) | Bloco PL/SQL anônimo disparado item a item pelo backend |
| Persistência do resultado calculado | Database (Postgres) | API/Backend | Tabela nova vinculada ao item da nota |
| Isolamento de erro por item | API/Backend (Go) | — | Cada chamada Oracle roda em unidade de trabalho isolada (goroutine/loop com recover + captura de erro), nunca aborta o lote |
| Configuração de credenciais Oracle | API/Backend (Go) | Frontend (tela ERP_BRIDGE herdada) | Já implementado na Fase 1 — reaproveitado sem alteração |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `github.com/sijms/go-ora/v2` | v2.9.0 [VERIFIED: proxy.golang.org] | Driver Oracle puro-Go para `database/sql` | Já aprovado e em uso desde a Fase 1 (T-04-04, checkpoint humano); único driver Oracle da stack, sem CGO |
| `encoding/xml` (stdlib) | Go 1.24 stdlib | Parse do XML de NFe | Já usado no FB_APU04 em `nfe_saidas.go`/`cte_entradas.go`; nenhuma lib externa de NFe é usada no projeto base — mapeamento manual via structs com tags `xml:"..."` |
| `golang.org/x/text` (charmap/transform) | v0.34.0 [CITED: go.mod do FB_APU04] | Conversão de encoding (windows-1252/ISO-8859-1 → UTF-8) de XMLs de NFe antigos | Usado em `nfeCharsetReader` no FB_APU04; XMLs de NFe legados frequentemente não estão em UTF-8 |
| `archive/zip` (stdlib) | Go 1.24 stdlib | Extração de XMLs de arquivos ZIP enviados em lote | Já usado em `xml_upload.go` (`extractXMLsFromZip`), com mitigações anti-ZIP-bomb |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|--------------|
| `github.com/nwaples/rardecode/v2` | v2.2.2 [CITED: go.mod do FB_APU04] | Extração de XMLs de arquivos `.rar` | Somente se a tela de import herdada aceitar `.rar` — confirmar se XML-01 exige isso ou só ZIP/XML avulso |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Bloco PL/SQL anônimo com OUT escalares (recomendado) | `go_ora.RegisterType` + `SELECT fn(...) FROM dual` | **Não aplicável** — a função usa notação nomeada (`=>`), que Oracle proíbe fora de contexto PL/SQL. Ver seção "Pitfall 1". |
| `go-ora` (puro Go) | `github.com/godror/godror` (cgo, wrapper ODPI-C) | godror tem suporte mais maduro a Object Types complexos (incluindo LOBs aninhados), mas exige CGO_ENABLED=1 e Oracle Instant Client instalado no container — quebra a decisão já tomada na Fase 1 de manter o driver puro-Go (T-04-04). Não recomendado trocar agora. |
| Parsing manual de XML com structs Go | Biblioteca de terceiros para NFe (ex.: `unidoc`, `nfe-go` de terceiros) | Nenhuma é padrão de mercado nem usada no FB_APU04; adicionaria dependência não auditada para um problema já resolvido com `encoding/xml` puro |

**Installation:**
```bash
# go-ora já está no go.mod desde a Fase 1 — nenhuma instalação nova necessária
cd backend && go build ./...
```

**Version verification:**
```bash
curl -s https://proxy.golang.org/github.com/sijms/go-ora/v2/@latest
# {"Version":"v2.9.0","Time":"2025-06-09T21:19:12Z", ...} — confirmado nesta pesquisa em 2026-07-01
```

## Package Legitimacy Audit

Nenhum pacote **novo** é instalado nesta fase. `go-ora/v2` já foi auditado e aprovado na Fase 1 (T-04-04, `01-04-SUMMARY.md`) via `proxy.golang.org` + checkpoint humano bloqueante. `slopcheck` não cobre o ecossistema Go modules (é orientado a pip/npm/cargo) — rodá-lo contra `go-ora` produziu um falso positivo esperado (tentativa de resolver como pacote PyPI). A verificação de legitimidade para Go modules foi feita via `proxy.golang.org`, que é a fonte de verdade oficial do Go.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|--------------|-----------|--------------|
| `github.com/sijms/go-ora/v2` | Go modules (proxy.golang.org) | v2.9.0 publicado 2025-06-09; projeto ativo desde 2019 | N/A (Go modules não expõe contagem de downloads) | github.com/sijms/go-ora | N/A (fora do escopo do slopcheck; verificado via proxy.golang.org) | Approved (já aprovado na Fase 1, T-04-04) |

**Packages removed due to slopcheck [SLOP] verdict:** nenhum (nenhum pacote novo nesta fase)
**Packages flagged as suspicious [SUS]:** nenhum

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────┐
                    │  Frontend (ImportarXMLs)    │
                    │  upload de 1..N XMLs/ZIP    │
                    └──────────────┬──────────────┘
                                   │ POST /api/xml-upload (multipart)
                                   ▼
        ┌──────────────────────────────────────────────────┐
        │  Backend Go — handler de upload                  │
        │  1. extractXMLsFromZip/Rar (se aplicável)         │
        │  2. parseNFeXML (encoding/xml) por arquivo        │
        │  3. valida tpNF=1 (saída), modelo 55/65           │
        └──────────────┬─────────────────────────────────────┘
                        │ tx.Begin()
                        ▼
        ┌──────────────────────────────────────────────────┐
        │  Postgres — nfe_saidas + nfe_saidas_itens         │
        │  (persiste cabeçalho + itens = valores ESPERADOS) │
        └──────────────┬─────────────────────────────────────┘
                        │ (assíncrono ou síncrono, por item)
                        ▼
        ┌──────────────────────────────────────────────────┐
        │  Backend Go — pipeline de execução fiscal         │
        │  Para cada item da nota:                          │
        │   a) lookup grupo fiscal em Oracle prod/PRODB     │
        │      (mesma conexão/instância do FCCORP_BKP)      │
        │   b) monta bloco PL/SQL anônimo com 23 params IN  │
        │      + ~65 binds OUT escalares (achatados)         │
        │   c) chama PKG_FISCAL_FCTAX.calcula_imposto_produto│
        │      via db.Exec(anonBlock, ...binds)             │
        │   d) captura erro por item (timeout, grupo fiscal  │
        │      ausente, erro PL/SQL) — NUNCA aborta o lote   │
        └──────────────┬─────────────────────────────────────┘
                        │
              ┌─────────┴─────────┐
              ▼                   ▼
   ┌────────────────────┐  ┌────────────────────────┐
   │ Oracle (leitura)    │  │ Postgres                │
   │ prod / PRODB        │  │ fiscal_execution_items  │
   │ FCCORP_BKP          │  │ (resultado calculado +  │
   │ (mesma instância)   │  │  status de erro/ok)     │
   └────────────────────┘  └────────────────────────┘
```

### Recommended Project Structure

```
backend/
├── handlers/
│   ├── xml_upload.go          # reaproveitado do FB_APU04 (parse + persistência do gabarito)
│   ├── nfe_saidas.go          # reaproveitado do FB_APU04 (structs XML, consulta)
│   ├── fiscal_group_lookup.go # NOVO — consulta prod/PRODB via go-ora, retorna grupo fiscal
│   ├── fiscal_execution.go    # NOVO — monta e executa o bloco PL/SQL anônimo, persiste resultado
│   └── erp_bridge.go          # já existe (Fase 1) — fonte da conexão Oracle (DSN/credenciais)
├── migrations/
│   ├── 006_nfe_saidas.sql          # NOVO — adaptado de 058/075 do FB_APU04
│   ├── 007_fiscal_execution.sql    # NOVO — tabela de resultado calculado + status de erro
└── services/
    └── oracle_fiscal.go        # NOVO — helper que monta a string do bloco PL/SQL a partir de um template
```

### Pattern 1: Bloco PL/SQL anônimo com OUT escalares "achatados"

**What:** Em vez de tentar mapear o Object Type `RDADOS_FISCAIS_PRODUTO` como uma struct Go registrada via `go_ora.RegisterType`, o bloco PL/SQL declara a variável do tipo objeto **dentro do próprio bloco** (como o script de teste faz), chama a função com notação nomeada, e então atribui cada campo do objeto a uma bind variable de saída escalar (`:out1 := result.TipoImposto; :out2 := result.AliquotaImposto; ...`).

**When to use:** Sempre que a function/procedure Oracle usar notação de parâmetro nomeado (`=>`) e/ou retornar um Object Type não trivial — ambos os casos aqui.

**Example:**
```go
// Source: padrão inferido do script de teste fornecido pelo usuário
// (/tmp/11_Script_Teste_Pacote_FCTAX_1S_Reforma_Tributaria.TST) +
// documentação oficial do go-ora (github.com/sijms/go-ora/blob/master/README.md)
const calculaImpostoBlock = `
declare
  result PKG_FISCAL_FCTAX.RDADOS_FISCAIS_PRODUTO;
begin
  result := PKG_FISCAL_FCTAX.calcula_imposto_produto(
    pCnpjEmpresa => :pCnpjEmpresa,
    pUFOrigem => :pUFOrigem,
    pUFDestino => :pUFDestino,
    pTipoContribuinte => :pTipoContribuinte,
    pTipoCentroFiscal => :pTipoCentroFiscal,
    pTipoOperacao => :pTipoOperacao,
    pEntradaSaida => :pEntradaSaida,
    pProduto => :pProduto,
    pCodigoGrupoFiscal => :pCodigoGrupoFiscal,
    pCnpjExcecao => :pCnpjExcecao,
    pIndicadorServico => :pIndicadorServico,
    pPrecoTotal => :pPrecoTotal,
    pDespesas => :pDespesas,
    pDesconto => :pDesconto,
    pIPI => :pIPI,
    pAliquotaSimplesNacional => :pAliquotaSimplesNacional,
    FornecedorSimplesNacional => :pFornecedorSimplesNacional,
    pTipoIsencaoPedidoBonificado => :pTipoIsencaoPedidoBonificado,
    pCFOPOperacao => :pCFOPOperacao,
    pTipoContribuinteSecundario => :pTipoContribuinteSecundario,
    pSimulacaoCalculo => :pSimulacaoCalculo,
    pDataReferenciaFiscal => :pDataReferenciaFiscal,
    pCodigoIbge => :pCodigoIbge
  );

  :oTipoImposto := result.TipoImposto;
  :oAliquotaImposto := result.AliquotaImposto;
  :oBaseCalculo := result.BaseCalculo;
  -- ... repetir para os ~65 campos de saída documentados no script de teste ...
  :oIdRegraCalculoCbs := result.IdRegraCalculoCbs;
end;`

// Chamada via database/sql
_, err := db.ExecContext(ctx, calculaImpostoBlock,
    sql.Named("pCnpjEmpresa", cnpjEmpresa),
    sql.Named("pUFOrigem", ufOrigem),
    // ... 21 params de entrada restantes ...
    sql.Named("oTipoImposto", sql.Out{Dest: &out.TipoImposto}),
    sql.Named("oAliquotaImposto", sql.Out{Dest: &out.AliquotaImposto}),
    // ... ~63 OUT params restantes, cada um foi tipado como
    //     go_ora.Out{Dest: &campo, Size: N} quando string, ou sql.Out{Dest: &campo} quando numérico ...
)
```

**Nota de implementação:** gerar essa string de bloco PL/SQL e a lista de `sql.Named`/`sql.Out` **programaticamente** a partir de uma tabela de metadados (nome do campo Oracle → nome do bind → tipo Go), extraída diretamente do script de teste (linhas 64-158). Não escrever os ~65 binds à mão em múltiplos lugares — um único slice/struct de definição de campos deve gerar tanto a string SQL quanto os destinos de scan, para evitar erro de digitação em nomes de 65 campos.

### Pattern 2: Parsing de XML de NFe via `encoding/xml` com structs simples

**What:** O FB_APU04 não usa nenhuma biblioteca de NFe — mapeia o XML manualmente com structs Go e tags `xml:"..."`, incluindo tratamento de encoding legado.

**When to use:** Reaproveitar tal qual — copiar os structs `nfeProc`, `nfe`, `infNFe`, `det`, `prod`, `detImposto`, `ide`, `emit`, `dest`, `total`, `icmsTot`, `ibsCbsTot` de `nfe_saidas.go` (linhas 38-214) e as funções `parseNFeXML`, `toDecimal`, `toNullDecimal`, `nfeCharsetReader` de `xml_upload.go`.

**Example:**
```go
// Source: FB_APU04/backend/handlers/nfe_saidas.go:76-89
type det struct {
    NItem string     `xml:"nItem,attr"`
    Prod  prod       `xml:"prod"`
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
```

### Anti-Patterns to Avoid

- **Tentar `go_ora.RegisterType` + `SELECT function(...) FROM dual` para `calcula_imposto_produto`:** não vai compilar/executar no Oracle porque a função usa notação nomeada, proibida fora de PL/SQL. Descoberto e confirmado nesta pesquisa lendo o script de teste + documentação Oracle sobre named notation.
- **Chamar a função Oracle de dentro de uma goroutine sem limite de concorrência:** Oracle tem limite de conexões/sessões; abrir uma goroutine por item de um lote de milhares de itens pode saturar o pool de conexões Oracle. Usar um semáforo (`chan struct{}` com capacidade limitada, ex.: 5-10) ou processar sequencialmente por nota.
- **Assumir que erro de um item derruba a transação Postgres inteira:** cada chamada Oracle + persistência do resultado deve ser sua própria unidade de trabalho (commit por item ou por nota), nunca uma transação Postgres única para o lote inteiro — senão um erro de Oracle em um item reverteria os itens já processados com sucesso.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Parsing de estrutura NFe (schema SEFAZ) | Parser XML genérico com XPath dinâmico | `encoding/xml` + structs tipados explícitos (padrão já usado no FB_APU04) | O schema da NFe é estável e conhecido; structs tipados são mais seguros contra XXE e mais fáceis de auditar que XPath dinâmico |
| Conversão de moeda BR (vírgula decimal) do script de teste (`3849,9`) | Parser de string customizado ad-hoc | Confirmar formato: valores do XML da NFe usam ponto decimal (`vProd` etc., padrão SEFAZ); já existe `toDecimal`/`toNullDecimal` no FB_APU04 para isso. O formato com vírgula (`3849,9`) aparece **apenas nos valores de exemplo do script de teste PL/SQL Developer** (formato local pt-BR do editor), não no XML em si — não confundir os dois formatos ao montar os parâmetros de entrada da função fiscal | Evita bug sutil de parsing de decimal duplicado/divergente entre dois formatos de origem diferentes |
| Retry/circuit breaker para chamadas Oracle instáveis | Lógica de retry customizada espalhada pelos handlers | Um único wrapper `callFiscalPackage(ctx, params) (FiscalResult, error)` com timeout de contexto e no máximo 1-2 retries em erro de conexão (não em erro de negócio/PL/SQL) | Centraliza a política de resiliência; evita que cada chamador reimplemente sua própria lógica de retry de forma inconsistente |

**Key insight:** O maior risco de "hand-roll" nesta fase não é reinventar uma lib existente — é reinventar, campo a campo, o mapeamento de 65 saídas de forma manual e propensa a erro de digitação em múltiplos lugares do código. Centralizar essa definição de campos em uma única fonte de verdade (slice/struct) é o principal cuidado de design.

## Runtime State Inventory

Não aplicável — esta é uma fase de construção greenfield (nova funcionalidade), não um rename/refactor/migração de dados existentes.

## Common Pitfalls

### Pitfall 1: Notação nomeada (`=>`) não funciona fora de bloco PL/SQL

**What goes wrong:** Uma tentativa de chamar `SELECT PKG_FISCAL_FCTAX.calcula_imposto_produto(pCnpjEmpresa => :x, ...) FROM dual` falha com erro de sintaxe Oracle (PLS-00306 ou ORA-06553, dependendo do contexto).
**Why it happens:** Oracle permite notação nomeada apenas dentro de PL/SQL (blocos anônimos, procedures, functions, packages) — nunca em SQL puro (SELECT/CALL). Isso é uma regra documentada da linguagem, não uma limitação do driver.
**How to avoid:** Sempre chamar via bloco PL/SQL anônimo (`declare...begin...end;`), replicando a estrutura do script de teste fornecido pelo usuário.
**Warning signs:** Erro de compilação PL/SQL ao tentar `SELECT function(...) FROM dual` com `=>` nos parâmetros.

### Pitfall 2: go-ora não escaneia Object Types complexos automaticamente sem `RegisterType` — e `RegisterType` não resolve este caso

**What goes wrong:** Achar que basta declarar uma struct Go com tags `udt:"..."` e o driver vai "simplesmente funcionar" para qualquer function que retorne Object Type.
**Why it happens:** `RegisterType`/`RegisterTypeWithOwner` funcionam para tipos acessíveis via `SELECT` (função standalone chamável em SQL, com parâmetros posicionais). Como `calcula_imposto_produto` exige notação nomeada, essa rota simplesmente não é aplicável aqui.
**How to avoid:** Usar o padrão de bloco PL/SQL anônimo com OUT escalares (Pattern 1 acima) em vez de tentar registrar o Object Type inteiro.
**Warning signs:** Erros de bind/scan mismatch ao tentar usar `rows.Scan(&structInteira)`.

### Pitfall 3: Confundir formato decimal do script de teste (vírgula) com formato do XML (ponto)

**What goes wrong:** Copiar o valor de exemplo `pPrecoTotal = 3849,9` do script de teste literalmente como string com vírgula ao montar o parâmetro de entrada Go→Oracle, causando erro de conversão numérica no Oracle (`ORA-01722: invalid number`) ou um valor 100x maior/menor por má interpretação do separador.
**Why it happens:** O script de teste é gerado pelo PL/SQL Developer no formato regional do editor (pt-BR, vírgula decimal); os campos do XML da NFe seguem o padrão SEFAZ (ponto decimal, ex.: `<vProd>3849.90</vProd>`).
**How to avoid:** Os parâmetros de entrada da função devem vir do parsing do XML (já em `float64` via `toDecimal`), nunca de uma cópia literal dos valores de exemplo do script `.TST`. Ao montar o bind Oracle, usar o tipo numérico Go nativo (`float64`), nunca serializar como string com vírgula.
**Warning signs:** Divergências absurdas (ordem de grandeza errada) nos primeiros testes de comparação da Fase 3.

### Pitfall 4: Falta de grupo fiscal (`pCodigoGrupoFiscal`) bloqueando o item sem necessidade

**What goes wrong:** Tratar "produto sem grupo fiscal encontrado em prod/PRODB" como erro fatal que aborta o processamento de toda a nota ou do lote.
**Why it happens:** Falta de isolamento de erro por item (requisito explícito ERP-03/FIS-03).
**How to avoid:** Cada item deve ter seu próprio registro de status (`pending`/`ok`/`error`) com mensagem de erro específica; a ausência de grupo fiscal deve marcar apenas aquele item como `sem_grupo_fiscal` e seguir para o próximo.
**Warning signs:** Um único produto sem cadastro completo em `prod`/`PRODB` interrompe a importação inteira de um XML com 50 itens.

### Pitfall 5: Concorrência excessiva de conexões Oracle

**What goes wrong:** Abrir uma goroutine por item sem limitar concorrência, esgotando o pool de sessões Oracle (erro `ORA-12520`/`ORA-00018: maximum number of sessions exceeded`) especialmente considerando que o acesso é somente leitura e provavelmente compartilhado com outros sistemas na mesma instância Oracle de produção.
**Why it happens:** `database/sql` por padrão não limita `MaxOpenConns`; sem configuração explícita, cada chamada concorrente pode abrir uma nova conexão física.
**How to avoid:** Configurar `db.SetMaxOpenConns(N)` (N pequeno, ex.: 5) na conexão Oracle dedicada ao pacote fiscal, e/ou processar itens sequencialmente/com semáforo.
**Warning signs:** Erros intermitentes de conexão sob carga que não ocorrem em testes com poucos itens.

## Code Examples

### Isolamento de erro por item (padrão recomendado em Go)

```go
// Source: padrão inferido — Go idioms para processamento resiliente de lote,
// sem referência direta no FB_APU04 (não há precedente lá para chamadas Oracle item a item)
type ItemResult struct {
    ItemID string
    Status string // "ok" | "error" | "sem_grupo_fiscal"
    ErrMsg string
}

func processFiscalBatch(ctx context.Context, oracleDB *sql.DB, pgDB *sql.DB, itens []NFeItem) []ItemResult {
    results := make([]ItemResult, 0, len(itens))
    sem := make(chan struct{}, 5) // limita concorrência Oracle

    var wg sync.WaitGroup
    var mu sync.Mutex

    for _, item := range itens {
        wg.Add(1)
        sem <- struct{}{}
        go func(it NFeItem) {
            defer wg.Done()
            defer func() { <-sem }()

            // recover: nunca deixar um panic em um item derrubar o processo/lote inteiro
            defer func() {
                if r := recover(); r != nil {
                    mu.Lock()
                    results = append(results, ItemResult{ItemID: it.ID, Status: "error",
                        ErrMsg: fmt.Sprintf("panic recuperado: %v", r)})
                    mu.Unlock()
                }
            }()

            res := processSingleItem(ctx, oracleDB, pgDB, it) // timeout de contexto por item
            mu.Lock()
            results = append(results, res)
            mu.Unlock()
        }(item)
    }
    wg.Wait()
    return results
}

func processSingleItem(ctx context.Context, oracleDB *sql.DB, pgDB *sql.DB, item NFeItem) ItemResult {
    itemCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()

    grupoFiscal, err := lookupGrupoFiscal(itemCtx, oracleDB, item.CProd)
    if err != nil {
        persistErrorStatus(pgDB, item.ID, "sem_grupo_fiscal", err)
        return ItemResult{ItemID: item.ID, Status: "sem_grupo_fiscal", ErrMsg: err.Error()}
    }

    fiscalResult, err := callFiscalPackage(itemCtx, oracleDB, item, grupoFiscal)
    if err != nil {
        persistErrorStatus(pgDB, item.ID, "error", err)
        return ItemResult{ItemID: item.ID, Status: "error", ErrMsg: err.Error()}
    }

    persistFiscalResult(pgDB, item.ID, fiscalResult)
    return ItemResult{ItemID: item.ID, Status: "ok"}
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| go-ora sem suporte a Object Types (versões antigas) | `RegisterType`/`RegisterTypeWithOwner` com nested types (v2.8.5+) e nested arrays (v2.8.6+) | v2.8.5/v2.8.6 (2024/2025) | Não muda a recomendação desta fase (a função usa notação nomeada, não acessível via SQL puro), mas é relevante caso uma versão futura do pacote fiscal exponha uma function chamável via SELECT |

**Deprecated/outdated:**
- Nenhuma API do go-ora foi deprecada que afete esta fase.

## Assumptions Log (RESOLVED)

> Todas as assunções abaixo foram resolvidas diretamente pelo usuário em 2026-07-01, com acesso real ao Oracle da Ferreira Costa. Mantidas aqui com a resolução inline para rastreabilidade. Nenhuma bloqueia o planejamento.

| # | Claim | Section | Risk if Wrong | Resolução |
|---|-------|---------|----------------|-----------|
| A1 | Schema exato de `prod`/`PRODB` não confirmado | Standard Stack / Architecture Patterns | Alto | **RESOLVIDO.** Usuário forneceu a query real: `SELECT pb.cod_empresa, pb.codigo, pb.grupo_fiscal, p.especial origem, p.ncm FROM prodb pb, prod p WHERE p.codigo = pb.codigo`. Chave de junção: `codigo` (código do produto — mapeia do `cProd` do XML). `prodb` tem `grupo_fiscal` (= `pCodigoGrupoFiscal`) e `cod_empresa` (ver A6 abaixo). `prod` tem `especial` (aliás "origem", provável fonte de `pOrigemProduto`) e `ncm`. |
| A2 | `pCodigoGrupoFiscal` vem só de `prod`/`PRODB`, nenhum outro param depende de tabela não mapeada | Perguntas a Responder / Architecture | Médio | Ainda válido como assunção de trabalho — a query fornecida cobre `pCodigoGrupoFiscal`; demais parâmetros (UF, CNPJ, tipo contribuinte etc.) vêm do XML/config, não de novas tabelas Oracle identificadas nesta rodada. |
| A3 | Função `calcula_imposto_produto` é somente leitura, sem efeito colateral de escrita | Security Domain | Médio-Alto | **RESOLVIDO.** Usuário confirmou: "só retorna" — função pura, sem INSERT/UPDATE interno. Constraint de somente-leitura do projeto está preservada. |
| A4 | Usuário Oracle já configurado (Fase 1) tem `SELECT`/`EXECUTE` necessários em `prod`/`PRODB`/pacote | Perguntas a Responder (#3) | Alto | **RESOLVIDO.** Usuário confirmou: "tem permissão" — as credenciais já cadastradas em `erp_bridge_config` (Fase 1) são suficientes, sem grants adicionais necessários. |
| A5 | Schema/owner de `PKG_FISCAL_FCTAX` não confirmado | Code Examples / Architecture | Médio | **RESOLVIDO.** Usuário confirmou que é o mesmo schema/usuário já configurado no ERP_BRIDGE — chamada sem necessidade de qualificação `SCHEMA.` explícita. |
| A6 (nova) | `prodb.cod_empresa` — como a aplicação determina o valor correto para a Ferreira Costa | Architecture / Proposta de Modelagem | Alto | **RESOLVIDO.** Não existe tabela de mapeamento no Oracle nem é necessário criar uma. `cod_empresa` é um valor fixo e pequeno por filial/UF, informado diretamente pelo usuário: exemplos concretos **`2` = filial Recife/PE** e **`1` = filial Garanhuns/PE**. A aplicação deve resolver esse valor a partir da UF de origem (`pUFOrigem`, já derivado do XML) via um mapeamento pequeno e estático no código Go (ex.: `map[string]int{"PE-RECIFE": 2, "PE-GARANHUNS": 1}` ou equivalente, chave exata a definir pelo planner/executor a partir do campo disponível no XML — provavelmente `emit.CNPJ` ou um código de filial, já que duas filiais podem compartilhar a mesma UF "PE"). Não requer nova tabela Postgres nem tela de administração — é constante de aplicação, extensível depois se novas filiais forem adicionadas. |

**Se esta tabela estivesse vazia:** não é o caso — havia assunções significativas de schema/permissão Oracle, todas resolvidas nesta rodada de esclarecimento com o usuário.

## Open Questions (RESOLVED)

1. **Estrutura real das tabelas `prod`/`PRODB`** — **RESOLVIDO** (ver A1 acima). Query de referência: `SELECT pb.grupo_fiscal, p.especial AS origem, p.ncm FROM prodb pb, prod p WHERE p.codigo = pb.codigo AND pb.codigo = :codigoProduto AND pb.cod_empresa = :codEmpresa` — o filtro por `cod_empresa` é OBRIGATÓRIO (ver A6) para não retornar múltiplas linhas/grupo fiscal errado quando o mesmo `codigo` de produto existe para mais de uma filial.

2. **Schema/owner do pacote `PKG_FISCAL_FCTAX` e grants** — **RESOLVIDO** (ver A4/A5 acima). Mesmo usuário/schema já configurado na Fase 1, sem qualificação adicional, sem grants pendentes.

3. **Efeitos colaterais de escrita dentro de `calcula_imposto_produto`** — **RESOLVIDO** (ver A3 acima). Função pura, apenas retorna.

4. **Volume esperado de itens por lote/nota — dimensionamento de concorrência**
   - Ainda em aberto (não bloqueia o planejamento): nenhuma informação nova do usuário sobre volume esperado. Mantida a recomendação original: processamento síncrono item-a-item com concorrência limitada (5-10) como MVP suficiente para volumes de teste manual; revisar se o uso real mostrar necessidade de fila assíncrona (fora do escopo desta fase).

5. **(Nova) Como resolver `cod_empresa` a partir do XML por filial** — parcialmente aberto: o usuário confirmou os valores (`2`=Recife/PE, `1`=Garanhuns/PE) e que basta passar o parâmetro, sem tabela nova. Falta apenas confirmar, durante o planejamento/execução, QUAL campo do XML (CNPJ emitente, código de filial, ou UF+outro discriminador) deve ser usado como chave para escolher entre os valores fixos — deixado como decisão de implementação (Claude's Discretion) para o planner, com base no XML de exemplo disponível durante a execução.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go 1.24 | Backend build | ✓ | go1.24.1 linux/amd64 | — |
| Docker | Ambiente de execução | ✓ | 29.5.2 | — |
| PostgreSQL | Persistência da aplicação | ✓ | aceita conexões (pg_isready OK) | — |
| `github.com/sijms/go-ora/v2` | Conexão Oracle (prod/PRODB + FCCORP_BKP) | ✓ (já no go.mod) | v2.9.0 | — |
| Cliente Oracle real (instância com `prod`/`PRODB`/`FCCORP_BKP`) | Testes de integração real desta fase | ✗ (não verificável neste ambiente de pesquisa) | — | Nenhum fallback viável — esta fase **depende** de acesso real à instância Oracle da Ferreira Costa para validação; sem isso, só é possível validar a lógica de parsing XML e a montagem da string do bloco PL/SQL, não a execução real |
| `sqlplus`/`tnsping` (Oracle client tools) | Diagnóstico manual de conexão/descoberta de schema | ✗ (não instalado nesta máquina) | — | go-ora não precisa de Oracle Instant Client (driver puro-Go); mas a ausência de `sqlplus` limita a capacidade de o desenvolvedor validar manualmente queries de descoberta de schema fora do código Go — recomenda-se pedir ao usuário/DBA acesso a um cliente Oracle (SQL Developer, DBeaver) para essa descoberta |

**Missing dependencies with no fallback:**
- Acesso real à instância Oracle (prod/PRODB/FCCORP_BKP) para qualquer teste de integração além de unit tests com mocks — bloqueia validação ponta a ponta desta fase até que a conexão real esteja disponível e as credenciais tenham os grants necessários (ver Assumptions A1, A4, A5).

**Missing dependencies with fallback:**
- Cliente Oracle CLI local (`sqlplus`) — pode ser contornado usando uma ferramenta gráfica (SQL Developer/DBeaver) já disponível ao usuário, ou delegando a descoberta de schema a uma tarefa `checkpoint:human-verify` no plano.

## Project Constraints (from CLAUDE.md)

- Stack obrigatória: Go 1.24, Postgres (dados do app), Oracle (ERP_BRIDGE + FCCORP_BKP), React + TypeScript + Vite + Tailwind, Docker — manter paridade com FB_APU04.
- Acesso Oracle **somente leitura** — nenhuma tarefa desta fase deve incluir INSERT/UPDATE/DELETE contra `prod`, `PRODB` ou `FCCORP_BKP`.
- Credenciais Oracle geridas como no ERP_BRIDGE herdado (AES-GCM, nunca versionadas) — reaproveitar `erp_bridge_config`/`DecryptFieldWithFallback` sem modificação do padrão de segurança.
- Escopo v1 restrito à empresa Ferreira Costa — nenhuma tarefa deve reconstruir seleção multi-empresa/multi-tenant.
- Workflow GSD: qualquer edição de arquivo desta fase deve ocorrer via `/gsd:execute-phase` (ou `/gsd:quick`/`/gsd:debug` para ajustes pontuais fora do fluxo de fase), nunca edição direta fora do workflow.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|--------------|--------------------|
| XML-01 | Usuário importa um ou vários XMLs completos de NF-e de saída via tela reaproveitada do FB_APU04 | Pattern 2 (parsing) + Standard Stack (`encoding/xml`, `archive/zip`, `rardecode`) — reaproveitar `xml_upload.go`/`ImportarXMLsSaida.tsx` do FB_APU04 quase sem alteração |
| XML-02 | Sistema faz parse e persiste cabeçalho da nota, itens e impostos do XML no Postgres | Recommended Project Structure (migração `006_nfe_saidas.sql`) + structs `nfeProc`/`det`/`prod`/`detImposto` do FB_APU04, adaptados para `nfe_saidas`/`nfe_saidas_itens` |
| XML-03 | Usuário visualiza as notas/itens importados e os valores de imposto originais do XML | Schema de referência 058/075 do FB_APU04 cobre todos os campos necessários para exibição do "esperado" |
| XML-04 | Importação reporta erros de parse/validação de XML de forma clara | Padrão `xmlUploadError`/validações de `tpNF`/modelo já usado em `processSingleXML` do FB_APU04 |
| ERP-01 | Conexão Oracle de leitura é configurável (reaproveitando credenciais/infra do ERP_BRIDGE) | Já implementado na Fase 1 (`erp_bridge_config`, `ERPBridgeConfigHandler`) — nenhuma tarefa nova de configuração necessária, apenas reaproveitamento |
| ERP-02 | Para cada item do XML, sistema consulta `prod` + `PRODB` e obtém o grupo fiscal do produto | Bloqueado por Open Question #1 (schema não confirmado) — Assumption A1; planner deve prever tarefa de descoberta de schema antes/durante a implementação |
| ERP-03 | Itens sem grupo fiscal localizado são sinalizados sem interromper o processamento dos demais | Pitfall 4 + Code Example "Isolamento de erro por item" |
| FIS-01 | Sistema executa o script do pacote fiscal no FCCORP_BKP com parâmetros do XML + grupo fiscal | Pattern 1 (bloco PL/SQL anônimo com OUT escalares) — ponto de maior risco técnico da fase, tratado em profundidade |
| FIS-02 | Sistema carrega e persiste o retorno do script vinculado ao item correspondente | Ver proposta de schema Postgres na próxima seção (`fiscal_execution_items`) |
| FIS-03 | Falhas na execução do script por item são capturadas e exibidas sem abortar o lote | Pitfall 5 + Code Example "Isolamento de erro por item" — padrão de goroutine com semáforo + recover |
</phase_requirements>

## Proposta de Modelagem Postgres (pergunta #4)

Três tabelas novas, todas vinculadas a `nfe_saidas_itens` (reaproveitada do FB_APU04 com pequenas adaptações):

### `nfe_saidas` / `nfe_saidas_itens` (valores ESPERADOS do XML)
Reaproveitar diretamente o schema das migrações `058_create_nfe_saidas.sql` e `075_create_nfe_itens_tables.sql` do FB_APU04 (seção `nfe_saidas_itens`), adaptando apenas o nome do módulo/schema conforme já feito na Fase 1 para outras tabelas. Esse schema já cobre exatamente os campos citados no PROJECT.md linha 32 (base ICMS, vlr ICMS, base ST, ICMS ST, base PIS/COFINS, vlr PIS/COFINS) como colunas dedicadas (`v_bc_icms`, `v_icms`, `v_bc_st`... nota: `v_st`/`v_bc_st` estão no cabeçalho `nfe_saidas`, e é preciso confirmar se a Fase 3 precisa desses campos também a nível de item — o schema do FB_APU04 tem ST apenas a nível de cabeçalho, não de item, o que é uma lacuna a resolver no planejamento).

### `fiscal_execution_items` (resultado calculado pelo pacote fiscal, por item)

Recomendação: **modelo híbrido** — colunas dedicadas para os campos usados na comparação visual da Fase 3 (conforme PROJECT.md linha 32: base ICMS, vlr ICMS, base ST, ICMS ST, base PIS/COFINS, vlr PIS/COFINS, DIFAL, FCP) + uma coluna `JSONB` para os ~65 campos completos do retorno (auditoria/depuração/futura extensão sem migração).

```sql
CREATE TABLE IF NOT EXISTS fiscal_execution_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id          UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    nfe_item_id         UUID NOT NULL REFERENCES nfe_saidas_itens(id) ON DELETE CASCADE,

    -- Status de execução (isolamento de erro por item — FIS-03/ERP-03)
    status              TEXT NOT NULL DEFAULT 'pending', -- pending | ok | error | sem_grupo_fiscal
    error_message        TEXT,
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
    valor_cofins                NUMERIC(15,2),  -- result.ValorCOFINS
    percentual_difal           NUMERIC(7,4),   -- result.PercentualDifal
    valor_icms_partilha_destino NUMERIC(15,2), -- result.ValorIcmsPartilhaDestino (DIFAL)
    valor_icms_pobreza         NUMERIC(15,2),  -- result.ValorIcmsPobreza (FCP)

    -- Retorno completo (~65 campos) para auditoria/depuração e campos da Reforma Tributária
    -- (IBS UF, IBS Município, CBS) ainda não mapeados para colunas dedicadas
    full_result         JSONB NOT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_fiscal_execution_item UNIQUE (nfe_item_id)
);

CREATE INDEX IF NOT EXISTS idx_fiscal_execution_status ON fiscal_execution_items(company_id, status);
CREATE INDEX IF NOT EXISTS idx_fiscal_execution_nfe_item ON fiscal_execution_items(nfe_item_id);
```

**Justificativa do híbrido:** a Fase 3 (tela de comparação) só precisa de um subconjunto claro de campos (PROJECT.md linha 32) para exibição rápida e ordenação/filtro — colunas dedicadas para esses evitam `JSONB ->> 'campo'` em toda query de UI. Os ~55 campos restantes (incluindo todo o bloco novo de Reforma Tributária — IBS/CBS) são armazenados em `full_result JSONB` para não exigir uma migração de 65 colunas agora, mantendo flexibilidade caso a Fase 3 (ou uma fase futura) precise expor mais campos — nesse caso, promover campos específicos de JSONB para colunas dedicadas é uma migração aditiva simples.

**Status distinto de erro (ERP-03 vs FIS-03):** `sem_grupo_fiscal` é usado quando o lookup Oracle (`prod`/`PRODB`) não encontra o produto (ERP-03); `error` é usado para qualquer falha na chamada do pacote fiscal em si — timeout, erro de conexão Oracle, exceção PL/SQL (FIS-03). Essa distinção permite à Fase 3 e a relatórios diferenciar "não deu pra nem tentar calcular" de "tentou calcular e falhou".

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V2 Authentication | yes | Já coberto pela Fase 1 (JWT + middleware) — endpoints novos desta fase (upload XML, execução fiscal) devem usar `withAuth` como os demais |
| V3 Session Management | yes | Herdado sem alteração da Fase 1 |
| V4 Access Control | yes | Escopo por `company_id` (Ferreira Costa) igual ao padrão já estabelecido; nenhum endpoint novo deve aceitar `company_id` arbitrário do cliente |
| V5 Input Validation | yes | Validação de XML (schema, `tpNF=1`, modelo 55/65 — já no padrão herdado); validação dos parâmetros antes de enviar ao Oracle (evitar injeção via bloco PL/SQL montado dinamicamente — ver Threat abaixo) |
| V6 Cryptography | yes | Credenciais Oracle já em AES-256-GCM (Fase 1, `crypto.go`) — reaproveitar sem modificação |
| V12 File Handling | yes | Upload de XML/ZIP/RAR — já mitigado no FB_APU04 (anti-ZIP-bomb, path traversal, limite de tamanho) — reaproveitar mitigações ao copiar `xml_upload.go` |

### Known Threat Patterns for este stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|------------------------|
| Injeção via montagem dinâmica de bloco PL/SQL (a string do bloco anônimo é gerada em Go, embora os *valores* dos parâmetros sejam sempre bind variables, não concatenação) | Tampering | A string do bloco PL/SQL (nomes de campo, estrutura `declare/begin/end`) deve ser **estática/gerada a partir de metadados fixos no código**, nunca a partir de entrada do usuário; todos os *valores* (pCnpjEmpresa, pProduto, etc.) devem ser passados exclusivamente via `sql.Named`/bind variables, nunca por `fmt.Sprintf` concatenando valor no SQL |
| XML External Entity (XXE) no parsing de NFe | Information Disclosure / Tampering | `encoding/xml` do Go **não expande entidades externas por padrão** (ao contrário de parsers de outras linguagens) — comportamento seguro herdado automaticamente; ainda assim, confirmar que nenhum `xml.Decoder` customizado habilita `Strict: false` de forma insegura ou usa uma lib externa vulnerável a XXE |
| Exaustão de conexões/sessões Oracle (DoS acidental) | Denial of Service | `db.SetMaxOpenConns` com limite baixo na conexão Oracle dedicada; semáforo de concorrência no processamento de lote (Pitfall 5) |
| Vazamento de credenciais Oracle em logs de erro do pacote fiscal | Information Disclosure | Seguir o padrão já estabelecido em T-04-02 (Fase 1) — sanitizar mensagens de erro Oracle antes de logar/exibir, nunca incluir DSN/usuário/senha em `log.Printf` de erro de execução do pacote fiscal |
| Acesso de escrita acidental ao FCCORP_BKP/prod/PRODB | Tampering | Confirmar (Assumption A3/A4) que o usuário Oracle configurado tem **apenas** `SELECT`/`EXECUTE` de leitura — nenhuma tarefa desta fase deve executar `INSERT`/`UPDATE`/`DELETE` contra essas tabelas; se a function `calcula_imposto_produto` tiver efeito colateral de escrita interno, isso é comportamento de terceiro a ser aceito/documentado como risco, não corrigido pelo código desta fase |

## Sources

### Primary (HIGH confidence)
- `/tmp/11_Script_Teste_Pacote_FCTAX_1S_Reforma_Tributaria.TST` — contrato exato da função `PKG_FISCAL_FCTAX.calcula_imposto_produto` fornecido pelo usuário, lido diretamente
- `proxy.golang.org/github.com/sijms/go-ora/v2/@latest` — confirmação de versão v2.9.0, 2025-06-09
- Código-fonte lido diretamente: `/home/claudiobezerra/projetos/FB_APU04/backend/handlers/{nfe_saidas.go,xml_upload.go,erp_bridge.go,erp_bridge_xml.go}`, migrações `058`, `075`, `106`, `002` do FB_APU04 e FB_TESTESFC
- `.planning/PROJECT.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`, `01-SECURITY.md`, `01-01-SUMMARY.md`, `01-04-SUMMARY.md` do FB_TESTESFC — lidos diretamente

### Secondary (MEDIUM confidence)
- [go-ora README oficial](https://github.com/sijms/go-ora/blob/master/README.md) — via WebFetch, seções sobre OUT parameters, RefCursor, RegisterType/RegisterTypeWithOwner, limitações de nested types
- [Oracle PL/SQL Language Reference — subprogram parameters](https://docs.oracle.com/en/database/oracle/oracle-database/19/lnpls/subprogram-parameters.html) — confirmação de que notação nomeada não é permitida em chamadas a partir de SQL

### Tertiary (LOW confidence — sinalizado para validação)
- [sijms/go-ora Discussion #120](https://github.com/sijms/go-ora/discussions/120) — relato de usuário sobre limitação de Object Types complexos como OUT parameter; consistente com o README oficial mas não é fonte primária
- [golangbridge forum — Custom Type Out Parameter](https://forum.golangbridge.org/t/how-to-call-oracle-stored-procedure-with-custom-type-out-parameter/22241) — refere-se ao driver `godror`, não `go-ora`; usado apenas como contexto comparativo, não como base de implementação

## Metadata

**Confidence breakdown:**
- Standard stack (parsing XML, driver Oracle): HIGH — código real do FB_APU04 lido diretamente + versão do driver confirmada via proxy.golang.org oficial
- Architecture (bloco PL/SQL anônimo com OUT escalares): MEDIUM — inferido combinando o contrato do script de teste (fonte primária) com documentação oficial do go-ora sobre OUT parameters; **não testado nesta pesquisa contra uma instância Oracle real** — validar no primeiro plano de execução com um teste de conexão real antes de assumir a implementação completa
- Lookup de grupo fiscal (`prod`/`PRODB`): HIGH — schema e query de junção reais fornecidos pelo usuário com acesso Oracle direto em 2026-07-01 (ver Assumptions Log, A1/A6 resolvidas); falta apenas confirmar, durante a execução, qual campo do XML deriva `cod_empresa` por filial (Open Question #5)
- Pitfalls: MEDIUM-HIGH — pitfalls 1-3 fundamentados em evidência direta (script de teste + doc Oracle); pitfalls 4-5 são boas práticas gerais de engenharia, não específicas desta integração

**Research date:** 2026-07-01
**Valid until:** 30 dias para a parte de stack/parsing (estável); a parte de integração Oracle real deve ser **revalidada no primeiro teste de conexão efetivo** — o schema de `prod`/`PRODB` e as permissões já foram confirmados pelo usuário nesta rodada, restando validar apenas a execução real do bloco PL/SQL contra a instância Oracle.
