# /speckit.plan — Fixora (AI-assisted IT Ticketing)

> Versi: 1.0
> Basis: speckit constitution (DDD, Clean Architecture, TDD)

## Ringkasan

Dokumen ini adalah **plan** teknis untuk implementasi Fixora — platform IT Ticketing yang dibantu AI. Dokumen mengikat domain model, ports, use cases, delivery adapters, dan cadangan infrastruktur. Ini ditulis agar langsung jadi blueprint pengembangan (kode + testable contracts).

---

## 1. Scope & Goals

* Implementasi minimum viable product (MVP) meliputi: ticket submission (manual + AI intake), comment flow, assignment, resolve flow, simple knowledge base CRUD, AI suggestion port (mockable), dan reporting dasar.
* Kriteria keberhasilan MVP: create → assign → resolve loop lengkap dengan audit + AI training hook.

---

## 2. Bounded Contexts & Responsibilities

* **Ticket Management**: lifecycle tiket, assignment, SLA status.
* **Commenting & Collaboration**: penyimpanan komentar, real-time notifications.
* **Knowledge Base**: CRUD entri solusi & tags, versioning ringan.
* **AI Assistant**: AISuggestionService (ingest description → suggestion + confidence), AITrainingService (learn from resolved tiket), EmbeddingProvider (generate text embeddings for KB & queries).
* **Reporting**: aggregasi metric (resolution time, first response, AI accuracy).

---

## 3. Domain Model (Ringkas)

### Ticket (Aggregate Root)

* id: UUID
* title: string
* description: string
* status: OPEN | IN_PROGRESS | RESOLVED | CLOSED
* category: NETWORK | SOFTWARE | HARDWARE | ACCOUNT | OTHER
* priority: LOW | MEDIUM | HIGH | CRITICAL
* createdBy: EmployeeID
* assignedTo: AdminID | null
* aiInsight: { text: string, confidence: float } | null
* createdAt, updatedAt

### Comment

* id, ticketId, authorId, role: EMPLOYEE|ADMIN|AI, body, createdAt

### KnowledgeBaseEntry

* id, contextLabel, solutionSteps, tags[], source: MANUAL|LEARNED, active:boolean, updatedAt

### Value Objects

* Metric (resolutionTime, firstResponseTime), AuditEntry

---

## 4. Ports (Interfaces) — contoh Go signatures

```go
// repository ports
type TicketRepository interface {
  Create(ctx context.Context, t *Ticket) error
  FindByID(ctx context.Context, id string) (*Ticket, error)
  Update(ctx context.Context, t *Ticket) error
  List(ctx context.Context, filter TicketFilter) ([]*Ticket, error)
}

type CommentRepository interface {
  Create(ctx context.Context, c *Comment) error
  ListByTicket(ctx context.Context, ticketID string) ([]*Comment, error)
}

type KnowledgeRepository interface {
  FindByContext(ctx context.Context, q string) ([]*KnowledgeBaseEntry, error)
  Upsert(ctx context.Context, e *KnowledgeBaseEntry) error
}

// ai ports
type AISuggestionService interface {
  SuggestMitigation(ctx context.Context, description string) (mitigation string, confidence float64, err error)
}

type AITrainingService interface {
  LearnFromResolved(ctx context.Context, ticket *Ticket, comments []*Comment) error
}

// embedding port for KB vector search
type EmbeddingProvider interface {
  // Returns embedding vector for single text
  Embed(ctx context.Context, text string) ([]float32, error)
  // Returns embedding vectors for batch
  EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
  // Embedding dimension (e.g., 768, 1536)
  Dimension() int
}

// Provider factory to support multi-vendor (OpenAI, Z.ai)
type AIProviderFactory interface {
  Suggestion() AISuggestionService
  Embeddings() EmbeddingProvider
}

// infra port
type NotificationService interface {
  NotifyUser(ctx context.Context, userID string, payload Notification) error
}
```

---

## 5. Use Cases (Application Layer)

Setiap use case harus ringan, idempotent, dan diuji menggunakan table-driven tests.

* **CreateTicketUseCase**

    * Input: CreateTicketDTO (title, description, createBy, optional: createViaAI boolean)
    * Flow: call AISuggestionService (non-blocking or bounded timeout) → persist Ticket with aiInsight if applicable → emit event `TicketCreated` → notify assignees/admins if auto-assign rule match
    * Output: TicketDTO

