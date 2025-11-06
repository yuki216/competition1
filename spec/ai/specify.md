# /speckit.specify — Fixora IT Ticketing

### Tujuan

Menentukan kapabilitas bisnis, kebutuhan fungsional/non-fungsional, aktor, dan acceptance criteria sistem IT Ticketing berbasis AI (Fixora) sebelum masuk tahap desain domain (`plan`).

---

## 1. Business Capability Map

| Capability                         | Deskripsi                                           | Bounded Context             |
| ---------------------------------- | --------------------------------------------------- | --------------------------- |
| **Ticket Submission**              | Employee membuat dan memantau tiket                 | Ticket Management           |
| **AI Assistance**                  | AI membantu klasifikasi & mitigasi awal             | AI Context / Knowledge Base |
| **Comment & Collaboration**        | Diskusi antar employee & admin                      | Commenting Context          |
| **Ticket Assignment & Resolution** | Admin assign, update, dan menyelesaikan tiket       | Ticket Management           |
| **Knowledge Management**           | Admin mengatur konteks solusi agar AI dapat belajar | Knowledge Base              |
| **Reporting & Insights**           | Admin melihat metrik performa                       | Reporting Context           |

---

## 2. Primary Actors

| Actor                     | Role      | Objective                                                           |
| ------------------------- | --------- | ------------------------------------------------------------------- |
| **Employee**              | Requester | Melaporkan masalah IT dan mendapat solusi cepat                     |
| **AI System (Fixora.AI)** | Assistant | Menganalisis, memberikan saran mitigasi, dan membuat tiket otomatis |
| **Admin / IT Support**    | Resolver  | Menindaklanjuti, mengomentari, dan menutup tiket                    |
| **IT Manager**            | Observer  | Melihat performa SLA dan tren masalah                               |

---

## 3. Core Use Cases

### 3.1 Submit Ticket

**Trigger:** Employee melaporkan masalah melalui portal atau chat AI.
**Main Flow:**

1. Employee mengetik deskripsi masalah (misal: *“Laptop tidak bisa connect Wi-Fi”*).
2. AI menganalisis deskripsi → mengklasifikasikan kategori (network/software/hardware).
3. Jika confidence tinggi, AI menampilkan mitigasi awal.
4. Employee memilih “Buat tiket” jika masalah belum terselesaikan.
5. Sistem menyimpan tiket dengan status `OPEN`.

**Alternative:**
AI membuat tiket otomatis jika perintah mengandung kata aksi (*“tolong buatkan tiket”*).

**Acceptance Criteria:**

* [ ] Tiket disimpan dengan status `OPEN` dan ID unik.
* [ ] AI Insight tersimpan bila confidence > 0.7.
* [ ] Employee menerima konfirmasi & bisa melihat tiket di dashboard.

---

### 3.2 AI Suggest Mitigation

**Trigger:** AI menerima deskripsi masalah dari employee.
**Main Flow:**

1. Sistem mengirimkan deskripsi ke `AISuggestionService`.
2. AI melakukan similarity search terhadap embedding `KnowledgeBase` yang disimpan di Postgres (pgvector).
3. AI mengembalikan rekomendasi solusi (jika ditemukan) beserta confidence score.
4. Solusi ditampilkan ke employee sebelum tiket dibuat. Opsional: rekomendasi dikirim bertahap via SSE (Server-Sent Events) untuk mempercepat time-to-first-hint.

**Acceptance Criteria:**

* [ ] AI memberikan saran yang mengandung langkah konkret (bukan teks generik).
* [ ] AI hanya memberikan saran dari konteks yang sudah disetujui admin.
* [ ] AI menolak menjawab jika confidence < 0.4 (minta klarifikasi).

---

### 3.2a AI Intake — Auto Create Ticket dari Saran AI

**Trigger:** Employee meminta sistem membuat tiket setelah melihat saran AI (atau menginstruksikan langsung via chat: "buatkan tiket").

**Main Flow:**

1. Employee mengirim deskripsi ke AI dan memilih opsi "Buatkan Tiket" atau menyertakan instruksi eksplisit.
2. Sistem melakukan analisis AI (kategori, prioritas, judul ringkas) dan menghasilkan `AIInsight` dengan confidence.
3. Sistem memutuskan override terkontrol berdasarkan ambang kepercayaan:
   - `category` diisi dari AI jika `categoryConfidence ≥ 0.7`, selain itu gunakan input user/default.
   - `priority` diisi dari AI jika `priorityConfidence ≥ 0.7`.
   - `title` dapat disarankan AI (ringkas), gunakan jika `titleQualityScore ≥ 0.6`.
