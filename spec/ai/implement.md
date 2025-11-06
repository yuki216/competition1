# /speckit.implement — Knowledge Base (KB) Implementasi

Versi: 1.0
Terkait: /speckit.specify, /speckit.plan

## 1. Tujuan & Scope
Implementasi Knowledge Base berbasis pgvector untuk ingest teks/markdown, publish (normalize → chunk → embed → index), retrieval Top-K untuk AI Suggest, serta integrasi SSE untuk streaming kandidat rekomendasi. Mendukung multi-provider embedding (OpenAI & Z.ai) dan siap produksi: idempoten, terukur, dan terobservasi.

## 2. Komponen & Interfaces
- Normalizer: konversi markdown → teks; redaksi opsional PII; normalisasi whitespace.
- Chunker: pemecahan teks dengan ukuran dan overlap terkonfigurasi; menjaga batas kalimat.
- EmbeddingProvider (dari plan): OpenAI/Z.ai adapters, Embed/EmbedBatch, Dimension.
- KnowledgeRepository:
  - FindEntry(id), UpsertEntry(entry), InsertChunks(entryID, []Chunk), MarkActive(id), QueryTopK(query, topK, tags).
- Indexer: EnsureIVFFlat(), AnalyzeChunks().
- Services:
  - KBIngestService: upload text → save draft entry.
  - KBPublishService: normalize → chunk → embed → persist → index → activate.
  - KBQueryService: embed query → pgvector Top-K → re-ranking → format kandidat.
- HTTP Handlers:
  - POST /v1/kb/entries, POST /v1/kb/entries/upload-text, PATCH /v1/kb/entries/{id}, POST /v1/kb/entries/{id}/publish.
  - POST /v1/ai/suggest (JSON), GET /v1/ai/suggest/stream (SSE).

## 3. Data Schema & Migrations
- Postgres: `knowledge_entries(id, title, content, status enum draft|active|archived, tags text[], source_type, version, created_by, created_at, updated_at)`.
- `kb_chunks(id, entry_id fk, chunk_index, content text, embedding vector(D), created_at)`.
- Extension: `CREATE EXTENSION IF NOT EXISTS vector;`.
- Index: `CREATE INDEX IF NOT EXISTS idx_kb_chunks_emb ON kb_chunks USING ivfflat (embedding vector_cosine_ops);`.
- Konsistensi: `D` mengikuti `EmbeddingProvider.Dimension()`.

## 4. Flow Detail
### 4.1 Upload Text (POST /v1/kb/entries/upload-text)
1) Validasi payload: title, content (text/markdown), tags opsional.
2) NormalizeMarkdown(content) → simpan entry status `draft`.
3) Response: `201 Created` dengan `id`, `status=draft`.
4) Untuk konten besar (>1MB), sarankan `async publish` dengan job queue.

### 4.2 Publish (POST /v1/kb/entries/{id}/publish)
1) Ambil entry (status draft/active), set `version++` untuk publish terbaru.
2) NormalizeMarkdown → ChunkText(size, overlap) → EmbedBatch(chunks).
3) Persist chunks dalam transaksi; idempoten per `(entry_id, version)`.
4) Indexer.EnsureIVFFlat() → AnalyzeChunks().
5) Update entry ke `active` + emit event `KBUpdated(entry_id)`.
6) Bila `ASYNC_PUBLISH=true`, enque job dan return `202 Accepted + jobId`.

### 4.3 Retrieval (KBQueryService)
1) vq = Embed(query).
2) SQL pgvector cosine distance dengan filter status `active` dan optional tags.
3) Re-ranking heuristik ringan: boost jika kategori/tags cocok.
4) Format `SuggestCandidate`: {text, sourceEntryId, chunkIndex, score, tags}.

### 4.4 SSE Integration untuk AI Suggest
- GET /v1/ai/suggest/stream:
  - Event sequence: `init` → banyak `candidate` → `end` atau `error`.
  - Heartbeat komentar `:heartbeat` ≤ 15 detik; flush ≤ 250ms.
  - Reconnect idempoten via `queryId` optional parameter.

## 5. Error Handling & Retry
- Provider 429/5xx: retry eksponensial, batas percobaan; fallback ke provider lain bila diaktifkan.
- Dimension mismatch: hard error; minta rekonsiliasi `EMBEDDING_DIM` dan migrasi/konfigurasi ulang.
- Transaksi insert chunks: atomic; rollback jika ada kegagalan.
- Validasi input: ukuran konten, karakter ilegal, sanitasi markdown.

## 6. Konfigurasi
- `AI_PROVIDER=openai|zai`.
- `EMBEDDING_DIM=1536` (default; konsisten dengan provider).
- `TOP_K=10`, `CHUNK_SIZE=800`, `CHUNK_OVERLAP=150`.
- `SSE_FLUSH_MS=250`, `SSE_HEARTBEAT_MS=15000`.
- `EMBED_CONCURRENCY=8` (contoh; sesuaikan CPU & rate limit).

## 7. Observability
- Metrics: `kb_publish_duration_ms`, `kb_embedding_batch_ms`, `kb_chunks_count`, `kb_index_refresh_ms`, `first_event_ms`, `events_count`, `total_duration_ms`.
- Logs berstruktur: `entry_id`, `version`, `provider`, `dimension`, `chunk_size`, `overlap`, `top_k`.
- Tracing: span untuk normalize, chunk, embed, persist, index, query, sse_send.

