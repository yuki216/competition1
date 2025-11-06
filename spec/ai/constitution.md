# /speckit.constitution — Project Constitution (Backend Go 1.24)

Tujuan
- Menetapkan prinsip pengembangan, aturan arsitektur, dan quality gates.
- Fokus: DDD + Clean Architecture + TDD untuk backend Go 1.24.

Ringkasan Proyek
- Nama proyek: <isi>
- Deskripsi singkat: <isi>
- Stakeholder: <isi>
- Sasaran bisnis (OKR/KPI): <isi>
- Bounded Context awal: <daftar>
- Integrasi eksternal: <daftar>

Prinsip Inti
- Domain-first: logika bisnis murni di lapisan domain.
- Use case–oriented: aplikasi berpusat pada kasus penggunaan.
- Dependencies ke dalam: kontrak didefinisikan dari sisi domain/application.
- Ports & Adapters: boundary jelas; implementasi di tepi (adapter/infra).
- Pure domain: tanpa I/O, logging, atau context di domain.
- TDD disiplin: red → green → refactor untuk setiap perubahan.

Prinsip SOLID
- Single Responsibility Principle (SRP): setiap modul/class/package memiliki satu alasan untuk berubah. Domain entity fokus pada aturan bisnis; adapter fokus pada I/O/presentasi.
- Open/Closed Principle (OCP): komponen terbuka untuk ekstensi, tertutup untuk modifikasi. Tambahkan adapter baru tanpa mengubah use case/ports; perluas lewat implementasi baru, bukan edit inti.
- Liskov Substitution Principle (LSP): implementasi port harus dapat menggantikan abstraksi tanpa mengubah perilaku klien. Kontrak port jelas; uji menggunakan test yang sama untuk semua implementasi.
- Interface Segregation Principle (ISP): bagi interface menjadi port kecil dan spesifik. Hindari "god interface"; pisahkan `UserRepository`, `RefreshTokenRepository`, `TokenService`, `RecaptchaVerifier` sesuai tanggung jawab.
- Dependency Inversion Principle (DIP): high-level modules (domain/application) tidak bergantung pada low-level (infra); keduanya bergantung pada abstraksi (ports). Detail teknis bergantung pada ports, bukan sebaliknya.

Praktik Implementasi untuk SOLID
- Gunakan injeksi dependensi via konstruktor; hindari global mutable state.
- Konfigurasi diisolasi dalam `Config` dan diteruskan ke layer luar (adapters), bukan domain.
- Error pakai sentinel dan dibungkus di adapter dengan konteks; mapping ke katalog error di interface REST.
- Jaga boundary: mapping DTO ↔ entity dilakukan di adapters; domain tidak mengetahui format transport.

Aturan Lapisan (Clean Architecture ala Uncle Bob)
- Dependency Rule: dependensi selalu mengarah ke dalam (domain pusat). Layer luar tidak boleh mengimport layer dalam.
- Domain: entities, value objects, aggregates, domain services; tidak memanggil I/O; tidak boleh ada struct tag (`json`, `db`, dll); tidak mengimport `context`, `net/http`, `database/sql`, atau framework/library eksternal.
- Application: orkestrasikan use case; bergantung pada domain & port (interface); tidak memanggil infra langsung; boleh menerima `context.Context`; gunakan clock terinjeksi untuk waktu.
- Interface Adapters: implement port (REST, persistence, message); mapping DTO ↔ entity/VO; presentasi response envelope; tidak menaruh logika bisnis.
- Infrastructure: DB (Postgres), HTTP client (reCAPTCHA), JWT provider, web server; detail teknis di tepi; gunakan parameterized queries & handler error yang aman.

Konvensi
- Penamaan package: deskriptif, huruf kecil, satu kata bila mungkin.
- Ports & Interfaces: suffix `Repository`, `Service`, `Verifier`, `Publisher`.
- DTO & Mapper: DTO berada di application/interface; mapping eksplisit; domain tetap bebas dari transport.
- Error: sentinel + `errors.Is/As`; catalog error selaras dengan `specify.md` (Error Catalog); wrap dengan konteks di adapter.
- Waktu: hindari `time.Now()` di domain; injeksikan `Clock` interface.
- Context: hanya di application & adapters; domain tidak menerima `context`.