4. Sistem membuat tiket (`status=OPEN`) menggunakan field akhir + menyematkan `AIInsight`.
5. Sistem mengembalikan tiket yang dibuat beserta AIInsight.

**Alternative:**
- Jika confidence AI di bawah ambang untuk semua field, sistem hanya menempelkan `AIInsight` tanpa override, tetap membuat tiket berdasarkan input user.
- Jika employee belum memberikan field minimal (mis. kategori/prioritas), sistem dapat mengisi menggunakan prediksi AI bila confidence cukup.

**Acceptance Criteria:**

* [ ] Tiket berhasil dibuat (`OPEN`) dengan `AIInsight` yang mencerminkan saran.
* [ ] Override field hanya terjadi bila confidence per field memenuhi ambang yang ditentukan.
* [ ] Semua keputusan override dicatat dalam audit metadata (field, nilai AI, confidence).
* [ ] Endpoint intake tersedia dan tervalidasi inputnya (lihat §9.6 API Surface).

---

### 3.3 Comment on Ticket

**Trigger:** Employee atau Admin menambahkan komentar pada tiket.
**Main Flow:**

1. Komentar dikirim ke endpoint `/tickets/{id}/comments`.
2. Sistem menyimpan komentar dengan role (EMPLOYEE/ADMIN).
3. Notifikasi dikirim ke pihak lain yang terlibat.

**Acceptance Criteria:**

* [ ] Komentar tersimpan dengan timestamp & author.
* [ ] Sistem mengirim notifikasi real-time (email/Slack).
* [ ] AI dapat menambahkan komentar otomatis (role: AI) jika diperlukan.

---

### 3.4 Assign Ticket

**Trigger:** Admin memilih tiket dan meng-assign ke anggota tim.
**Main Flow:**

1. Admin membuka tiket status `OPEN`.
2. Memilih anggota tim yang relevan.
3. Sistem memperbarui status ke `IN_PROGRESS`.

**Acceptance Criteria:**

* [ ] Hanya admin dapat mengubah assignment.
* [ ] Assignment dicatat dalam audit trail.
* [ ] Notifikasi dikirim ke assignee.

---

### 3.5 Resolve Ticket

**Trigger:** Admin menyelesaikan tiket.
**Main Flow:**

1. Admin menambahkan mitigasi akhir & komentar penyelesaian.
2. Sistem memperbarui status ke `RESOLVED`.
3. AI mempelajari tiket & solusi melalui `AITrainingService`.

**Acceptance Criteria:**

* [ ] Status berubah ke `RESOLVED`.
* [ ] AI menerima payload tiket & komentar terakhir untuk learning.
* [ ] SLA resolution time tercatat.

---

### 3.6 Knowledge Management (Admin)

**Trigger:** Admin menambahkan atau memperbarui solusi di knowledge base.
**Main Flow:**

1. Admin membuka halaman Knowledge Base.
2. Menambahkan context baru (“VPN error”) dengan solusi.
3. Sistem menyimpan entri dan sinkron ke AI.

**Acceptance Criteria:**

* [ ] Entry tersimpan dengan status “active”.
* [ ] Update otomatis memicu refresh ke `AISuggestionService`.
* [ ] Perubahan tercatat di audit log.

---

### 3.7 View Reports

**Trigger:** Admin/Manager membuka dashboard.
**Main Flow:**

1. Sistem menampilkan metrik: total tiket, SLA compliance, AI success rate.
2. Data ditarik dari `ReportingService`.

**Acceptance Criteria:**

* [ ] Metrik bisa difilter berdasarkan periode waktu & kategori.
* [ ] AI accuracy dihitung dari feedback user terhadap solusi.

---

## 4. Non-Functional Requirements (NFR)

| Area              | Target                                                        |
| ----------------- | ------------------------------------------------------------- |
| **Performance**   | Response ≤ 300ms untuk operasi non-AI                         |
| **AI Inference**  | Max latency 2s untuk saran mitigasi                           |
| **Vector Search** | Top-K=10 dalam ≤ 200ms pada ≥ 100k chunks menggunakan approximate index (ivfflat) |
| **SSE Streaming** | Time-to-first-event ≤ 500ms; flush cadence ≤ 250ms; heartbeat setiap ≤ 15s |
| **Reliability**   | 99.9% uptime                                                  |
| **Security**      | Data tiket hanya dapat diakses oleh pembuat & tim IT          |
| **Auditability**  | Semua perubahan status, assignment, dan komentar tercatat     |
| **Scalability**   | Dapat menangani ≥ 10.000 tiket aktif tanpa degradasi performa |
| **Observability** | Correlation ID di setiap request; logging structured JSON     |

