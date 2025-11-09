# Release Notes — Auth Feature

## Versi
- x.y.z

## Perubahan Utama
- <ringkas perubahan dan fitur yang ditambahkan>

## Migrasi & Kompatibilitas
- Migrasi `refresh_tokens` diperlukan sebelum start aplikasi.
- Kompatibilitas backward: <jelaskan bila ada perubahan kontrak API atau skema yang tidak kompatibel>.

## Catatan Implementasi
- Env diselaraskan: `DATABASE_URL`, `JWT_ALG`, `JWT_PRIVATE_KEY`/`JWT_PUBLIC_KEY` atau `JWT_SECRET`, `REFRESH_TOKEN_SALT`, `RECAPTCHA_*`, `PORT`.
- Observability: `X-Correlation-ID` aktif di semua request/response.

## Known Issues
- <daftar masalah yang diketahui dan mitigasi sementara>

## Tanggal Rilis
- <YYYY-MM-DD>