* **AISuggestUseCase**

    * Input: text
    * Flow: call AISuggestionService → if confidence >= threshold return mitigation; else ask clarifying questions

* **CommentOnTicketUseCase**

    * Persist comment → emit `CommentAdded` → notify relevant parties

* **AssignTicketUseCase**

    * Validate admin role → update assignedTo → append audit entry

* **ResolveTicketUseCase**

    * Admin adds final mitigation → change status to RESOLVED → call AITrainingService asynchronously (but must complete call or enqueue before returning success) → record resolution time

* **GetMetricsUseCase**

    * Query ReportingService → return MetricReportDTO

---

## 6. Eventing & Integration Patterns

* Use domain events for cross-cutting flows: `TicketCreated`, `TicketAssigned`, `TicketResolved`, `CommentAdded`, `KBUpdated`.
* Events are published to simple internal broker (in-memory for MVP) with adapters for Kafka/RabbitMQ in infra.
* AITrainingService should subscribe to `TicketResolved` and `KBUpdated`.

---

## 7. API Contracts (surface)

* REST endpoints (OpenAPI-ready):

    * POST /v1/tickets → create ticket
    * POST /v1/tickets/ai-intake → create ticket berbasis analisis AI dengan override terkontrol (lihat Use Case AIIntakeCreateTicketUseCase)
    * GET /v1/tickets/{id} → ticket detail + comments
    * POST /v1/tickets/{id}/comments → add comment
    * POST /v1/tickets/{id}/assign → assign ticket
    * POST /v1/tickets/{id}/resolve → resolve ticket
    * GET /v1/kb → list KB
    * POST /v1/kb/entries → create KB entry (draft)
    * POST /v1/kb/entries/upload-text → upload text/markdown content
    * PATCH /v1/kb/entries/{id} → update KB entry
    * POST /v1/kb/entries/{id}/publish → publish KB entry (chunking + embedding + index)
    * POST /v1/ai/suggest → return suggestion (JSON)
    * GET  /v1/ai/suggest/stream → SSE stream for suggestion candidates
    * GET /v1/reports → metric report

All endpoints must carry correlation-id and be secured via SSO/JWT.

---

## 8. Persistence & Schema (MVP)

Suggested tables (Postgres):

* tickets (id pk, title, description, status, category, priority, created_by, assigned_to, ai_insight jsonb, created_at, updated_at)
* comments (id, ticket_id fk, author_id, role, body, created_at)
* knowledge_entries (id, title, content, status enum: draft|active|archived, category, tags text[], source_type, version, created_by, created_at, updated_at)
* kb_chunks (id, entry_id fk, chunk_index, content text, embedding vector(D), created_at)
* metrics_agg (pre-aggregated if needed)
* audit_logs (id, resource_type, resource_id, action, actor, meta jsonb, created_at)

Add indices on ticket.status, ticket.assigned_to, created_at, and KB tables:
* idx_kb_entries_status (status)
* idx_kb_chunks_entry (entry_id)
* ivfflat index on kb_chunks.embedding using cosine ops (pgvector)

Extensions:
* CREATE EXTENSION IF NOT EXISTS vector; (pgvector)

---

## 9. Folder Structure (Starter for Go)

```
/cmd/fixora
/internal
  /domain
    ticket.go
    comment.go
    kb.go
  /usecase
    ticket_usecase.go
    ai_usecase.go
  /ports
    repository.go
    ai.go
    notification.go
  /adapter
    /http
      handler_ticket.go
      handler_ticket_ai_intake.go (endpoint intake)
      handler_ai_suggest.go (JSON + SSE)
      handler_kb.go (CRUD + publish)
    /persistence
      postgres_ticket_repo.go
      postgres_kb_repo.go
      pgvector_chunks_repo.go
    /ai
      openai_adapter.go (real + mockable)
      zai_adapter.go (real + mockable)
      embedding_adapter.go (wrap provider-specific embedding)
    /notification
      slack_adapter.go
  /infra
    events/broker.go
    sse/streamer.go (flush cadence, heartbeat)
  /test
    fixtures

/migrations
/docs

```

---

## 10. Testing Strategy

