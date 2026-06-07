# goexpert-desafio1-clientserver

Sistema cliente/servidor em Go que consulta a cotação USD-BRL, persiste em SQLite e salva o valor em arquivo, respeitando timeouts com `context`.

## Requisitos

- Go 1.22 ou superior
- Docker Desktop (opcional — veja a seção [Docker](#docker))

## Como executar

### 1. Instalar dependências

```bash
go mod tidy
```

### 2. Iniciar o servidor

Em um terminal:

```bash
go run server.go
```

O servidor ficará disponível em `http://localhost:8080/cotacao`.

### 3. Executar o cliente

Em outro terminal (com o servidor em execução):

```bash
go run client.go
```

O cliente consulta o servidor, extrai o campo `bid` e grava o arquivo `cotacao.txt` no formato:

```
Dólar: 5.1671
```

## Timeouts

| Componente              | Operação              | Timeout |
|-------------------------|-----------------------|---------|
| `server.go`             | API externa           | 200ms   |
| `server.go`             | Persistência SQLite   | 10ms    |
| `client.go`             | Requisição ao servidor| 300ms   |

Quando um timeout é excedido, o erro é registrado no console do respectivo processo.

## Banco de dados

As cotações são salvas em `./data/cotacao.db` (SQLite), criado automaticamente pelo `server.go` na pasta `./data`.

## Docker

O Docker é **opcional** e serve apenas para inspecionar o banco SQLite via linha de comando. A aplicação Go **não depende** do Docker para funcionar.

### Pré-requisitos

- [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) instalado
- Docker Desktop **em execução** (ícone da baleia estável na bandeja do sistema)

Verifique se o engine está ativo:

```bash
docker info
```

Se aparecer erro como `failed to connect to the docker API`, abra o Docker Desktop e aguarde até que ele esteja totalmente iniciado antes de continuar.

### Subir o container

Na raiz do projeto:

```bash
docker compose up -d
```

Isso sobe um container com a imagem `keinos/sqlite3`, montando a pasta local `./data` em `/data` dentro do container. O arquivo do banco fica acessível em `/data/cotacao.db`.

### Consultar cotações salvas

Com o servidor já tendo registrado ao menos uma cotação:

```bash
docker exec -it sqlite sqlite3 /data/cotacao.db "SELECT * FROM cotacoes;"
```

Para abrir o shell interativo do SQLite:

```bash
docker exec -it sqlite sqlite3 /data/cotacao.db
```

Comandos úteis dentro do SQLite:

```sql
.tables;
SELECT * FROM cotacoes;
.quit
```

### Parar e remover o container

```bash
docker compose down
```

Os dados em `./data/cotacao.db` permanecem no disco local mesmo após parar o container.

### Alternativa sem Docker

Instale o [SQLite CLI](https://sqlite.org/download.html) ou use o [DB Browser for SQLite](https://sqlitebrowser.org/) e abra o arquivo `./data/cotacao.db` diretamente.
