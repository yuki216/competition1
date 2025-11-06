# /speckit.tasks — Knowledge Base (KB) Backlog Implementasi

Versi: 1.0
Terkait: /speckit.plan, /speckit.implement

Catatan: Prioritas [High|Medium|Low]. Semua tugas harus menghasilkan test & dokumentasi singkat pada PR.

## Epic A — Migrations & Schema [High]
- A1: Buat migrasi Postgres untuk `knowledge_entries` dan `kb_chunks` (kolom, tipe, constraints). [High]
- A2: Tambahkan `CREATE EXTENSION IF NOT EXISTS vector;` dalam migrasi awal. [High]
- A3: Buat indeks `ivfflat` pada `kb_chunks.embedding` dengan `vector_cosine_ops`. [High]
- A4: Tambahkan indeks pendukung: `idx_kb_entries_status`, `idx_kb_chunks_entry`. [Medium]
- A5: Skrip `ANALYZE kb_chunks` pasca publish (ops: job berkala). [Medium]

## Epic B — Provider Adapters (OpenAI & Z.ai) [High]
- B1: Implement `EmbeddingProvider` adapter untuk OpenAI (Embed, EmbedBatch, Dimension). [High]
- B2: Implement `EmbeddingProvider` adapter untuk Z.ai (paritas kontrak). [High]
- B3: Konfigurasi `AI_PROVIDER=openai|zai`; validasi `EMBEDDING_DIM`. [High]
- B4: Retry & backoff untuk 429/5xx; observability per provider. [High]
- B5: Provider failover (opsional) bila provider utama gagal. [Medium]

## Epic C — Services (Domain/Application) [High]
- C1: `KBIngestService` — validasi payload, NormalizeMarkdown, simpan draft. [High]
- C2: `KBPublishService` — normalize → chunk → embed → persist → index → active. [High]
- C3: `KBQueryService` — embed query → pgvector Top-K → re-ranking → kandidat. [High]
- C4: `AISuggestionService` gunakan `KBQueryService` untuk retrieval konteks. [Medium]
- C5: `Indexer` — EnsureIVFFlat & AnalyzeChunks; idempoten. [Medium]

## Epic D — HTTP Handlers & API [High]
- D1: POST `/v1/kb/entries` (create draft) + schema validation. [High]
- D2: POST `/v1/kb/entries/upload-text` (upload markdown/text) + normalisasi. [High]
- D3: PATCH `/v1/kb/entries/{id}` (update metadata, tags). [Medium]
- D4: POST `/v1/kb/entries/{id}/publish` (publish & index). [High]
- D5: POST `/v1/ai/suggest` (JSON) — gunakan KB retrieval. [High]
- D6: GET `/v1/ai/suggest/stream` (SSE) — init, candidate, heartbeat, end/error. [High]
- D7: Tambahkan auth (bearer/JWT) & correlation-id di semua handler. [High]
- D8: Sinkronkan `openapi.yaml` bila ada perubahan detail respons/error. [Medium]

## Epic E — SSE & Observability [High]
- E1: Implement streamer dengan flush cadence ≤ 250ms dan heartbeat ≤ 15s. [High]
- E2: Metrik: `first_event_ms`, `events_count`, `total_duration_ms`. [High]
- E3: Logging berstruktur untuk publish & stream; trace spans. [High]
- E4: Rate limit & circuit breaker pada AI/SSE. [Medium]

## Epic F — Testing [High]
- F1: Unit: `NormalizeMarkdown` dan `ChunkText` (deterministik, overlap, batas ukuran). [High]
- F2: Integration: `EmbedAndPersist` dengan mock provider; verifikasi dimensi & urutan chunk_index. [High]
- F3: Query Top-K: dataset kecil; skor & ordering stabil. [High]
- F4: SSE handler: urutan event, heartbeat, graceful end/error, `first_event_ms` target. [High]
- F5: Provider parity tests: OpenAI & Z.ai memenuhi kontrak & error mapping. [High]
- F6: E2E minimal: upload → publish → suggest (JSON+SSE) menghasilkan kandidat valid. [High]

## Epic G — Security & Ops [Medium]
- G1: RBAC publish (admin/editor); employee hanya draft. [Medium]
- G2: Redaksi PII opsional saat normalisasi; konfigurasi pola regex. [Medium]
- G3: Audit trail pada publish (actor, metadata). [Medium]
- G4: Backup & restore; uji `ivfflat` setelah restore. [Medium]

## Epic H — Docs & Developer Experience [Medium]
- H1: README KB — cara kerja, konfigurasi, batasan provider, biaya & rate limit. [Medium]
- H2: Contoh payload & respons untuk upload, publish, suggest (JSON/SSE). [Medium]
- H3: Playbook operasional: ANALYZE, tuning `lists`, pemantauan metrik. [Medium]
- H4: Contoh konfigurasi `.env` untuk OpenAI & Z.ai. [Medium]

## Catatan Task Teknis (Detail Implementasi)
- Parameter default: `EMBEDDING_DIM=1536`, `TOP_K=10`, `CHUNK_SIZE=800`, `CHUNK_OVERLAP=150`, `SSE_FLUSH_MS=250`, `SSE_HEARTBEAT_MS=15000`.
- Batch embedding: `BATCH_SIZE=16–64` sesuai batas provider; concurrency `EMBED_CONCURRENCY=8`.
- SQL retrieval (cosine) contoh disediakan di /speckit.plan §20.6.
- Indexer: `CREATE INDEX IF NOT EXISTS idx_kb_chunks_emb ON kb_chunks USING ivfflat (embedding vector_cosine_ops);` + `ANALYZE kb_chunks;`.

## Definisi Selesai (DoD)
- Semua endpoint (KB & AI Suggest + SSE) terimplementasi dan diuji; OpenAPI sinkron.
- Migrasi dibuat dan dijalankan; extension vector & index tersedia; ANALYZE sukses.
- Observability dasar aktif: metrik & logs tersedia; SSE `first_event_ms` diukur.
- Provider adapters berfungsi dengan fallback opsional; konfigurasi dapat diganti tanpa perubahan kode.