## 8. Security & Privacy
- RBAC: admin/editor boleh publish; employee hanya draft.
- PII redaction opsional saat normalisasi.
- Rate limit: per user & per provider; circuit breaker pada error bertubi.
- Audit: simpan actor & metadata pada publish.

## 9. Testing & QA
- Unit: NormalizeMarkdown, ChunkText (deterministik, overlap benar).
- Integration: EmbedAndPersist (mock provider), EnsureIVFFlat, QueryTopK.
- SSE: urutan event, heartbeat, `first_event_ms` target.
- Provider parity: OpenAI & Z.ai memenuhi kontrak EmbeddingProvider.
- E2E kecil: upload → publish → suggest(JSON+SSE) menghasilkan kandidat valid.

## 10. Acceptance Criteria
- Publish menghasilkan chunks dengan dimensi konsisten dan index siap (ivfflat + ANALYZE).
- Query Top-K mengembalikan kandidat dengan skor dan sumber entry yang benar dalam ≤ target latensi.
- SSE stream mengalirkan minimal 1 kandidat dalam ≤ target `first_event_ms` dan menutup dengan `end`.
- Env/konfigurasi dapat mengubah provider dan parameter (size/overlap/top_k) tanpa kode ulang.
- Tes lulus dengan cakupan minimal sesuai DoD.

## 11. Rencana Rollout
- Tahap 1: Migrations & adapter embedding mock; jalankan publish & retrieval lokal.
- Tahap 2: Integrasi provider OpenAI; verifikasi latensi & biaya; tambahkan fallback Z.ai.
- Tahap 3: SSE stabilisasi; observability; tuning ivfflat (lists) & ANALYZE.
- Tahap 4: Hardening security & audit; dokumentasi pengguna.

## 12. Risiko & Mitigasi
- Kuota provider habis: rate limit & fallback; cache embedding jika diperlukan.
- Dimensi tidak konsisten: validasi dan fail-fast; migrasi ulang data.
- Latensi tinggi: batch & concurrency; tuning ivfflat; kurangi CHUNK_SIZE bila perlu.

## 13. AI Intake — Auto Create Ticket dari Saran AI

### 13.1 Handler & Routing
- Endpoint: `POST /v1/tickets/ai-intake`
- Handler: `handler_ticket_ai_intake.go`
  - Validasi body: `description` (wajib), `created_by` (wajib), opsional: `title`, `category`, `priority`, flags `autoCategorize`, `autoPrioritize`, `autoTitleFromAI`.
  - Panggil `AIIntakeCreateTicketUseCase` (application layer) dengan DTO yang sama.
  - Return `201 Created` dengan `ticket`, `ai_insight`, `override_meta`.

### 13.2 Use Case (Application Layer)
- Nama: `AIIntakeCreateTicketUseCase`
- Flow:
  1. `AISuggestionService.SuggestMitigation(description)` → dapatkan `mitigation`, `confidence`, serta prediksi atribut (jika tersedia) dari model/heuristik.
  2. Bangun objek `AIInsight{ text: mitigation, confidence }` dan attach ke tiket.
  3. Terapkan override terkontrol per field:
     - `category` bila `categoryConfidence ≥ 0.7` atau flag `autoCategorize=true` dengan confidence memenuhi.
     - `priority` bila `priorityConfidence ≥ 0.7` atau flag `autoPrioritize=true`.
     - `title` ringkas bila `titleQualityScore ≥ 0.6` atau flag `autoTitleFromAI=true` dengan skor memenuhi.
  4. Persist tiket dengan `TicketRepository.Create()`; emit event `TicketCreated`.
  5. Return `TicketDTO` + `overrideMeta` (field diisi AI + confidence + alasan).

Catatan: Saran AI ditopang oleh KB vector (pgvector) sesuai implementasi di dokumen ini; Suggestion Service melakukan Top-K retrieval pada `kb_chunks` yang telah dipublish.

### 13.3 Aturan & Validasi
- Minimal field: `description`, `created_by`.
- Jika input `category/priority` kosong namun AI confidence tinggi, sistem boleh mengisi otomatis.
- Audit metadata menyimpan asal nilai (USER vs AI) dan confidence.
- Rate limit intake agar tidak menjadi spam tiket.

## 14. Konfigurasi Khusus Intake
- `AI_INTake_CATEGORY_THRESHOLD=0.7`
- `AI_INTake_PRIORITY_THRESHOLD=0.7`
- `AI_INTake_TITLE_THRESHOLD=0.6`
- `AI_INTake_ENABLED=true`
- `AI_INTake_MAX_PER_USER_PER_MIN=10` (rate limit contoh)

## 15. Observability (Intake)
- Metrics: `ai_intake_request_count`, `ai_intake_created_count`, `ai_intake_override_count`, `ai_intake_latency_ms`.
- Logs: `user_id`, `ticket_id`, `override_meta`, `confidence`, `provider`.
- Tracing: span untuk `suggest_call`, `override_apply`, `ticket_create`.

## 16. Testing (Intake)
- Unit: aturan override per field vs threshold; fallback ke nilai user jika confidence rendah.
- Integration: alur end-to-end intake → repository `Create()` terpanggil dan audit metadata tersimpan.
- Negative: deskripsi kosong, provider error (5xx/429), rate limit tercapai.
- Contract: DTO input/output sesuai dengan `spec/ai/specify.md` dan `plan.md`.