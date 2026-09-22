# Task Management API (Multi-User)

> High-performance, production-ready Multi-User Task Management REST API written in Go, implementing Clean Architecture, Distributed Idempotency, Database Transaction Integrity, and Structured Observability.

---

## Language / Bahasa
- [English Documentation](#english-documentation)
- [Dokumentasi Bahasa Indonesia](#dokumentasi-bahasa-indonesia)

---

# English Documentation

## 1. Overview & Architecture

This project is built following **Clean Architecture (Domain-Driven Hexagonal Principles)** to decouple core business logic from database drivers, frameworks, and third-party services.

```
                    ┌───────────────────────────┐
                    │      HTTP Transport       │
                    │   (Gin, Routes, Headers)  │
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │      Usecase Layer        │
                    │  (Pure Business Rules)    │
                    └─────────────┬─────────────┘
                                  │
         ┌────────────────────────┴────────────────────────┐
         │                                                 │
┌────────▼──────────┐                             ┌────────▼──────────┐
│  Domain Interface │                             │  Domain Interface │
│   (Repository)    │                             │  (Idempotency)    │
└────────┬──────────┘                             └────────┬──────────┘
         │                                                 │
┌────────▼──────────┐                             ┌────────▼──────────┐
│ PostgreSQL (GORM) │                             │   Redis (SETNX)   │
└───────────────────┘                             └───────────────────┘
```

### Key Architectural Design Patterns:
1. **Dual-Identifier Architecture (Public UUID + Internal BigInt)**:
   - **External Safety:** Public APIs expose exclusively UUIDv4 identifiers (`/tasks/:uuid`, `assignee_uuid`). This prevents enumeration attacks and auto-increment sniffing.
   - **Internal Performance:** Databases use clustered sequential `BIGSERIAL PRIMARY KEY` (`id`). This enables high-density B-Tree indexing, zero fragmentation, and $O(1)$ cursor-based pagination.
2. **Distributed Idempotency Engine (`POST /tasks`)**:
   - Header: `Idempotency-Key: <UUID>`.
   - Uses an atomic in-flight lock (`SETNX` with 30s timeout) to serialize concurrent requests.
   - Once persisted, response payloads are stored with a 24-hour TTL. Duplicate requests within 24 hours immediately receive the cached `201 Created` response without modifying the database.
   - Concurrent duplicates arriving at the same millisecond receive `409 Conflict` (`CONCURRENT_REQUEST`) rather than triggering race-condition duplicate inserts.
3. **Database Transaction Integrity (`POST /tasks/:uuid/assign`)**:
   - Managed through `TxManager.WithTransaction(ctx, fn)`.
   - Atoms: (1) Reassign task $\rightarrow$ (2) Audit log insert to `task_logs` $\rightarrow$ (3) Dispatch notification.
   - If any step fails (e.g. notification dispatch error), the entire transaction rolls back cleanly.
4. **Structured Error Handling**:
   - Strictly conforms to the specification:
     ```json
     {
       "status": 400,
       "code": "TASK_VALIDATION_ERROR",
       "message": "Title is required and cannot be empty",
       "timestamp": "2026-09-22T17:00:00Z"
     }
     ```
   - Global panic recovery catches unhandled exceptions, logs internal details, and serves a sanitized `500 INTERNAL_SERVER_ERROR` envelope without leaking internal stack traces.
5. **Observability & Logging**:
   - Standard library `log/slog` structured JSON logs.
   - Enriched with: `request_id`, `method`, `path`, `status`, `latency`, and `client_ip`.
   - Automatic severity routing: `INFO` (2xx/3xx), `WARN` (4xx), `ERROR` (5xx).

---

## 2. API Endpoints

### Authentication
| Method | Endpoint | Description | Auth Required |
|---|---|---|---|
| `POST` | `/auth/register` | Register a new user | No |
| `POST` | `/auth/login` | Login and obtain JWT token | No |

### Teams (For Multi-User Assignment)
| Method | Endpoint | Description | Auth Required |
|---|---|---|---|
| `POST` | `/teams` | Create a team (creator becomes ADMIN) | Yes |
| `GET` | `/teams` | List current user's teams | Yes |
| `POST` | `/teams/:uuid/members` | Add a member to a team | Yes |

### Tasks
| Method | Endpoint | Description | Auth Required | Idempotency |
|---|---|---|---|---|
| `POST` | `/tasks` | Create task | Yes | `Idempotency-Key` (UUID) |
| `GET` | `/tasks` | List user tasks (filter, search, pagination) | Yes | No |
| `GET` | `/tasks/:uuid` | Get task detail | Yes | No |
| `PUT` | `/tasks/:uuid` | Update task | Yes | No |
| `DELETE` | `/tasks/:uuid` | Soft-delete task | Yes | No |
| `POST` | `/tasks/:uuid/assign` | Assign task to team member (Atomic Tx) | Yes | No |

#### List Query Parameters (`GET /tasks`):
- `page`: Page number (default: 1)
- `limit`: Number of items per page (default: 10, max: 100)
- `status`: Filter by enum (`TODO`, `IN_PROGRESS`, `DONE`)
- `search` / `title`: Filter by title (`ILIKE` supported by `pg_trgm` GIN index)
- `cursor`: Internal sequential cursor for $O(1)$ cursor-based pagination

---

## 3. Getting Started

### Prerequisites
- Docker & Docker Compose
- Or Go 1.23+, PostgreSQL 17+, and Redis 7+ installed locally.

### Running via Docker Compose (Turnkey)
```bash
# Clone and enter directory
git clone <repo-url>
cd task-management-api

# Start App, PostgreSQL, and Redis
docker compose up --build -d

# View real-time structured logs
docker compose logs -f app
```
The API is available at `http://localhost:8080`.
The database is automatically pre-seeded with initial users, teams, and sample tasks.

### Default Seed Accounts (Password: `password123`)
| Email | Team | Role | Notes |
|---|---|---|---|
| `alice@example.com` | Backend Engineering | ADMIN | Can assign tasks to Bob & Charlie |
| `bob@example.com` | Backend Engineering | MEMBER | In Alice's team |
| `charlie@example.com` | Backend Engineering | MEMBER | In Alice's team |
| `david@example.com` | Product Management | ADMIN | Different team (useful to test cross-team rejection) |

### Running Locally
```bash
# Copy environment configuration
cp .env.example .env

# Run database seeder (Optional if running outside Docker)
make seed

# Run unit tests (No external DB/Redis required)
make test

# Run tests with race condition detector
make test-race

# Run API server locally
make run
```

---

## 4. Concurrency & Race Condition Verification

The test suite validates idempotency and database transaction rollback **without requiring live databases or external dependencies**:
- **Sequential Idempotency:** Validates identical responses and asserts that duplicate requests do not create additional records.
- **Concurrent Duplicate Test:** Fires 30 concurrent goroutines simultaneously with identical `Idempotency-Key` headers. Asserts that **exactly one task** is persisted in the database with zero race conditions.
- **Transaction Rollback Test:** Induces mock notification failure to guarantee atomic rollback across the task update and audit log.

To run:
```bash
make test-race
```

---

<br/>

---

# Dokumentasi Bahasa Indonesia

## 1. Ringkasan & Arsitektur

Proyek ini dibangun menggunakan **Clean Architecture** untuk memisahkan logika bisnis inti dari implementasi database, framework, maupun layanan eksternal.

### Karakteristik Arsitektur:
1. **Dual-Identifier Architecture (Public UUID + Internal BigInt)**:
   - **Keamanan Eksternal:** Seluruh endpoint dan payload client menggunakan UUIDv4 (`/tasks/:uuid`, `assignee_uuid`) agar aman dari *enumeration attack*.
   - **Performa Database:** Database menggunakan `id BIGSERIAL PRIMARY KEY` berurutan untuk memaksimalkan efisiensi B-Tree index dan mendukung pagination berbasis kursor $O(1)$.
2. **Distributed Idempotency Engine (`POST /tasks`)**:
   - Header: `Idempotency-Key: <UUID>`.
   - Menggunakan distributed lock Redis (`SETNX`) dengan window 24 jam.
   - Request pertama memproses data (status 201 Created). Request ulang dengan key yang sama dalam 24 jam mengembalikan response yang identik tanpa membuat duplikat task.
   - Request paralel pada milidetik yang sama mendapatkan status `409 Conflict` (`CONCURRENT_REQUEST`) untuk mencegah race condition.
3. **Integritas Transaksi Database (`POST /tasks/:uuid/assign`)**:
   - Dijalankan dalam satu transaksi atomik: (1) Validasi tim $\rightarrow$ (2) Update assignee $\rightarrow$ (3) Catat audit log di `task_logs` $\rightarrow$ (4) Kirim notifikasi.
   - Jika salah satu proses gagal, seluruh perubahan di-rollback secara otomatis.
4. **Structured Error Handling**:
   - Format response error seragam:
     ```json
     {
       "status": 400,
       "code": "TASK_VALIDATION_ERROR",
       "message": "Title is required and cannot be empty",
       "timestamp": "2026-09-22T17:00:00Z"
     }
     ```
   - Dilengkapi middleware panic recovery yang mencegah kebocoran stack trace internal ke production client.
5. **Structured Logging (Observability)**:
   - Menggunakan `log/slog` bawaan Go stdlib dengan format JSON.
   - Mencakup field: `request_id` (UUID), `method`, `path`, `status`, `latency`, `client_ip`.
   - Level log otomatis: `INFO` (2xx/3xx), `WARN` (4xx), `ERROR` (5xx).

---

## 2. Cara Menjalankan

### Menggunakan Docker Compose (Siap Pakai)
```bash
# Masuk ke direktori
cd task-management-api

# Jalankan container (App + PostgreSQL + Redis)
docker compose up --build -d

# Cek logs
docker compose logs -f app
```
Aplikasi berjalan pada port `http://localhost:8080`.
Database otomatis terisi data awal (users, teams, tasks) saat container pertama kali dijalankan.

### Akun Awal / Seed (Password: `password123`)
| Email | Tim | Role | Keterangan |
|---|---|---|---|
| `alice@example.com` | Backend Engineering | ADMIN | Bisa assign task ke Bob & Charlie |
| `bob@example.com` | Backend Engineering | MEMBER | Satu tim dengan Alice |
| `charlie@example.com` | Backend Engineering | MEMBER | Satu tim dengan Alice |
| `david@example.com` | Product Management | ADMIN | Beda tim (untuk uji validasi beda tim) |

### Menjalankan Unit Test Mandiri
Unit test tidak membutuhkan koneksi database atau Redis aktif (menggunakan mock thread-safe in-memory store):
```bash
# Seed database jika dijalankan di luar docker
make seed

# Jalankan seluruh unit test
make test

# Jalankan test dengan deteksi race condition
make test-race
```

---

## 3. Contoh Request cURL

### 1. Register User
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ichsan",
    "email": "ichsan@example.com",
    "password": "securepassword123"
  }'
```

### 2. Login User
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "ichsan@example.com",
    "password": "securepassword123"
  }'
```

### 3. Create Task dengan Idempotency Key
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -H "Idempotency-Key: 9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d" \
  -d '{
    "title": "Setup Microservices CI/CD",
    "description": "Configure GitHub Actions pipeline",
    "status": "TODO"
  }'
```

### 4. List Tasks dengan Filter & Pagination
```bash
curl -X GET "http://localhost:8080/tasks?status=TODO&search=CI%2FCD&page=1&limit=10" \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>"
```

### 5. Assign Task ke Anggota Tim
```bash
curl -X POST http://localhost:8080/tasks/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -d '{
    "assignee_uuid": "3a3cfd38-9993-4903-b0f3-cb094911aa12"
  }'
```