* Domain tests: invariants, status transitions, VO equality.
* Use case tests: mock ports for deterministic flows.
* Adapter integration tests: run with test Postgres and local mock AI (use docker-compose for CI).
* Provider parity tests: OpenAI & Z.ai adapters share contract; use VCR-like recorded fixtures or strict mocks.
* SSE handler tests: assert first-event latency, ordering of `candidate` events, heartbeat presence, graceful `end` & `error`.
* Vector retrieval tests: cosine distance correctness, Top-K stability under ivfflat; dimension mismatches handled.
* E2E: simulate a full flow with in-memory event broker and mock AI.

---

## 11. Security & Privacy

* Tickets may contain PII — store `description` encrypted at rest or redact patterns (msisdn, email) as configurable sanitizer.
* Provider API keys kept in secret store; per-provider privacy settings respected (no training on prompts unless opted-in).
* Prompt safety: enforce system guardrails; filter outbound content to prevent data leakage.
* Rate limit per-user and per-provider to avoid quota exhaustion; backoff & circuit-breaker on provider errors.
* RBAC: employee only CRUD own tickets; admin roles can act on assigned tickets.
* Audit trail mandatory for assignment & status changes.

---

## 12. Definition of Done (DoD) untuk MVP Stories

* Unit tests ≥ 80% coverage for domain & use cases.
* Integration test for repo + API endpoints.
* OpenAPI spec synced with endpoints (KB + AI Suggest + SSE).
* SSE streaming validated with integration tests and measured `first_event_ms`.
* pgvector indexing created and ANALYZE executed post-ingest.
* Basic telemetry (request duration, error count) wired.

---

## 13. Next Steps (Concrete tasks)

1. Sync `openapi.yaml` dengan endpoint KB & AI Suggest (JSON + SSE).
2. Create schema migrations for tables.
3. Implement TicketRepository + in-memory mock + Postgres adapter.
4. Implement AISuggestionService adapters: OpenAI & Z.ai; pilih via config.
5. Implement EmbeddingProvider adapters: OpenAI & Z.ai; gunakan untuk KB publish & query embeddings.
6. Implement HTTP handlers (tickets, KB CRUD/publish, AI suggest JSON + SSE) dan minimal React admin/employee pages untuk MVP flows.
6. Write tests for CreateTicketUseCase and AISuggestUseCase.
7. Write tests for SSE stream handler dan vector retrieval.

---

## 14. Notes on Speckit Alignment

* All ports & use cases are written to be testable & mockable.
* Domain-first: domain invariants live in `internal/domain` and have no infra deps.
* TDD-friendly: tests for every use case; event contracts versioned.

---

## 15. AI Provider Abstraction (OpenAI & Z.ai)

Tujuan: Mendukung multi-provider tanpa mengubah domain/usecase. Pemilihan provider dilakukan via konfigurasi.

Desain:
- `AIProviderFactory` menghasilkan `AISuggestionService` dan `EmbeddingProvider` sesuai env `AI_PROVIDER = openai|zai`.
- Adapter `openai_adapter.go` dan `zai_adapter.go` mengimplementasikan kontrak yang sama, termasuk mapping error & retry.
- Konfigurasi:
  - `OPENAI_API_KEY`, `ZAI_API_KEY`
  - `EMBEDDING_DIM` (default 1536)
  - `TOP_K` (default 10), `CHUNK_SIZE` (500–1000), `CHUNK_OVERLAP` (100–200)

Fallback:
- Jika provider utama gagal (5xx/timeouts), opsi fallback ke provider lain dengan batas percobaan. Logging `provider_failover=true`.

Compliance:
- Pastikan kebijakan data tiap provider dipatuhi (no training on data tanpa izin, anonymization bila perlu).

## 16. Embedding & Retrieval Pipeline (pgvector)

Publish KB:
1) Normalisasi teks → 2) Chunking → 3) EmbedBatch → 4) Simpan `kb_chunks` (embedding VECTOR(D)) → 5) Buat/refresh indeks ivfflat (cosine ops) → 6) ANALYZE.

Query Suggest:
1) Embed query → 2) Similarity search Top-K pada `kb_chunks` (entry status=active) → 3) Re-ranking heuristik (kategori/tags) → 4) Format kandidat untuk UI.

Dimensi:
- `Dimension()` di `EmbeddingProvider` menentukan D; migrasi/DDL harus konsisten dengan nilai tersebut.

## 17. SSE Implementation Plan

