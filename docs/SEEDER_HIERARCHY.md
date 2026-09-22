# Database Seeder Hierarchy & Test Scenarios

Dokumen ini menjelaskan struktur relasi, hierarki data seeder, dan skenario pengujian bisnis yang didukung secara otomatis oleh data awal (*seed data*).

---

## 1. Visual Hierarchy Tree

```
                       [TEAMS]
                          │
         ┌────────────────┴────────────────┐
         │                                 │
┌────────▼──────────────────┐    ┌─────────▼──────────────────┐
│ Backend Engineering Team   │    │ Product Management Team    │
│ UUID: aaaaaaaa-aaaa...    │    │ UUID: bbbbbbbb-bbbb...     │
└────────┬──────────────────┘    └─────────┬──────────────────┘
         │                                 │
         ├──────────────────────┐          │
         │ (Role: ADMIN)        │          │ (Role: ADMIN)
┌────────▼──────────┐           │ ┌────────▼──────────┐
│ Alice Admin       │           │ │ David Product     │
│ alice@example.com │           │ │ david@example.com │
└────────┬──────────┘           │ └───────────────────┘
         │                      │
         ├──────────────┐       │ (Role: MEMBER)
         │ Creates      │ ┌─────▼─────────────┐
         │              │ │ Bob Developer     │
         │              │ │ bob@example.com   │
         │              │ └───────────────────┘
         │              │
         │              │ (Role: MEMBER)
         │        ┌─────▼─────────────┐
         │        │ Charlie Backend   │
         │        │ charlie@example.com│
         │        └───────────────────┘
         │
         ├──► Task 1: "Setup CI/CD Pipeline with GitHub Actions"
         │    Status: IN_PROGRESS
         │    Assignee: Bob Developer
         │
         ├──► Task 2: "Implement Distributed Idempotency Key Lock"
         │    Status: DONE
         │    Assignee: Charlie Backend
         │
         └──► Task 3: "Database Transaction & Audit Log on Assign"
              Status: TODO
              Assignee: NULL (Ready for assignment test)
```

---

## 2. Default Seed Accounts (Password: `password123`)

| No | Name | Email | UUID | Team | Team Role |
|---|---|---|---|---|---|
| 1 | Alice Admin | `alice@example.com` | `11111111-1111-1111-1111-111111111111` | Backend Engineering | ADMIN |
| 2 | Bob Developer | `bob@example.com` | `22222222-2222-2222-2222-222222222222` | Backend Engineering | MEMBER |
| 3 | Charlie Backend | `charlie@example.com` | `33333333-3333-3333-3333-333333333333` | Backend Engineering | MEMBER |
| 4 | David Product | `david@example.com` | `44444444-4444-4444-4444-444444444444` | Product Management | ADMIN |

---

## 3. Pre-seeded Tasks

| Title | Status | Creator | Assignee | Task UUID |
|---|---|---|---|---|
| **Setup CI/CD Pipeline** | `IN_PROGRESS` | Alice | Bob | `9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d` |
| **Implement Idempotency Lock** | `DONE` | Alice | Charlie | `8c2edb5e-4c8e-5cbe-acee-3c1e8c4eda7e` |
| **DB Transaction on Assign** | `TODO` | Alice | *Unassigned* | `7d3fec6f-5d9f-6dcf-bdff-4d2f9d5feb8f` |

---

## 4. Test Scenarios Supported by Seed Data

### Skenario 1: Assign Task Berhasil (Dalam Satu Tim)
* **Aktor:** Alice (`alice@example.com`)
* **Task Target:** Task 3 (`7d3fec6f-5d9f-6dcf-bdff-4d2f9d5feb8f`)
* **Assignee:** Bob (`22222222-2222-2222-2222-222222222222`)
* **Hasil:** HTTP 200 OK. Task terupdate, riwayat tercatat di `task_logs`, dan notifikasi terkirim.

### Skenario 2: Assign Task Ditolak (Beda Tim / Cross-Team Boundary)
* **Aktor:** Alice (`alice@example.com`)
* **Task Target:** Task 3 (`7d3fec6f-5d9f-6dcf-bdff-4d2f9d5feb8f`)
* **Assignee:** David (`44444444-4444-4444-4444-444444444444`)
* **Hasil:** HTTP 400 Bad Request dengan kode `DIFFERENT_TEAM` (*"Assignee must belong to the same team as the assigner"*). Perubahan di-rollback penuh dan tidak ada `task_logs` yang tersimpan.

### Skenario 3: Multi-User Task Visibility Isolation
* Login sebagai **David** $\rightarrow$ `GET /tasks` menghasilkan daftar kosong, karena David bukan creator maupun assignee dari Task 1, 2, atau 3.
* Login sebagai **Bob** $\rightarrow$ `GET /tasks` hanya menampilkan Task 1 (karena Bob adalah assignee).
* Login sebagai **Alice** $\rightarrow$ `GET /tasks` menampilkan Task 1, 2, dan 3 (karena Alice adalah creator).

---

## 5. System Maintenance Endpoints (Up/Down Migration & Seed)

Endpoints ini dilindungi oleh header: `X-Admin-Key: <ADMIN_KEY>` (Default: `admin-secret-key-123`).

| Endpoint | Method | Fungsi | File SQL yang Dieksekusi |
|---|---|---|---|
| `/system/migrate/up` | `POST` | DDL inisialisasi skema tabel & indexes | `000001_init_schema.up.sql` |
| `/system/migrate/down` | `POST` | Drop seluruh tabel | `000001_init_schema.down.sql` |
| `/system/seed/up` | `POST` | Memasukkan data seeder | `000002_seed_data.up.sql` |
| `/system/seed/down` | `POST` | Menghapus seluruh data seeder | `000002_seed_data.down.sql` |

### Contoh Request cURL:
```bash
# Reset dan jalankan ulang seed data
curl -X POST http://localhost:8080/system/seed/down \
  -H "X-Admin-Key: admin-secret-key-123"

curl -X POST http://localhost:8080/system/seed/up \
  -H "X-Admin-Key: admin-secret-key-123"
```