---

## 5. Domain Terminology (Ubiquitous Language)

| Istilah            | Definisi                                                          |
| ------------------ | ----------------------------------------------------------------- |
| **Ticket**         | Representasi formal dari masalah IT yang dilaporkan oleh employee |
| **Mitigation**     | Tindakan sementara untuk mengurangi dampak masalah                |
| **Resolution**     | Langkah akhir penyelesaian yang menutup tiket                     |
| **Knowledge Base** | Kumpulan konteks masalah dan solusi yang digunakan AI             |
| **AI Insight**     | Hasil analisis AI berupa saran mitigasi dan confidence            |
| **Comment**        | Catatan percakapan antara pihak terkait tiket                     |
| **Assignment**     | Penunjukan tanggung jawab tiket kepada admin tertentu             |
| **Embedding**      | Representasi numerik berdimensi tetap dari teks (vector) yang dipakai untuk pencarian kemiripan |
| **Vector Index**   | Struktur indeks untuk mempercepat pencarian kemiripan vektor (mis. ivfflat pada pgvector) |
| **Metric**         | Data performa sistem (SLA, response time, AI accuracy)            |

---

## 6. User Story Summary

| ID    | As a      | I want to                                  | So that                             |
| ----- | --------- | ------------------------------------------ | ----------------------------------- |
| US-01 | Employee  | Submit issue and get immediate suggestion  | I can fix small issues faster       |
| US-02 | AI System | Suggest mitigation based on knowledge base | Employee gets faster support        |
| US-03 | Admin     | Assign ticket to IT staff                  | Workload is distributed efficiently |
| US-04 | Admin     | Add solution to knowledge base             | AI becomes smarter over time        |
| US-05 | Manager   | View metrics dashboard                     | I can monitor IT performance        |

---

## 7. Acceptance Checklist (Definition of Ready)

* [ ] Semua use case di atas memiliki DTO input/output terdefinisi.
* [ ] Setiap entity domain memiliki invariant & relasi dasar.
* [ ] Error catalog awal disiapkan (`pkg/errorcodes/`).
* [ ] Knowledge base schema dan AI integration didefinisikan di `plan.md`.
* [ ] Postgres + pgvector dipilih sebagai penyimpanan embedding; DDL dasar disepakati.
* [ ] SSE streaming untuk `/v1/ai/suggest/stream` memiliki spesifikasi event & skenario uji.
* [ ] Endpoint REST sesuai di `api/openapi.yaml`.
* [ ] SLA metrics sudah punya definisi formula & sumber data.
* [ ] Feedback loop AI sudah punya alur pembelajaran minimal (resolve → train).
* [ ] Mock service tersedia untuk `AISuggestionService` & `AITrainingService`.
* [ ] Unit test plan untuk domain & use case diset sebelum coding.

---

## 8. Next Step

1. Sinkronkan endpoint dari use case ini ke `api/openapi.yaml`.
2. Turunkan struktur domain & ports ke `/speckit.plan`.
3. Definisikan katalog error (`pkg/errorcodes/`).
4. Siapkan data mock (tickets, comments, knowledge entries) untuk testing awal.
5. Jalankan sesi review domain ubiquitous sebelum implementasi usecase pertama.

---

## 9. Knowledge Base dengan Embedding Vektor (Postgres)

### 9.1 Tujuan

- Memungkinkan AI melakukan pencarian solusi berbasis kemiripan semantik.
- Menyimpan dan mengelola konten knowledge base secara terstruktur dengan kemampuan vektor search yang efisien.

### 9.2 Kebutuhan Fungsional

- Admin dapat membuat, memperbarui, dan mem-publish entri knowledge base.
- Sistem melakukan chunking konten panjang menjadi potongan kecil (mis. 500–1000 karakter) untuk kualitas embedding yang lebih baik.
- Sistem menghasilkan embedding untuk setiap chunk dan menyimpannya di Postgres dengan tipe `vector(D)`.
- AI Suggestion melakukan Top-K similarity search (cosine distance) terhadap embedding yang berstatus `active/published`.
- Dapat memfilter hasil berdasarkan kategori, status, atau tag.
- Re-embed & re-index otomatis saat entri diperbarui.

### 9.3 Kebutuhan Non-Fungsional Khusus KB

