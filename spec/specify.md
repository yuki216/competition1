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
2. AI membandingkan dengan entri `KnowledgeBase`.
3. AI mengembalikan rekomendasi solusi (jika ditemukan) beserta confidence score.
4. Solusi ditampilkan ke employee sebelum tiket dibuat.

**Acceptance Criteria:**

* [ ] AI memberikan saran yang mengandung langkah konkret (bukan teks generik).
* [ ] AI hanya memberikan saran dari konteks yang sudah disetujui admin.
* [ ] AI menolak menjawab jika confidence < 0.4 (minta klarifikasi).

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
