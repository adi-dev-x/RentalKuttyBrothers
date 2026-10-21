# Rental Software — Backend API

A Go backend for running an equipment rental business. It tracks every rentable
unit in stock, lets staff rent units to customers through delivery orders
(challans), bills those orders by the day, records payments and invoices, follows
damaged and under-repair items, issues quotations, and reports on revenue,
damage and outstanding balances.

All endpoints are served under `/irrl` on the configured `HOST:SERVER_PORT`.

## What the application does

### Stock / items
- **Add products** (`POST /irrl/addProduct`). Each product gets a product code
  (`IRR0001`, `IRR0002`, …), and adding *N* units creates *N* items with
  sequential item codes (`IRR0001-001`, `IRR0001-002`, …).
- **Bulk import from Excel** (`POST /irrl/uploadItemSheet`, multipart field
  `file`). Each row goes through the same logic as `addProduct`.
- **Stock lookups**: product listing (`/listing`), product info by code
  (`/listStockCnt/:product_code`), and `/item/available`, which picks a random
  available unit for a product code and reserves it straight away, so two
  callers never get the same unit.
- **Item status changes** (`/updateItem/:itemID`, `/genericStatusUpdate`).
  Statuses include `AVAILABLE`, `RESERVED`, `RENTED` and others.

### Inventory
- On startup the app creates (or reuses) one inventory row named `MAIN`
  (`constants.DefaultInventorySite`) and keeps its id in `config.InventoryID`.
  Every new item and order is attached to that inventory.

### Customers
- Create customers (`POST /irrl/customers`).

### Orders (delivery challans)
- **Create an order** (`POST /irrl/addOrder`) as `RESERVED` or `INITIATED`. It
  creates a `delivery_chelan` row with an auto-generated invoice number, adds one
  `delivery_items` row per unit, and marks those units `RENTED`.
- **Start an order** (`POST /irrl/initiateOrder`) with guarantee images.
- **Update an order**: status (`/updateOrder/:orderID`), line items
  (`/updateOrderItem`), gate-pass details (`/updateOrderPass`), and return dates
  for the whole order or a single item (`/updateOrderReturnedAt`,
  `/updateOrderItemReturnedAt`).
- **Order details** (`GET /irrl/orderDetails/:orderID`): the order, its customer
  and all of its line items.
- **Delivery report** (`GET /irrl/reports/delivery`): an Excel download that can
  be filtered by `date_range` and `remark`.

### Billing and payments
- **Daily rent accrual**: a cron job runs every night at 00:00 IST
  (`pkg/cron`). It recalculates each active line item's amount as
  `rent_amount × days rented` (stopping at the return date) and adds those up
  into the order totals.
- **Transactions / invoices** (`POST /irrl/addSubTransaction`): records a payment
  or invoice against an order and returns the invoice id the server generated.
  You can also list transactions (`/subTransactions`) and edit one
  (`/editTransaction/:transactionID`).

### Damage, repair and site tracking
- Mark an item as damaged, with photos and a description, or clear the flag
  (`POST /irrl/markDamage`).
- Move an item into repair (`POST /irrl/markRepairing`) and see its repair
  history (`/repairHistory`).
- Mark an item as being at a customer site (`POST /irrl/markSite`).
- Each of these has grouped and flat list views: `/damaged/*`, `/repairing/*`
  and `/site/*`.

### Quotations
- Create, list and view quotations (`/quotation`, `/quotation/:id`).

### Analytics
- **Operations**: damage by customer and by item, blocked customers, rental
  counts per item, and an order-status summary (`/analytics/*`).
- **Money**: cost summary, revenue trend, repair costs, outstanding balances,
  revenue per customer and revenue per inventory (`/analytics/cost/*`,
  `/analytics/customer-revenue`).

### Files
- `POST /irrl/upload` uploads images (damage photos, guarantee documents and so
  on) to an AWS S3 bucket and returns their URLs.

### Generic / config-driven APIs
- `/genericApiJoin/:ApiType` and `/genericApiUnJoin/:ApiType` run read queries
  stored in the `api_registry` table and filtered by `api_attributes`. This lets
  the frontend get new list views without new Go code.