- Latency pencarian: ≤ 200ms untuk Top-K=10 pada ≥ 100k chunks dengan ivfflat.
- Konsistensi: publish harus atomic (konten + embedding tersedia bersamaan).
- Observability: log `query_time_ms`, `top_k`, `avg_score`, dan jumlah kandidat.
- Keamanan: hanya AI service yang memiliki akses read ke tabel embedding; hanya Admin yang dapat publish.

### 9.4 Skema Data (Konseptual)

- `knowledge_entries`
  - id (uuid), title (text), content (text), status (enum: draft|active|archived), category (text), tags (text[]), source_type (text), version (int), created_by (uuid), created_at (timestamptz), updated_at (timestamptz)
- `kb_chunks`
  - id (uuid), entry_id (uuid FK → knowledge_entries), chunk_index (int), content (text), embedding (vector(D)), created_at (timestamptz)

Catatan: D (dimension) dikonfigurasi mengikuti model embedding yang dipilih. Nilai umum: 768–1536.

### 9.5 DDL Contoh (Postgres + pgvector)

Prasyarat: `CREATE EXTENSION IF NOT EXISTS vector;`

```
-- Tabel entri KB
CREATE TABLE IF NOT EXISTS knowledge_entries (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','active','archived')),
  category TEXT,
  tags TEXT[],
  source_type TEXT,
  version INT NOT NULL DEFAULT 1,
  created_by UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabel chunks + embedding
-- Ganti 1536 dengan dimensi model embedding yang digunakan
CREATE TABLE IF NOT EXISTS kb_chunks (
  id UUID PRIMARY KEY,
  entry_id UUID NOT NULL REFERENCES knowledge_entries(id) ON DELETE CASCADE,
  chunk_index INT NOT NULL,
  content TEXT NOT NULL,
  embedding VECTOR(1536),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indeks untuk pencarian cepat
CREATE INDEX IF NOT EXISTS idx_kb_chunks_entry ON kb_chunks(entry_id);
CREATE INDEX IF NOT EXISTS idx_kb_entries_status ON knowledge_entries(status);

-- Approximate vector index (ivfflat) menggunakan cosine distance
-- Pastikan ANALYZE setelah populate untuk optimasi list
CREATE INDEX IF NOT EXISTS idx_kb_chunks_embedding_ivfflat
  ON kb_chunks USING ivfflat (embedding vector_cosine_ops);
```

### 9.6 API Surface (Ringkas)

- `POST /kb/entries` → buat entri (draft) menggunakan body JSON `{title, content, category, tags}`
- `POST /kb/entries/upload-text` → upload teks/Markdown (Content-Type: `text/plain` atau `text/markdown`), param opsional: `title`, `category`, `tags`, `publish=true|false`
- `PATCH /kb/entries/{id}` → update konten
- `POST /kb/entries/{id}/publish` → publish + jalankan embedding & indexing
- `POST /v1/ai/suggest` → input: `query`, optional `filters`, `top_k`; output: daftar rekomendasi dengan `content`, `score`, `entry_id`, `chunk_index` (JSON)
- `GET  /v1/ai/suggest/stream` → SSE, header: `Content-Type: text/event-stream`; query: `query`, optional `filters`, `top_k`. Mengirim event `init`, banyak `candidate`, `progress`, `end`, `error`.
  - Catatan: endpoint legacy `POST /ai/suggest` dapat dipertahankan sebagai alias non-versioned (opsional).

- `POST /v1/tickets/ai-intake` → membuat tiket berbasis analisis AI. Body:
  - `description` (wajib), `created_by` (wajib), `title` (opsional), `category` (opsional), `priority` (opsional)
  - flag opsional: `autoCategorize`, `autoPrioritize`, `autoTitleFromAI`
  - aturan override: hanya bila confidence per field memenuhi ambang (default: 0.7 untuk kategori/prioritas, 0.6 untuk judul).
  - Response: `ticket` + `ai_insight` + `override_meta` (field yang diisi AI + confidence)

### 9.7 Alur Ingest & Retrieval

1. Admin membuat/ubah entri melalui JSON atau upload teks (text/plain/Markdown) → status draft.
2. Saat publish: sistem normalisasi teks (hapus artefak HTML, normalisasi whitespace, pertahankan heading) → chunking → generate embeddings → simpan ke `kb_chunks` → buat/refresh index.
3. AI Suggestion menerima query → embed → similarity search Top-K pada `kb_chunks` (status entry = active) → re-ranking heuristik (mis. skor + metadata kecocokan kategori) → hasil dikirim ke UI.