Boundary & Impor yang Diizinkan
- Domain → (tidak mengimpor siapapun kecuali standard library non-I/O)
- Application → mengimpor Domain + Ports (interface didefinisikan di Application)
- Interface Adapters → mengimpor Application (use case, ports) + presenter/mapper; tidak mengimpor Domain langsung.
- Infrastructure → menyediakan implementasi teknis (DB, JWT, reCAPTCHA) yang dipakai oleh Adapters; tidak mengimpor Domain/Application secara langsung kecuali melalui kontrak (ports).

Ports (Kontrak Aplikasi)
- Definisikan interface untuk eksternal: `UserRepository`, `RefreshTokenRepository`, `TokenService`, `RecaptchaVerifier`, `EventPublisher`.
- Pedoman signature: selalu pakai parameter eksplisit (hindari global); return error terstruktur (sentinel).

Data Flow & DTO
- Request (REST) → DTO (application) → UseCase → Entities/VO → Output DTO → Presenter (envelope) → HTTP response.
- Cookie & header dikelola di adapter; domain tidak mengetahui medium transport.

Testing Policy
- Domain: unit test murni; table-driven; coverage tinggi.
- Application: unit test use case dengan mock ports; verifikasi aturan bisnis & error katalog.
- Adapters: integration test (Postgres, reCAPTCHA HTTP); gunakan database test & migrasi.
- End-to-end: opsional untuk jalur utama login/refresh/logout.

Security & Compliance
- Domain tidak menyimpan plaintext refresh token; adapter persistence menyimpan hash (`sha256(plaintext + SALT)`).
- JWT: kunci/secret dikelola di infra; domain hanya konsumsi kontrak `TokenService`.
- Cookie: diatur HttpOnly, Secure, dan SameSite=Lax di adapter REST.

Transaksi & Idempotensi
- Operasi DB singkat; transaksi terbatas pada boundary repository.
- Idempotensi: `Logout` aman diulang; `Refresh` tidak mengubah state kecuali event/publish.

Observability
- `X-Correlation-ID` dipertahankan dari interface ke adapter; domain tetap ignorant.
- Event `UserLoggedIn/LoggedOut` dipublikasi via port `EventPublisher` (opsional).


Definition of Ready (DoR)
- Use case terdefinisi; acceptance criteria jelas; model data awal siap.
- Kontrak DTO & error katalog tersedia; skenario test terdaftar.

Definition of Done (DoD)
- Unit & integration tests hijau; coverage domain/application ≥ 80%.
- Lint/format lolos; dokumentasi diperbarui; observability baseline ada.

Quality Gates
- `golangci-lint` clean; `go test ./... -race -cover` hijau.
- Benchmark dasar untuk use case kritikal bila relevan.

Non-Functional Requirements (NFR)
- Kinerja, keandalan, keamanan, observability; target numerik: <isi>.

Keputusan Arsitektur (ADR)
- ID, tanggal, keputusan, alternatif, konsekuensi: <isi>

Glosarium
- Istilah domain & definisi: <isi>

Checklist Kepatuhan SOLID & Clean Architecture
- Domain
  - [ ] Tidak ada I/O, logging, atau akses jaringan/file.
  - [ ] Tidak ada struct tag (`json`, `db`, dsb.) dan tidak mengimpor `context`, `net/http`, `database/sql`.
  - [ ] SRP: setiap entity/VO/servis memiliki satu alasan untuk berubah.
  - [ ] Hindari `time.Now()`; gunakan `Clock` interface bila perlu.
  - [ ] Tidak ada DTO/transport concerns; hanya aturan bisnis murni.
- Application
  - [ ] Bergantung pada Domain & Ports (interfaces), bukan adapter/infra.
  - [ ] Use case menerima `context.Context` dan DTO; mapping ke entity/VO dilakukan di adapter.
  - [ ] OCP/DIP: ekstensi via implementasi ports; tidak mengubah use case saat menambah adapter.
  - [ ] LSP: semua implementasi port dapat dipakai bergantian; uji dengan suite yang sama.
- Interface Adapters
  - [ ] REST mempresentasikan envelope `status/message/data` sesuai standar.
  - [ ] Mapping DTO ↔ entity/VO eksplisit; tidak ada logika bisnis di adapter.
  - [ ] Persistence memakai parameterized queries; error dibungkus dengan konteks dan dipetakan ke katalog error.
  - [ ] Cookie `HttpOnly+Secure+SameSite` dan header keamanan ditangani di sini.
- Infrastructure
  - [ ] Koneksi Postgres, migrasi, JWT provider, dan reCAPTCHA client berada di tepi.
  - [ ] Konfigurasi via env; tidak bocor ke domain.
  - [ ] Observability: `X-Correlation-ID` dipertahankan; logging terstruktur.