HTTP Adapter:
- Endpoint `GET /v1/ai/suggest/stream`.
- Header: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`.
- Event: `init` → banyak `candidate` → opsional `progress` → `end` atau `error`.
- Heartbeat komentar `:heartbeat` tiap ≤ 15 detik.
- Flush cadence ≤ 250ms.
- Idempotent reconnect via `queryId`.

Observability:
- Metrik: `first_event_ms`, `events_count`, `total_duration_ms` per `queryId`.

## 18. Configuration & Environment

ENV contoh:
- `AI_PROVIDER=openai` (opsi: `zai`)
- `OPENAI_API_KEY=...`
- `ZAI_API_KEY=...`
- `EMBEDDING_DIM=1536`
- `TOP_K=10`
- `CHUNK_SIZE=800`
- `CHUNK_OVERLAP=150`
- `SSE_FLUSH_MS=250`
- `SSE_HEARTBEAT_MS=15000`

## 19. Migrations & Ops

- `CREATE EXTENSION IF NOT EXISTS vector;`
- Tabel `knowledge_entries`, `kb_chunks` sesuai DDL spesifikasi.
- Indeks ivfflat pada `kb_chunks.embedding` dengan `vector_cosine_ops`.
- Jadwal VACUUM/ANALYZE berkala; backup harian; uji restore indeks.

---

## 20. Knowledge Base — Detail Implementasi

Tujuan: menjadikan KB siap produksi dengan pipeline yang deterministik (normalisasi → chunking → embedding → indeks → retrieval) serta dapat diuji dan diobservasi.

### 20.1 Data & Status
- `knowledge_entries`
  - status: `draft` → `active` → `archived`
  - version: integer autoincrement per entry; setiap publish menambah version.
  - tags: array string untuk filter semantik (kategori, produk, platform).
  - source_type: `MANUAL|LEARNED`.
- `kb_chunks`
  - `chunk_index`: urutan tetap dari hasil chunking (0..N-1).
  - `embedding`: `vector(D)` sesuai `EmbeddingProvider.Dimension()`.
  - konsistensi: semua chunk dari satu entry harus memiliki dimensi identik.

### 20.2 Alur Upload Teks (POST /v1/kb/entries/upload-text)
1) Validasi payload: `title`, `content` (text/markdown), `tags` opsional.
2) Normalisasi konten (lihat 20.3) dan simpan sebagai `knowledge_entries` status `draft`.
3) Kembalikan `KbEntryResponse` (id, status=`draft`).
4) Untuk konten besar (> LIMIT, misal 1MB), saran proses async: simpan dan berikan `jobId` untuk publish (lihat 20.7).

### 20.3 Normalisasi Teks
Tujuan: hilangkan noise, standarisasi whitespace, ubah markdown ke teks yang mudah di-chunk.
- Langkah:
  - Trim whitespace berlebih, normalisasi newline (\n).
  - Hapus/konversi elemen markdown: header jadi baris teks, bullet dipertahankan sebagai baris, code block disimpan apa adanya namun dibatasi panjang.
  - Deteksi bahasa opsional (untuk masa depan re-ranking), default dianggap bahasa dokumen.
  - Redaksi PII opsional: email, nomor telepon, password.
- Pseudokode:
  ```go
  func NormalizeMarkdown(md string) string {
    txt := stripMarkdown(md)            // gunakan lib atau converter sederhana
    txt = normalizeWhitespace(txt)      // regex untuk spasi, newline
    txt = limitCodeBlocks(txt, 2000)    // potong code block sangat panjang
    return strings.TrimSpace(txt)
  }
  ```

### 20.4 Algoritma Chunking
Tujuan: bagi teks menjadi potongan yang seimbang untuk embedding.
- Parameter: `CHUNK_SIZE` (default 800 chars), `CHUNK_OVERLAP` (default 150 chars).
- Strategi: sliding window berbasis karakter dengan pemisah kalimat agar tidak memutus kalimat di tengah.
- Pseudokode:
  ```go
  func ChunkText(text string, size, overlap int) []string {
    sentences := splitIntoSentences(text) // naive: regex .!? dengan spasi; atau lib NLP
    var chunks []string
    var buf strings.Builder
    for _, s := range sentences {
      if buf.Len()+len(s) <= size {
        if buf.Len() > 0 { buf.WriteString(" ") }
        buf.WriteString(s)
      } else {
        chunks = append(chunks, buf.String())
        // buat overlap: ambil tail dari chunk sebelumnya
        tail := tailOverlap(buf.String(), overlap)
        buf.Reset()
        buf.WriteString(tail)
        buf.WriteString(s)
      }
    }
    if buf.Len() > 0 { chunks = append(chunks, buf.String()) }
    return chunks
  }
  ```

### 20.5 Embedding Batch & Persistensi
Tujuan: hasilkan embedding secara efisien dan simpan ke Postgres.
- Batch:
  - `EmbedBatch(chunks)` untuk meminimalkan overhead jaringan.
  - `BATCH_SIZE` adaptif (misal 16–64) sesuai batas provider.
- Paralelisme:
  - Gunakan worker pool dengan batas concurrency (`EMBED_CONCURRENCY`).
  - Pastikan urutan hasil tetap selaras dengan `chunk_index`.
- Idempoten:
  - `publish` menggunakan idempoten key: `entry_id:version` untuk mencegah duplikasi.
- Persistensi:
  - Simpan ke `kb_chunks(entry_id, chunk_index, content, embedding)` dalam transaksi.
- Pseudokode:
  ```go
  func EmbedAndPersist(ctx context.Context, entryID string, chunks []string, emb EmbeddingProvider, db *sql.DB) error {
    vectors, err := emb.EmbedBatch(ctx, chunks)
    if err != nil { return err }
    return withTx(db, func(tx *Tx) error {
      for i, v := range vectors {
        if len(v) != emb.Dimension() { return fmt.Errorf("dim mismatch") }
        if err := insertChunk(tx, entryID, i, chunks[i], v); err != nil { return err }
      }
      return nil
    })
  }
  ```

### 20.6 Query & Retrieval
Tujuan: cari Top-K chunk relevan untuk query.
- Langkah:
  1) `Embed(query)` → vq
  2) SQL pgvector (cosine distance) dengan filter `entries.status='active'` dan optional `tags @> ARRAY[...]`
- Contoh SQL:
  ```sql
  SELECT c.entry_id, c.chunk_index, c.content,
         1 - (c.embedding <=> $1) AS score
  FROM kb_chunks c
  JOIN knowledge_entries e ON e.id = c.entry_id
  WHERE e.status = 'active'
    AND ($2::text[] IS NULL OR e.tags @> $2)
  ORDER BY c.embedding <=> $1
  LIMIT $3;
  ```
- Re-ranking heuristik: jika kategori/tags sama dengan query context, naikkan skor dengan faktor kecil.

### 20.7 Publish & Index Management (POST /v1/kb/entries/{id}/publish)
1) Ambil entry (status harus `draft` atau `active`).
2) Normalisasi + chunking → embed → simpan chunk.
3) Pastikan indeks ivfflat tersedia:
   - Jika belum, buat: `CREATE INDEX IF NOT EXISTS idx_kb_chunks_emb ON kb_chunks USING ivfflat (embedding vector_cosine_ops);`
   - Jalankan `ANALYZE kb_chunks;`
4) Update entry: `status=active`, `version++`, `updated_at=now()`.
5) Emit event `KBUpdated(entry_id)`.
6) Opsi async: jika `ASYNC_PUBLISH=true`, buat job di queue, endpoint mengembalikan `202 Accepted` + `jobId`.

### 20.8 Error Handling & Idempoten
- Idempoten publish: jika `kb_chunks` untuk `(entry_id, version)` sudah ada, jangan insert ulang; kembalikan sukses.
- Provider errors: retry eksponensial untuk `429`/`5xx` hingga maksimal N percobaan; fallback provider bila diaktifkan.
- Dimension mismatch: hard error, log + alarm; minta admin memperbaiki `EMBEDDING_DIM` pada konfigurasi atau migrasi ulang.
- Transaksi: semua insert chunk atomic; jika gagal, rollback.

### 20.9 Observability & Telemetri
- Metrik ingest:
  - `kb_publish_duration_ms` (end-to-end)
  - `kb_embedding_batch_ms`
  - `kb_chunks_count`
  - `kb_index_refresh_ms`
- Log berstruktur: `entry_id`, `version`, `provider`, `dimension`, `top_k`, `chunk_size`, `overlap`.
- Trace: span untuk normalize, chunk, embed, persist, index.

### 20.10 Pengujian (Test Cases)
- Unit: `NormalizeMarkdown`, `ChunkText` (deterministik, overlap benar, tidak melebihi size).
- Integration: `EmbedAndPersist` dengan mock/stub EmbeddingProvider (vector tetap), verifikasi dimensi dan urutan `chunk_index`.
- Retrieval: query Top-K dengan dataset kecil; assert peringkat stabil bila skor sama.
- Migration: buat tabel + ekstensi vector; pastikan tipe `vector(D)` konsisten.

### 20.11 Performa & Tuning
- `ivfflat` parameter `lists`: mulai dari 100–1000 (tergantung ukuran data); ANALYZE ulang setelah perubahan.
- `maintenance_work_mem` naik saat membuat indeks besar.
- Batasi panjang chunk maksimal agar embedding tidak memotong konteks (sesuai batas provider).
- Gunakan connection pooling; prepared statements untuk query similarity.

### 20.12 Keamanan & Privasi
- Sanitasi input: cegah SQL/markdown injection; batas ukuran konten.
- Redaksi PII opsional saat normalisasi.
- RBAC: hanya admin/editor dapat publish; employee dapat mengusulkan draft.
- Audit: simpan siapa yang publish dan perubahan metadata.

### 20.13 Contoh Potongan Kode (Go)
```go
// Handler publish KB entry
func (h *KBHandler) Publish(w http.ResponseWriter, r *http.Request) {
  id := mux.Vars(r)["id"]
  ctx := r.Context()
  entry, err := h.repo.FindEntry(ctx, id)
  if err != nil { httpError(w, 404, err); return }
  normalized := NormalizeMarkdown(entry.Content)
  chunks := ChunkText(normalized, h.cfg.ChunkSize, h.cfg.ChunkOverlap)
  if err := EmbedAndPersist(ctx, id, chunks, h.providers.Embeddings(), h.db); err != nil {
    httpError(w, 500, err); return
  }
  if err := h.indexer.EnsureIVFFlat(ctx); err != nil { httpError(w, 500, err); return }
  if err := h.repo.MarkActive(ctx, id); err != nil { httpError(w, 500, err); return }
  emitEvent(ctx, "KBUpdated", id)
  writeJSON(w, map[string]any{"id": id, "status": "active", "chunks": len(chunks)})
}