### 9.8 Acceptance Criteria (KB Vector)

- [ ] Query Top-K=10 mengembalikan kandidat relevan dengan rata-rata skor ≥ 0.6 pada dataset uji internal.
- [ ] Publish atomic: tidak ada state di mana entri aktif tanpa embedding yang tersedia.
- [ ] Re-embed untuk 10.000 chunks selesai ≤ 5 menit.
- [ ] Hanya entri berstatus `active` yang dipakai dalam pencarian.
- [ ] Audit log menyimpan perubahan publish dan versi entri.

### 9.9 Operasional & Observability

- Metrik: `kb_vector_search_latency_ms`, `kb_index_size`, `kb_chunks_count`.
- Prosedur: VACUUM/ANALYZE berkala pada tabel `kb_chunks` dan `knowledge_entries`.
- Backup: snapshot harian database; uji restore untuk konsistensi index.

### 9.10 Alur Upload Teks (Detail)

**Flow:**

1. Admin memanggil `POST /kb/entries/upload-text` dengan header `Content-Type: text/plain` atau `text/markdown` dan menyertakan `title` (wajib), `category`/`tags` (opsional), serta `publish=true|false`.
2. Sistem membuat `knowledge_entries` (status `draft` secara default).
3. Sistem melakukan normalisasi teks dan memecah konten menjadi chunk (500–1000 karakter, overlap 100–200).
4. Sistem menghasilkan embedding per chunk dan menyimpannya di `kb_chunks` dengan tipe `VECTOR(D)`.
5. Jika `publish=true`: proses publish bersifat atomic (entry menjadi `active` setelah seluruh embedding tersimpan dan indeks tersedia).
6. Untuk konten besar (mis. > 2MB), proses embedding dijalankan asinkron dan API mengembalikan `job_id` untuk pemantauan.

**Acceptance Criteria:**

- [ ] Upload teks berhasil membuat entri dengan `title` dan `content` yang tersimpan.
- [ ] Normalisasi tidak menghilangkan struktur semantik penting (heading, list) pada Markdown.
- [ ] Chunking dan embedding dilakukan sesuai parameter default; parameter dapat dikonfigurasi di `plan.md`.
- [ ] Jika publish sinkron, entri `active` hanya setelah seluruh embedding tersedia dan index siap.
- [ ] Jika publish asinkron, tersedia API untuk cek status job dan entri tetap `draft` hingga job selesai.

### 9.11 SSE Streaming untuk AI Suggest

**Tujuan:** Memberikan saran mitigasi secara bertahap sehingga pengguna memperoleh petunjuk awal lebih cepat sambil pencarian Top-K berlangsung.

**Endpoint:** `GET /v1/ai/suggest/stream`

**Headers (Server Response):**
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

**Params:**
- `query` (string, wajib)
- `filters` (object atau key-value optional: `category`, `tags`, `status`)
- `top_k` (int, default 10)

**Event Types & Payload (data berformat JSON):**
- `event: init`
  - `data: { queryId, topK, filters, startTime }`
- `event: candidate`
  - `data: { rank, score, entry_id, chunk_index, content_snippet, category, tags }`
- `event: progress`
  - `data: { retrievedCount, elapsed_ms }`
- `event: end`
  - `data: { totalCandidates, elapsed_ms }`
- `event: error`
  - `data: { code, message }`

**Heartbeat:**
- Kirim komentar SSE (`:heartbeat`) setiap ≤ 15 detik untuk menjaga koneksi tetap hidup.

**Retry:**
- Klien dapat menggunakan `retry: 3000` (3 detik) dan akan otomatis reconnect. Server harus idempotent terhadap reconnect (menggunakan `queryId`).

**Security & Rate Limit:**
- Autentikasi wajib (token/Session) dan CORS diizinkan hanya dari origin yang tepercaya.
- Batasi rate pada endpoint stream (mis. 5 koneksi aktif per user) untuk mencegah resource exhaustion.

**Acceptance Criteria (SSE):**
- [ ] First event (`init`) terkirim ≤ 500ms setelah request.
- [ ] Minimal 3 `candidate` events terkirim dengan `rank` berurutan sebelum `end`.
- [ ] Heartbeat dikirim reguler dan koneksi tetap stabil pada idle ≥ 1 menit.
- [ ] `error` event dikirim dengan kode yang jelas saat kegagalan terjadi (mis. invalid filters, rate limit).
- [ ] Observabilitas: logging `queryId`, `events_count`, `first_event_ms`, dan `total_duration_ms`.