Template Ports (Contoh Interface)
- UserRepository
```go
// Cari user dan verifikasi password (hashing ada di adapter)
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (User, error)
    VerifyPassword(ctx context.Context, user User, password string) error
}
```
- RefreshTokenRepository
```go
type RefreshTokenRepository interface {
    Create(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time, deviceID string) error
    GetByHash(ctx context.Context, tokenHash []byte) (RefreshToken, error)
    RevokeByHash(ctx context.Context, tokenHash []byte) error
}
```
- TokenService (JWT)
```go
type TokenService interface {
    IssueAccessToken(ctx context.Context, userID string, email string) (token string, expiresAt time.Time, err error)
    ParseAccessToken(ctx context.Context, token string) (Claims, error)
}
```
- RecaptchaVerifier
```go
type RecaptchaVerifier interface {
    Verify(ctx context.Context, token string) (bool, error)
}
```
- EventPublisher (opsional)
```go
type EventPublisher interface {
    Publish(ctx context.Context, eventName string, payload map[string]any) error
}
```

Catatan Implementasi Ports
- Hindari global state; gunakan injeksi dependensi via konstruktor.
- Error sentinel didefinisikan di application/domain; adapter membungkus dan memetakan ke katalog error REST.
- Signature eksplisit; jangan menyelipkan detail teknis ke domain.

Definition of Done (DoD) per Layer
- Domain
  - [ ] Tidak ada I/O, struct tag, atau impor ke paket teknis (`context`, `net/http`, `database/sql`).
  - [ ] Invariant entity/VO terdokumentasi dan diuji (success & failure paths).
  - [ ] Fungsi deterministik; waktu di-abstraksi via `Clock` interface.
  - [ ] Cakupan unit test memadai untuk aturan bisnis inti (≥80% file domain) dan tanpa mocks eksternal.
  - [ ] Tidak ada kebocoran DTO atau format transport.
- Application
  - [ ] Bergantung hanya pada Domain dan Ports; tidak mengimpor adapter/infra.
  - [ ] Use case mendefinisikan input/output DTO jelas; menerima `context.Context`.
  - [ ] Idempotensi dan transaksi didesain sesuai `plan.md`; error memakai katalog terstandar.
  - [ ] Unit test untuk setiap use case (happy path + error mapping); kontrak port diuji dengan test suite yang sama untuk semua implementasi.
  - [ ] Tidak ada kebijakan keamanan/transport di sini (ditangani adapter/infra).
- Interface Adapters (REST/Persistence)
  - [ ] Validasi input, mapping DTO ↔ entity dilakukan eksplisit; tidak ada logika bisnis.
  - [ ] Respons memakai envelope standar (`status/message/data`), error dipetakan dari katalog.
  - [ ] Persistence memakai parameterized queries/prepared statements; indeks sesuai `plan.md`.
  - [ ] Keamanan cookie (`HttpOnly`, `Secure`, `SameSite`), header keamanan, dan `X-Correlation-ID` konsisten.
  - [ ] Test handler (unit) dan integration test untuk adapter DB (dengan Postgres lokal/CI).
- Infrastructure
  - [ ] Migrasi up/down untuk skema; dijalankan sebelum aplikasi start.
  - [ ] Konfigurasi via env (`DATABASE_URL`, `REFRESH_TOKEN_SALT`, dll.); tidak bocor ke domain.
  - [ ] Observability: logging terstruktur, trace ID dipertahankan, metrik dasar tersedia.
  - [ ] Ketahanan: retry/backoff yang wajar, batas koneksi, timeouts.

Terminologi & Mapping Layer ke Struktur Base
- Domain (Clean Architecture) → `domain/` (aggregate, entity, valueobject, domain service, repository interfaces)
- Application (use cases) → `application/` (`usecase/`, `repository/` untuk ports pada level aplikasi)
- Interface Adapters → terbagi menjadi:
  - Presentasi/API → `delivery/` (`http/`, `grpc/` controllers & routing)
  - Data & External Services → `infrastructure/` (`repository/` untuk DB, `service/` untuk layanan eksternal)
- Infrastructure (boundary teknis) → tetap di `infrastructure/` dan tidak diimpor oleh `domain/` atau dipanggil langsung oleh `application/` tanpa melalui ports.

Catatan Konsistensi
- Struktur base di bagian “Standar Struktur Proyek (Base)” adalah rujukan utama. Semua referensi `internal/...` diabaikan dan diganti oleh struktur `domain/`, `application/`, `delivery/`, `infrastructure/`, dll.
- Aturan Clean Architecture & SOLID yang sudah ditetapkan tetap berlaku dengan penyesuaian terminologi folder di atas.
- Aturan import mengikuti versi diselaraskan pada bagian “Aturan Import (Diselaraskan)”.

