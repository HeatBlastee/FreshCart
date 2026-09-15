# FreshCart

FreshCart is a full-stack e-commerce platform with a Go backend (REST + gRPC) and a React storefront. It covers the parts a real store needs — catalog browsing and search, cart and checkout, Stripe payments, order tracking, coupons, wishlists, product reviews, and transactional email — built on a ports-and-adapters architecture with JWT (or OIDC) auth, versioned SQL migrations, and both unit and Docker-backed integration test suites.

**Highlights**
- Clean domain separation (`user`, `product`, `order`, `payment`, `notification`), each independently testable behind repository/service interfaces
- REST and gRPC exposed side by side for the same business logic
- Stripe-backed checkout with webhook-verified payment confirmation
- Pluggable auth: local JWT by default, or OIDC via Authentik for SSO setups
- CI-covered: unit tests, testcontainer-based integration tests, linting, and generated Swagger docs

## Architecture

The application runs two servers concurrently (backend) plus a React web frontend:

- **HTTP (REST)** — Gin framework, port `8888`
- **gRPC** — port `8889`, with JWT auth interceptor

Each domain (`user`, `product`, `order`, `payment`, `notification`) follows a ports-and-adapters layout:

```
internal/{domain}/
├── model/       # GORM models
├── dto/         # Request/response structs with validation tags
├── repository/  # Database access (depends on dbs.Database interface)
├── service/     # Business logic (depends on repository interfaces)
└── port/
    ├── http/    # Gin handlers and route registration
    └── grpc/    # gRPC handlers and server registration
```

| Domain | HTTP | gRPC |
|--------|------|------|
| user | ✓ | ✓ |
| product | ✓ | ✓ |
| order | ✓ | ✓ |
| payment | ✓ | — |
| notification | ✓ | — |

## Tech Stack

**Backend**