- `/attribute/:AttributeType` returns lookup values: brands, categories, and
  item main and sub types.
- `/genericDelete` deletes a row by table name and id.

## Tech stack

| Area | What is used |
|---|---|
| Language | Go 1.23 |
| HTTP framework | [Echo v4](https://echo.labstack.com/) |
| Database | PostgreSQL 15. Raw SQL through `database/sql`; GORM only opens the connection pool; `lib/pq` runs the migrations |
| File storage | AWS S3 (`aws-sdk-go-v2`) |
| Scheduling | `robfig/cron/v3` (nightly billing job, IST) |
| Excel | `xuri/excelize/v2` (item import, delivery report) |
| Config | `spf13/viper` + `go-playground/validator` |
| Auth / security | `golang-jwt/jwt`, `bcrypt` |
| Email | SMTP (order confirmation and return emails) |
| Containers | Docker, docker-compose (app + Postgres) |

## Project layout

```
cmd/api/main.go          entry point: load config → run migrations → DI → start server
pkg/config               .env loading/validation, global InventoryID
pkg/database             Postgres connection, schema migrations, default inventory
pkg/di                   wires repository → service → handler, starts cron, builds the S3 client
pkg/boot                 Echo server setup
pkg/irrl                 the domain: handler.go (HTTP), service.go (business logic), repository.go (SQL)
pkg/model                request/response and DB structs
pkg/cron                 nightly current_amount job
pkg/client               email / OTP helpers
pkg/util                 generic query builder, response helpers, constants
pkg/templates            HTML templates
schema.sql, migration_*.sql   database schema and upgrades
```

## Database

When the app starts, `db.RunSchemaMigrations` applies these files, in this
order, from the working directory:

1. `schema.sql`: the full schema (`CREATE ... IF NOT EXISTS`)
2. `migration_product_code.sql`
3. `migration_transaction_invoice.sql`
4. `migration_order_returned_at.sql`
5. `migration_transaction_daily_amount.sql`

Every file can be run more than once safely, so this works on a fresh database,
a fully migrated one, or an old one. The main tables are `items`, `inventory`,
`customer`, `delivery_chelan`, `delivery_items`, `transaction`,
`item_status_history`, `repair_history`, `quotation` / `quotation_items`,
`attributes`, `api_registry`, `api_attributes` and `users`.

## Configuration

Settings are read from `pkg/config/.env`, and environment variables override
them:

| Key | Purpose |
|---|---|
| `PG_HOST`, `PG_PORT`, `PG_USERNAME`, `PG_PASSWORD`, `PG_DB_NAME`, `PG_SSL_MODE`, `PG_DBMS_NAME`, `PG_DRIVER_NAME` | PostgreSQL |
| `HOST`, `SERVER_PORT` | Address the API listens on |
| `EMAIL`, `PASSWORD` | SMTP account for outgoing email |
| `adminjwtkey`, `vendorjwtkey` | JWT signing keys |
| `BUCKET_ID`, `AWS_REGION`, `AWS_ACCKEY`, `AWS_SECKEY` | S3 uploads |

## Running

Start Postgres (or the whole stack) with Docker:

```bash
make docker-run      # docker compose up
make docker-down     # stop it
```

Run the API locally from the repo root. It must be the repo root because the
migration SQL files are read with relative paths:

```bash
make run             # go run cmd/api/main.go
make build           # build ./main
make watch           # live reload with air
```

## Known gaps

- Authentication middleware is written but commented out in
  `MountRoutes`, so every `/irrl` endpoint is currently open.
- `make test` points at `./tests`, which does not exist yet.
- The Docker image copies only the binary. The `schema.sql` /
  `migration_*.sql` files that the app reads at startup are not copied, and
  `ENTRYPOINT ["/bin/bash"]` combined with `CMD ["/app/api"]` makes bash try to
  run the binary as a script. Fix both before relying on the container build.
- The S3 client's region is hard-coded to `eu-north-1` in `pkg/di/di.go`;
  `AWS_REGION` is not used.