Checklist Pra-Merge
- [ ] Ketergantungan layer mematuhi aturan (domain tidak mengimpor adapter/infra).
- [ ] DoD tiap layer terpenuhi dan dibuktikan dengan test yang sesuai.
- [ ] Migrasi sinkron dengan skema di `plan.md`; verifikasi up/down di lingkungan dev/CI.
- [ ] Endpoint REST sesuai kontrak di `spec/api.md` dan respons konsisten.
- [ ] Keamanan cookie/header aktif; variabel lingkungan diset dengan default aman.

Standar Struktur Proyek (Base)
- Struktur Folder
```
your-project/
 ├── api/                  # OpenAPI/Swagger specs, JSON schema, proto files 
 ├── build/                # CI/CD configs, Dockerfile, etc. 
 ├── cmd/                  # Application entrypoints 
 │   └── app/              # Main app (cmd name = executable name) 
 ├── docs/                 # Design & documentation 
 ├── domain/               # Domain layer (pure business rules) 
 │   ├── aggregate/        # Aggregates 
 │   ├── enum/             # Enums 
 │   ├── entity/           # Entities 
 │   ├── event/            # Domain events 
 │   ├── valueobject/      # Value objects 
 │   ├── repository/       # Repository interfaces 
 │   └── service/          # Domain services 
 ├── application/          # Application layer (use cases, orchestrating domain logic) 
 │   ├── usecase/          # Use cases (commands, queries) 
 │   └── repository/       # Application-level repository interfaces 
 ├── infrastructure/       # Infrastructure layer 
 │   ├── repository/       # DB implementations 
 │   └── service/          # External service implementations 
 ├── delivery/             # API delivery (HTTP, gRPC handlers, routing) 
 │   ├── http/             # HTTP controllers, middleware 
 │   └── grpc/             # gRPC controllers 
 ├── migrations/           # SQL migrations 
 ├── pkg/                  # Shared/public utilities 
 │   ├── middleware/       # Reusable middlewares 
 │   └── errorcodes/       # Error codes/messages 
 ├── proto/                # Protobuf definitions, buf config, generated files 
 ├── test/                 # Test data and integration tests 
 │   ├── testdata/         # Unit test data 
 │   └── infrastructure/   # Infrastructure tests 
 ├── web/                  # Web assets (if applicable: templates, SPA, static files) 
 └── go.mod 
```
- Catatan Kunci
  - Domain & Application adalah inti; jaga bersih dari framework.
  - Infrastructure mengimplementasikan interface yang didefinisikan di Domain/Application.
  - Delivery bergantung pada Application/usecases, bukan sebaliknya.
  - Error handling terpusat di `pkg/errorcodes/`.
  - Testing: unit test berdampingan dengan kode (`*_test.go`); data uji di `test/testdata/`.

Aturan Import (Diselaraskan)
- `domain/` tidak mengimpor paket teknis/I/O; hanya paket standar non-I/O.
- `application/` hanya mengimpor `domain/` dan kontrak di `application/repository` (ports).
- `infrastructure/` mengimpor `domain/` dan `application/` untuk kontrak; boleh lib eksternal.
- `delivery/` mengimpor `application/usecase` dan `pkg/errorcodes`; tidak mengimpor `domain` langsung.
- `pkg/` berisi util publik; tidak boleh mengimpor `delivery`/`infrastructure`.

Penempatan Kode
- Domain types (entities, value objects, aggregates, domain services) di `domain/`.
- Use cases di `application/usecase/`; ports/kontrak repo di `application/repository/`.
- Implementasi DB di `infrastructure/repository/`; layanan eksternal (JWT, reCAPTCHA) di `infrastructure/service/`.
- HTTP/gRPC handlers, router, middleware di `delivery/{http,grpc}`; middleware reusable di `pkg/middleware`.
- Error catalog dan mapping di `pkg/errorcodes/`.
- Migrasi SQL di `migrations/`; proto di `proto/`.
- Unit test berdampingan; integration test infra di `test/infrastructure/`.

Catatan
- Bagian ini menjadi rujukan utama untuk struktur proyek dan menggantikan rekomendasi struktur `internal/...` sebelumnya; semua aturan Clean Architecture & SOLID tetap berlaku dengan penyesuaian nama folder sesuai standar di atas.