| Concern | Library |
|---------|---------|
| HTTP framework | [Gin v1.12](https://github.com/gin-gonic/gin) |
| gRPC | [grpc-go v1.79](https://github.com/grpc/grpc-go) |
| ORM | [GORM v1.31](https://gorm.io) + PostgreSQL |
| Cache | [go-redis v9](https://github.com/redis/go-redis) |
| Auth | JWT ([golang-jwt v5](https://github.com/golang-jwt/jwt)) |
| Validation | [gocommon/validation](https://github.com/quangdangfit/gocommon) |
| API Docs | [Swagger](https://github.com/swaggo/swag) |
| Testing | [testify v1.11](https://github.com/stretchr/testify) + [mockery](https://github.com/vektra/mockery) |
| Proto codegen | [buf](https://buf.build) + [protobuf v1.36](https://google.golang.org/protobuf) |

**Frontend**

| Concern | Library |
|---------|---------|
| Framework | [React 18](https://react.dev) + TypeScript |
| Build tool | [Vite](https://vitejs.dev) |
| Styling | [Tailwind CSS](https://tailwindcss.com) |
| Routing | [React Router v6](https://reactrouter.com) |
| Data fetching | [TanStack Query](https://tanstack.com/query) |
| Forms | [React Hook Form](https://react-hook-form.com) + [Zod](https://zod.dev) |
| HTTP client | [Axios](https://axios-http.com) |

## Prerequisites

- Go 1.26+
- Node.js 18+
- PostgreSQL
- Redis

Docker Compose for local dependencies: [docker-compose-template](https://github.com/quangdangfit/docker-compose-template/blob/master/base/docker-compose.yml)

## Getting Started

**1. Clone and configure**

```bash
git clone <your-repo-url> freshcart
cd freshcart
cp config.sample.yaml config.yaml
```

Edit `config.yaml` (lives at the repo root and is loaded from the working directory; override with `CONFIG_FILE=/path/to/config.yaml`):

```yaml
environment: production
http_port: 8888
grpc_port: 8889
auth_secret: your-secret-key
database_uri: postgres://username:password@localhost:5432/freshcart
redis_uri: localhost:6379
redis_password:
redis_db: 0

# Stripe (payments)
stripe_secret_key: sk_test_xxx
stripe_webhook_secret: whsec_xxx
stripe_publishable_key: pk_test_xxx

# SMTP (notifications — point at MailHog locally: host=localhost, port=1025)
smtp_host:
smtp_port: 25
email_from: no-reply@freshcart.local
```

**2. Apply database migrations**

The app no longer runs `AutoMigrate` — schema lives in versioned SQL files under
`migrations/` and is applied with [golang-migrate](https://github.com/golang-migrate/migrate).

```bash
brew install golang-migrate   # or: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
DATABASE_URI="postgres://username:password@localhost:5432/freshcart?sslmode=disable" make migrate-up
```

See [`migrations/README.md`](migrations/README.md) for conventions and the
production / Kubernetes init-container pattern.

**3. Run the backend**

```bash
go run cmd/api/main.go
```

```
INFO    HTTP server is listening on PORT: 8888
INFO    GRPC server is listening on PORT: 8889
```

**4. Run the web frontend**

```bash
cd web
npm install
npm run dev
```

Web UI: [http://localhost:3000](http://localhost:3000)

> The frontend proxies all `/api` requests to the backend at `http://localhost:8888`, so both servers must be running.

**5. Browse the API**

Swagger UI: [http://localhost:8888/swagger/index.html](http://localhost:8888/swagger/index.html)

## API Reference

### Auth
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| GET | `/api/v1/auth/me` | Get current user |
| PUT | `/api/v1/auth/change-password` | Change password |

### Addresses
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/addresses` | List my addresses |
| POST | `/api/v1/addresses` | Create address |
| GET | `/api/v1/addresses/:id` | Get address |
| PUT | `/api/v1/addresses/:id` | Update address |
| DELETE | `/api/v1/addresses/:id` | Delete address |
| PUT | `/api/v1/addresses/:id/default` | Set default address |

### Wishlist
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/wishlist` | Get my wishlist |
| POST | `/api/v1/wishlist` | Add product to wishlist |
| DELETE | `/api/v1/wishlist/:productId` | Remove from wishlist |

### Categories
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/categories` | List categories |
| GET | `/api/v1/categories/:id` | Get category |
| POST | `/api/v1/categories` | Create category (auth) |
| PUT | `/api/v1/categories/:id` | Update category (auth) |
| DELETE | `/api/v1/categories/:id` | Delete category (auth) |

### Products
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/products` | List products (cached) |
| GET | `/api/v1/products/:id` | Get product (cached) |
| POST | `/api/v1/products` | Create product (auth) |
| PUT | `/api/v1/products/:id` | Update product (auth) |
| GET | `/api/v1/products/:id/reviews` | List product reviews |
| POST | `/api/v1/products/:id/reviews` | Create review (auth) |
| PUT | `/api/v1/products/:id/reviews/:reviewId` | Update review (auth) |
| DELETE | `/api/v1/products/:id/reviews/:reviewId` | Delete review (auth) |

### Orders
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/orders` | Place order |
| GET | `/api/v1/orders` | List my orders |
| GET | `/api/v1/orders/:id` | Get order details |
| PUT | `/api/v1/orders/:id/cancel` | Cancel order |
| PUT | `/api/v1/orders/:id/status` | Update order status (admin) |

### Coupons
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/coupons` | Create coupon (auth) |
| GET | `/api/v1/coupons/:code` | Get coupon by code |

### Payments
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/orders/:id/payment-intent` | Create Stripe PaymentIntent for order |
| POST | `/api/v1/webhooks/stripe` | Stripe webhook (signature-verified, no JWT) |
| GET | `/api/v1/config/public` | Public client config (Stripe publishable key) |

### Notifications
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/me/notification-preferences` | Get my notification preferences |
| PUT | `/api/v1/me/notification-preferences` | Update notification preferences |

> Cart is client-side only (persisted in the browser). Order creation accepts the full
> line items and the server re-validates products, prices, and stock.

## Development

**Run all unit tests with coverage**

```bash
make unittest
```

**Run a single test suite**

```bash
go test ./internal/product/service/... -v -run TestProductServiceTestSuite
```

**Run a single test case**

```bash
go test ./internal/product/service/... -v -run TestProductServiceTestSuite/TestCreateSuccess
```

**Run integration tests**

Integration suites live under `tests/integration/` (per-domain: `order`,
`payment`, `user`, `product`, `notification`), are gated by the
`//go:build integration` tag, and use [testcontainers](https://golang.testcontainers.org/)
to spin up real Postgres / Redis / MailHog. They are invisible to `make
unittest` / `go test ./...`.

```bash
make integration
# = go test -tags=integration -timeout 9000s -v -coverprofile=coverage.integration.out ./tests/integration/...
```

Requirements: a running Docker daemon. The shared helpers (`StartPostgres`,
`StartRedis`, `NewHTTPEnv`) live in `tests/testutil/`. CI runs the
integration job in parallel with the unit job; both upload coverage to
Codecov under the `unittest` / `integration` flags.

**Database migrations**

```bash
make migrate-up                  # apply all pending migrations
make migrate-down                # roll back the latest migration
make migrate-status              # print current schema version
make migrate-new name=add_index  # scaffold the next NNNN_*.{up,down}.sql pair
```

Set `DATABASE_URI` in your shell to override the default
(`postgres://postgres:test@localhost:5432/freshcart?sslmode=disable`).

**Regenerate mocks**

```bash
make mock
```

**Regenerate Swagger docs**

```bash
make doc
```

**Regenerate proto (Uses https://buf.build)**

```bash
cd proto && make build
```

## Deploying to Render

`render.yaml` at the repo root is a Blueprint that provisions:

- **freshcart-db** — managed Postgres
- **freshcart-api** — the Go backend, built from the root `Dockerfile` (REST only; the gRPC
  port is not reachable externally since Render web services route a single port)
- **freshcart-web** — the React frontend as a static site, with a rewrite so
  `/api/*` transparently proxies to `freshcart-api` (keeps the frontend's relative
  `/api/v1` base URL working unchanged, and means no CORS setup is needed)

In the Render dashboard: **New → Blueprint**, point it at this repo, and Render
reads `render.yaml`. You still need to:

1. **Add a Key Value (Redis) instance** — not declared in `render.yaml` because
   Render's Blueprint fields for Key Value services aren't stable enough to
   hardcode. Create one manually (New → Key Value), then on `freshcart-api` set:
   - `redis_uri` — the instance's `host:port` (this app wants a plain
     `host:port` address, not a `redis://` URL)
   - `redis_password` — the instance's password
2. **Set Stripe keys** on `freshcart-api` (required for the payment flow):
   `stripe_secret_key`, `stripe_webhook_secret`, `stripe_publishable_key`.
   Point the Stripe webhook at `https://<freshcart-api-url>/api/v1/webhooks/stripe`.
3. **Set SMTP credentials** on `freshcart-api` for order-receipt emails:
   `smtp_host`, `smtp_user`, `smtp_password` (`smtp_port` defaults to 587).
4. **Run migrations** against the new database — Render doesn't run them for
   you. From your machine, with `golang-migrate` installed:
   ```bash
   DATABASE_URI="<freshcart-db external connection string>?sslmode=require" make migrate-up
   ```
5. If `freshcart-api` doesn't land on the exact hostname
   `freshcart-api.onrender.com` (names are first-come, first-served), update
   the rewrite `destination` in `render.yaml` under `freshcart-web` to match
   the real URL Render assigned.

All other backend config is plain environment variables (see
`config.sample.yaml` for the full list — `auth_secret` is auto-generated by
the blueprint, `database_uri` is wired to `freshcart-db` automatically).
Auth mode defaults to `jwt`; only set the `oidc_*` / `authentik_*` variables
if you're switching `auth_mode` to `oidc` against your own Authentik instance.

Treat `render.yaml` as a starting point, not gospel — Render's Blueprint
schema (plan names, `fromService` properties) changes over time, so diff it
against the current [Render Blueprint docs](https://render.com/docs/blueprint-spec)
before your first deploy.
