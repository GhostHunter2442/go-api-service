# CLAUDE.md

แนวทางสำหรับ Claude Code เมื่อทำงานกับ repo นี้

## ภาพรวมโปรเจกต์

`go-api-service` — REST API service เขียนด้วย Go ใช้ **Gin** (web) + **GORM** (ORM)
วางโครงแบบ **layered (clean) architecture**

- **Module:** `github.com/apidet/go-api-service`
- **Go version:** 1.26 (ดู directive ใน `go.mod`)
- **Remote:** https://github.com/GhostHunter2442/go-api-service (branch หลัก: `main`)
- **Web framework:** Gin (`github.com/gin-gonic/gin`)
- **Database:** SQL Server (`easymoney_dev`) ผ่าน GORM + `gorm.io/driver/sqlserver` — **read-only ไม่แตะ schema เดิม (ไม่มี AutoMigrate)**
- **Logging:** `log/slog` (structured JSON/text)
- **API docs:** swaggo + gin-swagger (Swagger UI ที่ `/swagger/index.html`)
- รายละเอียดติดตั้ง/ตั้งค่า/ต่อ DB/`.env` → ดู **`setup.md`** (ที่เดียวจบ)

## โครงสร้าง (layered: handler → service → repository → model)

```
go-api-service/
├── cmd/api/main.go              # entrypoint: config → logger → ต่อ DB → wire → run + graceful shutdown
├── internal/
│   ├── config/config.go         # อ่าน env → Config (+ auto-load .env ผ่าน godotenv) + สร้าง DSN
│   ├── database/database.go     # NewSQLServer() — เปิด GORM + pool + ping (จุดเดียวที่เปิด connection)
│   ├── model/customer.go        # GORM entity (map ตาราง dbo.customers)
│   ├── dto/                     # request bind + response shape (แยก API contract จาก model)
│   ├── repository/              # data access (interface + GORM impl) — mock/สลับ impl ได้
│   ├── service/                 # business logic — ไม่รู้จัก HTTP, ไม่ผูก Gin
│   ├── handler/                 # transport (Gin): customer, health
│   ├── middleware/              # requestid, logger, recovery, cors, errorhandler
│   └── server/                  # server.go (engine+stack) + router.go (route registration)
├── pkg/
│   ├── logger/                  # slog setup
│   ├── apperror/                # typed error → HTTP status
│   └── response/                # envelope กลาง { success, data, error, meta }
├── docs/                        # gen ด้วย swag (commit ลง git — server import docs)
├── .env / .env.example          # config (.env ถูก gitignore — มีรหัสผ่าน)
├── .air.toml                    # live reload (build ./cmd/api)
└── go.mod
```

**Endpoints:**
- `GET /healthz` — liveness (ไม่แตะ DB) · `GET /readyz` — readiness (ping SQL Server)
- `GET /api/v1/customers?limit=&offset=` — list/paginate (default 50, max 200); `?phone=` ค้นรายเดียว (unique)
- `GET /api/v1/customers/{id}` — รายเดียวตาม `customer_id`
- `GET /swagger/index.html` — Swagger UI

**Response envelope (ทุก endpoint):**
```
สำเร็จ : { "success": true,  "data": ..., "meta": {limit,offset,count} }
ล้มเหลว: { "success": false, "error": { code, message, request_id } }
```

## คำสั่งที่ใช้บ่อย

```powershell
air                              # dev + live reload (:8080) — config ใน .air.toml
go run ./cmd/api                 # รัน dev mode ตรงๆ (ไม่ผ่าน air)
go build ./...                   # คอมไพล์ทั้ง module
go vet ./...                     # static analysis
go fmt ./...                     # format
go test ./...                    # รัน test ทั้งหมด
go mod tidy                      # sync dependency
swag init -g cmd/api/main.go     # gen API docs ใหม่ (หลังแก้ annotation — ดู setup.md)
```

**ทดสอบ:**
```powershell
curl http://localhost:8080/readyz                       # ต่อ DB ติด
curl "http://localhost:8080/api/v1/customers?limit=2"
```

## ข้อตกลง / Conventions

