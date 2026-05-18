# Product Requirements Document (PRD)
**Project:** Multi-Tenant WhatsApp Message Gateway (Golang)
**Date:** May 2026

## 1. Executive Summary

- **Problem Statement:** Dua belas (12) perusahaan yang berbeda membutuhkan platform *Customer Service* omnichannel yang cepat, stabil, dan bisa menangani skenario *blast* otentikasi/marketing. Menggunakan WABA (WhatsApp Business Account) tunggal untuk banyak perusahaan memunculkan risiko *rate-limit exhaustion*, kebocoran data antar *tenant*, dan kesulitan melacak biaya *template* per perusahaan.
- **Proposed Solution:** Membangun *Message Gateway* tersentralisasi menggunakan standar *Golang Pro* (Go 1.21+) dengan arsitektur *microservices* (API, Webhook, Worker Pool). Sistem memanfaatkan Message Broker (RabbitMQ/Redis) untuk antrean asinkron dan PostgreSQL untuk isolasi data *tenant* serta pencatatan tagihan (*billing*).
- **Success Criteria:**
  - Latensi Inbound: Pesan masuk harus tampil di layar Agen CS dalam waktu < 1000ms.
  - Throughput: Mampu menangani hingga 1.800.000 pesan *outbound* (berdasarkan target 36 aplikasi x 50.000 user) tanpa *memory leak* atau *crash*.
  - Billing Accuracy: Laporan tagihan *template* memiliki akurasi 100% tanpa *race condition*, terintegrasi *real-time* di *dashboard*.

## 2. User Experience & Functionality

- **User Personas:**
  - **CS Agent:** Staf perusahaan yang membalas pesan secara *real-time* (Estimasi: 36 agen aktif bersamaan).
  - **Company Admin:** Manajer dari 12 perusahaan yang memantau agen, membuat *template*, dan memeriksa tagihan biaya pesan.
  - **Superadmin:** Pengelola infrastruktur utama dan pemilik WABA.
- **User Stories:**
  - `Sebagai [CS Agent], saya ingin [menerima dan membalas pesan tanpa me-refresh halaman] sehingga [bisa merespons pelanggan di bawah 1 detik].`
  - `Sebagai [Company Admin], saya ingin [melihat tagihan biaya template secara real-time di dashboard] sehingga [saya bisa memantau pengeluaran perusahaan saya secara transparan].`
  - `Sebagai [Company Admin], saya ingin [membuat dan mendaftarkan template pesan baru] sehingga [otomatis perusahaan lain bisa memakai template tersebut].`
  - `Sebagai [Superadmin], saya ingin [mendapat peringatan jika penggunaan kuota WABA mencapai 90%] sehingga [bisa melakukan antisipasi sebelum nomor diblokir Meta].`
- **Acceptance Criteria:**
  - Sistem menegakkan *24-Hour Window* Meta. CS tidak bisa membalas dengan teks bebas jika pesan terakhir pelanggan melebihi batas 24 jam; UI memaksa penggunaan *template*.
  - *Billing* untuk *template message* bertambah otomatis per `company_id` dan disajikan secara *live* pada *dashboard Company Admin*.
  - Seluruh *template* yang terdaftar bersifat *shared* dan dapat diakses melalui *dropdown* oleh seluruh perusahaan dalam ekosistem.
- **Non-Goals:**
  - Sistem AI Chatbot atau auto-reply (*scope* saat ini murni untuk CS Manusia dan API Blast).
  - Registrasi WABA mandiri per *tenant* (*hosting* sepenuhnya berada di bawah 1 WABA pusat).

## 3. AI System Requirements
- *N/A untuk fase ini. Arsitektur disiapkan untuk forwarder webhook ke layanan LLM/AI di masa mendatang.*

## 4. Technical Specifications

- **Architecture Overview:**
  - **Inbound Service:** HTTP server (`net/http`) yang memvalidasi *webhook* Meta, menaruh *payload* di RabbitMQ, dan langsung me-*return* `200 OK`.
  - **Worker Pool:** Kumpulan Goroutine dengan *Context cancellation* yang bertugas mengkonsumsi antrean. Dilengkapi modul **Dynamic Rate Limiter** (menggunakan *Token Bucket*).
  - **Real-Time Engine:** WebSocket server terhubung dengan Redis Pub/Sub untuk mendistribusikan notifikasi *chat* masuk langsung ke layar agen.
- **Integration Points:**
  - **Meta Graph API (v20.0+):** `POST /{phone_number_id}/messages` (Pengiriman) dan manajemen *template*.
  - **Database:** PostgreSQL 15+ untuk skema data relasional, mengisolasi `company_id`.
  - **Queue/Cache:** RabbitMQ untuk antrean pesan; Redis untuk status agen dan parameter *rate limit* dinamis.
- **Security & Privacy:**
  - **Tenant Data Isolation:** Setiap eksekusi kueri pada kontak dan pesan harus menggunakan filter `company_id`.
  - **Idempotency & Concurrency:** Transaksi *database* (terutama untuk pencatatan *billing*) diamankan dengan fitur *locking* (`sync.Mutex` di aplikasi, atau *Row-Level Lock* di PostgreSQL) guna mencegah *double-billing*.

## 5. Risks & Roadmap

- **Phased Rollout:**
  - **MVP:** Fondasi Inbound Webhook, Outbound API, Database Schema, dan integrasi UI WebSocket untuk operasional CS 24 Jam dasar.
  - **v1.1:** Manajemen *Template* (Shared Library), sistem *Live Billing*, dan notifikasi Superadmin.
  - **v2.0:** Skalabilitas tinggi (*Load Testing* untuk 1.8M antrean) dan operasional sistem *Marketing/Auth Blast*.
- **Technical Risks & Mitigation:**
  - **Risk 1: WABA Tier Limit Exhaustion (CRITICAL).** Pengiriman massal bisa membentur batas (misal: tier 2.5K/10K).
    - *Mitigation:* Implementasi *Dynamic Throttler* di level *worker* antrean Golang. Kapasitas *Throttler* dikonfigurasi melalui Redis/DB agar fleksibel seiring bertambahnya limit dari Meta. Mengirim peringatan (*alert*) ke Superadmin pada penggunaan 90%.
  - **Risk 2: Webhook Latency & Timeout.** Meta memutus koneksi jika respons lebih dari batas wajar.
    - *Mitigation:* Proses IO-berat (DB Insert/Update) dilarang pada *layer handler*. *Handler* hanya memvalidasi dan mem- *publish* data ke *Message Broker*.