// Query Top-K untuk AI Suggest (digunakan di AISuggestionService)
func (r *KBRepo) QueryTopK(ctx context.Context, query string, topK int, tags []string) ([]KBChunk, error) {
  vq, err := r.embed.Embed(ctx, query)
  if err != nil { return nil, err }
  // gunakan pq.Array(tags) untuk parameter $2 bila diperlukan
  rows, err := r.db.QueryContext(ctx, `
    SELECT c.entry_id, c.chunk_index, c.content, 1 - (c.embedding <=> $1) AS score
    FROM kb_chunks c
    JOIN knowledge_entries e ON e.id = c.entry_id
    WHERE e.status = 'active' AND ($2::text[] IS NULL OR e.tags @> $2)
    ORDER BY c.embedding <=> $1
    LIMIT $3;
  `, vq, pq.Array(tags), topK)
  if err != nil { return nil, err }
  defer rows.Close()
  var out []KBChunk
  for rows.Next() {
    var ch KBChunk
    if err := rows.Scan(&ch.EntryID, &ch.Index, &ch.Content, &ch.Score); err != nil { return nil, err }
    out = append(out, ch)
  }
  return out, rows.Err()
}
```

---

*End of plan v1.0*
* **AIIntakeCreateTicketUseCase**

    * Input: AICreateTicketDTO `{ description (required), createdBy (required), title?, category?, priority?, flags: { autoCategorize?, autoPrioritize?, autoTitleFromAI? } }`
    * Flow:
      1. Call `AISuggestionService.SuggestMitigation(description)` untuk memperoleh `AIInsight` (mitigation + confidence) dan prediksi atribut (kategori/prioritas/judul ringkas) bila tersedia.
      2. Terapkan override terkontrol per field menggunakan ambang kepercayaan: `categoryConfidence ≥ 0.7`, `priorityConfidence ≥ 0.7`, `titleQualityScore ≥ 0.6`.
      3. Bangun payload tiket final dari input user + override AI; set `status=OPEN`; sematkan `aiInsight`.
      4. Persist menggunakan `TicketRepository.Create()` dalam transaksi; emit event `TicketCreated`.
      5. Return `TicketDTO` + `overrideMeta` (field yang diisi AI beserta confidence) untuk audit.
    * Output: `TicketDTO` dengan `AIInsight` + `overrideMeta`.
