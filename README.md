# HNL Bank — Sistema de Banca en Linea (MVP)

**Repositorio:** https://github.com/Extibax/hnl_bank (publico)

**HNL Bank** es un MVP de banca en linea construido con un backend de doble base de datos: PostgreSQL para identidad y metadatos, y TigerBeetle como libro mayor (ledger) inmutable para saldos y transferencias. Incluye autenticacion JWT, cuentas, depositos, retiros, transferencias, historial y un asistente conversacional con IA (OpenRouter) capaz de ejecutar operaciones bancarias mediante tool-use.

## Stack tecnologico

| Capa | Tecnologia |
|------|-----------|
| Backend | Go (el instalado en este entorno: 1.26), Chi router, arquitectura en capas (handler -> service -> repository) |
| Base de datos relacional | PostgreSQL 16 (users, user_accounts, transactions) |
| Libro mayor financiero | TigerBeetle (cuentas y transferencias, saldos autoritativos) |
| Auth | JWT (HS256) + bcrypt para passwords |
| Chat IA | OpenCode (OpenAI-compatible) + DeepSeek, tool-use via un **servidor MCP** (Model Context Protocol) |
| Frontend | Vite + React + TypeScript + Tailwind + shadcn/ui + react-router-dom + axios + sonner |
| Infraestructura | Docker Compose (4 servicios) |
| Tests | `go test` en la capa de servicio con repositorios mockeados |

## Arquitectura

El backend usa un patron dual-DB: la fuente de verdad del dinero es TigerBeetle; PostgreSQL guarda la identidad y los metadatos enriquecidos para consultas e historial.

```
                          +---------------------+
                          |   Frontend (Vite)    |
                          |  React + shadcn/ui   |
                          +----------+----------+
                                     |  /api  (nginx proxy / Vite dev proxy)
                                     v
                          +----------+----------+
                          |  Backend (Go/Chi)   |
                          |  handler -> service |
                          |         -> repo     |
                          +---+-------------+---+
                              |             |
               +--------------+--+      +---+--------------+
               | PostgreSQL 16  |      | TigerBeetle      |
               | users          |      | accounts (ids)   |
               | user_accounts  |      | transfers        |
               | transactions   |      | (saldos, movtos) |
               +-----------------+      +------------------+
```

Las operaciones de dinero (deposito, retiro, transferencia) se registran como transferencias en TigerBeetle (autoritativo para el saldo) y, en paralelo, se inserta un registro enriquecedor en `transactions` (PostgreSQL) para el historial con descripcion, tipo y paginacion.

## Ejecucion

```bash
cp .env.example .env
docker-compose up --build
```

Visita http://localhost:5180. El backend queda en http://localhost:8080. (El puerto 5173 estaba ocupado por otro proyecto en este equipo, por lo que el frontend se sirve en el 5180; en un entorno libre basta con cambiar el mapeo `ports` de la seccion `frontend` de `docker-compose.yml`.) En el primer arranque el backend crea el esquema y siembra 1000 usuarios, 1605 cuentas y 6429 transacciones desde los datos de prueba embebidos. Un segundo arranque no vuelve a sembrar.

## Variables de entorno

| Variable | Descripcion | Default |
|----------|-------------|---------|
| `DATABASE_URL` | DSN de PostgreSQL | `postgres://hnlbank:hnlbank@postgres:5432/hnlbank?sslmode=disable` |
| `TIGERBEETLE_ADDRESS` | Direccion del cluster TigerBeetle | `3000` |
| `JWT_SECRET` | Secreto para firmar tokens JWT | `dev-secret-change-me` |
| `OPENAI_BASE_URL` | Base URL OpenAI-compatible para el chat IA | `https://opencode.ai/zen/v1` |
| `OPENAI_API_KEY` | API key de OpenCode para el chat IA (vacio = chat deshabilitado) | `(vacio)` |
| `OPENAI_MODEL` | Modelo de chat a usar (debe soportar tool-use) | `nemotron-3-ultra-free` |
| `PORT` | Puerto del backend | `8080` |

## Credenciales de prueba

| Email | Password |
|-------|----------|
| ihernandez@email.com | Isabel2024! |
| mjimenez@example.com | Miguel2024! |
| paulamolina@mail.com | Paula2024! |

## Endpoints API

| Metodo | Ruta | Descripcion | Auth |
|--------|------|-------------|------|
| GET | `/api/health` | Health check | No |
| POST | `/api/auth/register` | Crear usuario | No |
| POST | `/api/auth/login` | Login (devuelve token + user) | No |
| POST | `/api/auth/logout` | Invalidar token | Si |
| GET | `/api/accounts` | Listar cuentas con saldo | Si |
| GET | `/api/accounts/{id}/balance` | Saldo de una cuenta | Si |
| POST | `/api/transactions/deposit` | Deposito | Si |
| POST | `/api/transactions/withdraw` | Retiro | Si |
| POST | `/api/transactions/transfer` | Transferencia | Si |
| GET | `/api/transactions/{account_id}?limit&offset` | Historial de una cuenta | Si |
| POST | `/api/chat` | Chat con IA (tool-use) | Si |
| POST | `/api/chat/action` | Ejecutar accion critica confirmada | Si |

