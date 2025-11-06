# /speckit.constitution — Fitur Constitution (Backend Go 1.24)

Tujuan
- Menetapkan prinsip pengembangan, aturan arsitektur, dan quality gates.
- Fokus: DDD + Clean Architecture + TDD untuk backend Go 1.24.

Ringkasan Fitur
- Nama fitur: User Management
- Deskripsi singkat: Admin bisa menambahkan user baru

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