- **routing ผ่าน Gin** ใน `internal/server/router.go` — group เวอร์ชันที่ `/api/v1`; path param ด้วย `c.Param("id")`, query ด้วย `c.ShouldBindQuery(&dto)`
- **error handling รวมที่เดียว**: handler แค่ `c.Error(err)` แล้ว `return` (ไม่เขียน JSON error เอง)
  — `middleware.ErrorHandler` แปลง `*apperror.AppError` → status/code/message; error อื่น → 500
  สร้าง error ด้วย `apperror.NotFound(...)`, `apperror.BadRequest(...)` ฯลฯ
- **response ผ่าน `pkg/response`**: `response.Success(data)`, `response.Paginated(data, meta)` — อย่าเขียน gin.H เองให้รูปแบบเพี้ยน
- ทิศ dependency: **handler → service → repository → model** ชั้นในไม่รู้จักชั้นนอก;
  repository เป็น **interface** เพื่อ mock ใน test และสลับ impl (GORM → sqlc/sqlx) ได้
- **DTO แยกจาก model**: response/​request ใช้ struct ใน `internal/dto` ไม่ผูก API เข้ากับ schema ตรงๆ
- **logging ใช้ slog** (`log/slog`) ผ่าน logger ที่ inject — ไม่ใช้ `fmt.Println`/`gin.Default()`
- **config อ่านที่ startup เท่านั้น** (`config.Load()` ใน main) — layer อื่นรับเป็น argument ไม่อ่าน env เอง
- **เปิด DB ที่เดียว** ที่ `database.NewSQLServer()` — layer อื่นรับ `*gorm.DB` ไปใช้
- **ห้าม AutoMigrate กับ DB จริง** — ใช้ schema ที่มีอยู่ตามเดิม (migration จริงใช้ tool แยก เช่น golang-migrate)
- GORM model map ตาราง: คอลัมน์ `NULL` → ใช้ **pointer** (`*string`/`*int`/`*time.Time`) กัน error ตอน scan NULL;
  ใส่ **`gorm:"column:..."` ทุก field** (กัน GORM แปลงชื่อย่อเพี้ยน เช่น `IDCard`, `OTP...`);
  field อ่อนไหว (`password`, `access_token`) ใส่ **`json:"-"`** ไม่ให้หลุดออก response
- JSON response: ตั้ง `Content-Type: application/json` เสมอ, field tag เป็น `snake_case`
- โค้ดใน `internal/` import จากนอก module ไม่ได้ — เหมาะกับ encapsulation

## หมายเหตุ Environment (Windows)

- Shell หลักคือ **PowerShell** — เลี่ยง here-string ที่มีอักขระพิเศษ (`{}`, `!`, `:`) กับ
  `git commit -m`; ใช้ `git commit -F <file>` แทนเพื่อกัน quoting พัง
- git ตั้ง `credential.helper = manager` (GCM) ไว้แล้ว push/pull ไม่ต้อง login ซ้ำ
- การรัน git ผ่าน PowerShell อาจโชว์ stderr เป็น `RemoteException` ทั้งที่สำเร็จ
  — ดูบรรทัดผลลัพธ์จริง (เช่น `xxx..yyy main -> main`) เป็นหลัก

## งานที่ยังทำต่อได้ (backlog ที่คุยกันไว้)

- [x] แยกโครงสร้างเป็น `cmd/internal` (layered)
- [x] เพิ่ม graceful shutdown (`signal.NotifyContext` + `server.Shutdown`)
- [x] ต่อ SQL Server ผ่าน GORM (read-only) + health/readiness probe
- [x] ย้ายไป **Gin** + middleware stack (requestid/logger/recovery/cors/errorhandler) + `/api/v1`
- [x] structured logging (slog), typed error (apperror), response envelope
- [ ] เพิ่ม unit test ให้ service/handler (mock repository, `httptest`)
- [ ] auth middleware (JWT) + rate limit เมื่อมี endpoint ที่ต้องป้องกัน
- [ ] migration tool แยก (เช่น golang-migrate) ถ้าต้องจัดการ schema
