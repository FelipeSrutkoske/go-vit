# go-vit

**go-vit** é um gerenciador visual de Git para o terminal (TUI), feito em Go, pensado para ser limpo, intuitivo e amigável tanto para iniciantes quanto para o uso diário.

## Instalação

Certifique-se de ter o **Go** e o **Git** instalados no sistema e execute:

```bash
go install github.com/FelipeSrutkoske/go-vit@latest
```

## Uso

Execute no diretório do repositório desejado ou informe o caminho como argumento:

```bash
# Abre o repositório no diretório atual
go-vit

# Abre um repositório em outro caminho
go-vit /caminho/do/repositorio
```

Opções disponíveis:

- `-h`, `--help`: Exibe a mensagem de ajuda
- `-v`, `--version`: Exibe a versão instalada

## Principais Funcionalidades

- **Branches**: Liste branches locais e remotas, crie novas branches, troque de branch (com opção automática de *Stash + Trocar* quando houver alterações pendentes), faça merge e delete branches com segurança.
- **Alterações**: Visualize arquivos modificados, adicionados, removidos e não rastreados; faça *stage* e *unstage*, crie commits, use *stash*/*pop* e descarte alterações com confirmação.
- **Diff**: Inspecione diferenças linha a linha com destaque colorido (`+` em verde, `-` em vermelho) em um painel rolável.
- **Conflitos**: Identifique blocos de conflito (`<<<<<<<` / `=======` / `>>>>>>>`) destacados visualmente e resolva escolhendo suas alterações (*Ours*), as alterações recebidas (*Theirs*) ou abrindo o arquivo diretamente no `$EDITOR`.

## Atalhos de Teclado

### Globais

| Atalho | Ação |
| --- | --- |
| `Tab` / `1` / `2` | Alternar entre as abas **Branches** e **Alterações & Conflitos** |
| `↑` / `k` e `↓` / `j` | Navegar pelos itens da lista |
| `g` / `Home` e `G` / `End` | Ir para o início ou fim da lista |
| `K` / `PgUp` / `Ctrl+U` e `J` / `PgDn` / `Ctrl+D` | Rolar o painel direito (commits, diff ou conflitos) |
| `f` | Executar `git fetch` |
| `p` | Executar `git pull` |
| `R` | Atualizar informações do repositório |
| `?` | Exibir / ocultar tela de ajuda |
| `q` / `Ctrl+C` | Sair do go-vit |

### Aba Branches (`1`)

| Atalho | Ação |
| --- | --- |
| `Enter` | Trocar para a branch selecionada |
| `r` | Mostrar / ocultar branches remotas |
| `n` | Criar nova branch |
| `d` | Deletar branch selecionada |
| `m` | Fazer merge da branch selecionada na branch atual |

### Aba Alterações & Conflitos (`2`)

| Atalho | Ação |
| --- | --- |
| `Espaço` / `s` | Adicionar ou remover arquivo do *stage* (*Stage / Unstage*) |
| `c` | Abrir modal de commit |
| `z` | Guardar (*Stash*) ou restaurar (*Pop*) alterações |
| `x` | Descartar alterações do arquivo (com confirmação) |

### Resolução de Conflitos

| Atalho | Ação |
| --- | --- |
| `1` | Manter nossas alterações (*Ours*) |
| `2` | Aceitar alterações recebidas (*Theirs*) |
| `e` | Abrir arquivo conflitado no `$EDITOR` |

## Estrutura do Projeto

```text
.
├── main.go              # Ponto de entrada e parsing de flags/argumentos
├── internal/
│   ├── git/             # Cliente Git via os/exec e parsers de status/diff
│   └── tui/             # Interface terminal em Bubble Tea v2 e Lip Gloss v2
├── go.mod
└── README.md
```

## Contribuição

Contribuições são bem-vindas. Para colaborar:

1. Faça um fork do repositório.
2. Crie uma branch para sua alteração (`git checkout -b minha-alteracao`).
3. Execute os testes antes de abrir o Pull Request:

```bash
go test -race ./...
go vet ./...
```

4. Abra um Pull Request descrevendo o problema resolvido ou a melhoria proposta.