Los montos viajan como cadenas decimales (`"1234.56"`). Internamente se almacenan como enteros en centavos.

## Decisiones arquitectonicas

- **Por que TigerBeetle para finanzas:** es un libro mayor de doble entrada, inmutable y de alto rendimiento, disenado para balances y transferencias; garantiza atomicidad y evita errores de concurrencia en el dinero.
- **Por que MCP:** las operaciones bancarias de la IA se exponen como un servidor **Model Context Protocol** (JSON-RPC 2.0, en `internal/mcp/`) con las tools list_accounts, get_balance, get_transactions, make_deposit, make_withdrawal, make_transfer. El chat las ejecuta a traves de ese servidor; tambien corre standalone sobre stdio (`cmd/mcp-server`).
- **Por que Chi:** router ligero, compatible con `net/http` y con agrupacion de rutas + middlewares (JWT), ideal para una API pequeña y tipada.
- **Enteros en centavos:** evitar errores de punto flotante en dinero; la conversion a decimal ocurre solo en los limites de la API.

## Estructura del proyecto

```
hnl_bank/
├── .context/                     # Datos de prueba y plan
├── backend/
│   ├── cmd/server/main.go        # Arranque, wiring, seed
│   ├── internal/
│   │   ├── config/               # Configuracion por env vars
│   │   ├── handler/              # Handlers HTTP + router Chi
│   │   ├── id/                   # Generador de UUIDs
│   │   ├── middleware/           # Middleware JWT
│   │   ├── model/                # Modelos de dominio
│   │   ├── money/                # Conversion centavos <-> decimal
│   │   ├── repository/           # PG + TigerBeetle repos
│   │   ├── seed/                 # Seed de datos de prueba
│   │   └── service/              # Logica de negocio + tests
│   ├── go.mod / go.sum
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── api/                  # Cliente axios
│   │   ├── components/           # UI + layout + chat panel
│   │   ├── context/              # AuthContext
│   │   ├── lib/                  # utils + format
│   │   └── pages/                # Auth, Dashboard, Transactions, History
│   ├── Dockerfile
│   └── nginx.conf
├── docker-compose.yml
└── .env.example
```

## Tests

Los tests de servicio corren dentro de un contenedor `golang` (ya que Go no está instalado localmente en este equipo y `gcc` está bloqueado por AppLocker). Puedes hacerlo con:

```bash
make test
```

o manualmente:

```bash
docker run --rm -v "$PWD/backend:/src" -v "$HOME/go/pkg/mod:/go/pkg/mod" -w /src golang:1.26 sh -c "go build ./... && go vet ./... && go test ./internal/service/..."
```

## Notas de entorno (Docker Desktop en WSL2)

- **TigerBeetle**: requiere io_uring, que el perfil seccomp por defecto de Docker bloquea; por eso el servicio usa security_opt seccomp=unconfined. Ademas, en Docker Desktop sobre WSL2 los volumenes montados no soportan el backend de almacenamiento de TigerBeetle (O_DIRECT), por lo que el directorio de datos queda dentro del contenedor (efimero). El arranque formatea el cluster con format y luego ejecuta start. Por ello, para una demo reproducible se recomienda `docker compose down -v && docker compose up --build`.
- **Seed**: los datos de prueba contienen emails duplicados; el seeder les agrega un sufijo (ej. email+2@dominio) para preservar la restriccion de unicidad e insertar los 1000 usuarios.
- **Chat IA**: sin OPENROUTER_API_KEY el endpoint /api/chat responde 503; el flujo completo de tool-use requiere esa clave.
## Integracion MCP (Model Context Protocol)

El asistente de IA accede a las operaciones bancarias a traves de un servidor MCP:

- `backend/internal/mcp/server.go` — servidor MCP con las 6 tools (saldos, historial, deposito, retiro, transferencia).
- `backend/internal/mcp/jsonrpc.go` — manejador JSON-RPC 2.0 (initialize, tools/list, tools/call).
- `backend/cmd/mcp-server/main.go` — servidor MCP standalone sobre stdio.

El `user_id` autenticado se pasa en `arguments` de cada `tools/call`.

## Scripts de base de datos / inicializacion

- `db/schema.sql` — DDL de las tablas PostgreSQL (entregable; el backend tambien lo crea via migracion embebida).
- `scripts/tb-init.sh` — inicializacion y arranque de TigerBeetle (formatea el cluster y ejecuta start); el servicio `tigerbeetle` del compose lo usa.