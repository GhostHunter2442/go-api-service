# Setup Guide — go-api-service

สรุปทุกอย่างที่ติดตั้ง/ตั้งค่าในโปรเจกต์นี้ ตั้งแต่เริ่มจนถึงปัจจุบัน พร้อมคำสั่งและคำอธิบายทีละขั้น
อ่านไฟล์นี้ไฟล์เดียวก็เข้าใจภาพรวมและกลับมาทำงานต่อได้

---

## ภาพรวมโปรเจกต์

REST API เขียนด้วย **Go + Gin** ฟังพอร์ต **8080** วางโครงแบบ **layered (cmd/internal)** — handler → service → repository → model
ต่อ **SQL Server** (`easymoney_dev`) ผ่าน **GORM** (`gorm.io/driver/sqlserver`) แบบ read-only ไม่แตะ schema เดิม
มี middleware stack (requestid/logger/recovery/cors/errorhandler), structured logging (slog), response envelope กลาง

**Endpoints:** `GET /healthz` (liveness), `GET /readyz` (ping DB), `GET /api/v1/customers` (list/paginate, `?phone=`), `GET /api/v1/customers/{id}`

**2 workflow ที่ตั้งค่าไว้ (ใช้พอร์ตคนละตัว รันพร้อมกันได้):**

| งาน | เครื่องมือ | คำสั่ง | พอร์ต |
|-----|-----------|--------|-------|
| เขียนโค้ดวนแก้ (dev loop) | **air** | `air` | `:8080` |
| ทดสอบ image ในคอนเทนเนอร์ | **Docker** | ดูหัวข้อ 3 | `:9090` |

---

## ไฟล์ทั้งหมดที่สร้าง/แก้

| ไฟล์ | หน้าที่ |
|------|---------|
| `cmd/api/main.go` | entrypoint — config → logger → ต่อ DB → wire → run + graceful shutdown |
| `internal/` | layered code: `config`, `database`, `model`, `dto`, `repository`, `service`, `handler`, `middleware`, `server` |
| `pkg/` | reusable: `logger` (slog), `apperror` (typed error), `response` (envelope) |
| `.env` / `.env.example` | config (ดูขั้นที่ 1.5) — `.env` ถูก gitignore |
| `Dockerfile` | Multi-stage build → binary เล็ก รันบน distroless (non-root, amd64) |
| `.dockerignore` | กันไฟล์ที่ไม่เกี่ยวออกจาก Docker image |
| `.air.toml` | config ของ air (live reload ตอน dev — build `./cmd/api`) |
| `docs/` | **gen อัตโนมัติ** ด้วย `swag init -g cmd/api/main.go` — commit ลง git |
| `.gitignore` | ignore `bin/`, `tmp/`, `.env`, `*.exe` |
| `docker.md` | คู่มือ Docker แบบละเอียด |
| `setup.md` | ไฟล์นี้ |

---

## ขั้นที่ 0 — ของที่ต้องมีในเครื่อง (ติดตั้งครั้งเดียว)

| เครื่องมือ | ใช้ทำอะไร | ตรวจสอบ |
|-----------|----------|---------|
| **Go 1.26** | คอมไพล์/รันโค้ด | `go version` |
| **SQL Server** | database (`easymoney_dev`) | ต่อได้ผ่าน `.env` (ขั้นที่ 1.5) |
| **Docker Desktop** | build & run container | `docker version` |
| **air** | live reload (ติดตั้งในขั้นที่ 1) | `air -v` |
| **swag** | gen API docs จาก annotation (ขั้นที่ 2.5) | `swag --version` |

---

## ขั้นที่ 1 — Dev loop ด้วย air (live reload)

ใช้ตอนเขียนโค้ดปกติ แก้แล้วเซฟ → rebuild + restart อัตโนมัติ เร็วสุด

**ติดตั้ง air (ครั้งเดียว):**
```powershell
go install github.com/air-verse/air@latest
```
> binary ลงที่ `C:\Users\Hunter\go\bin\air.exe` — โฟลเดอร์นี้ต้องอยู่ใน PATH
> (ถ้า `air -v` หาไม่เจอ ให้เพิ่ม `%USERPROFILE%\go\bin` เข้า PATH)

**รัน dev:**
```powershell
air
```
แก้ไฟล์ `.go` ไหนแล้วเซฟ → air build เป็น `tmp/main.exe` แล้ว restart server ที่ `:8080` ให้ทันที
กด `Ctrl+C` เพื่อหยุด

**ทดสอบ:**
```powershell
curl http://localhost:8080/hello
```

> config อยู่ใน `.air.toml` (เฝ้าเฉพาะ `.go`, ข้าม `tmp/ bin/ node_modules/ .git/ .github/`)

---

## ขั้นที่ 1.5 — ต่อ SQL Server (`.env`)

service อ่าน config จาก env (มี default สำหรับ dev) ตอน dev ใส่ค่าไว้ในไฟล์ `.env` ที่ root
แล้วโค้ด auto-load ให้ผ่าน `godotenv` (production ไม่มี `.env` ก็ข้าม ใช้ env จริงของ host/container)

```powershell
copy .env.example .env       # แล้วแก้ค่า DB_* ให้ตรงเครื่อง
```

ค่าใน `.env` (ดูเทมเพลตเต็มใน `.env.example`):
```
DB_HOST=192.168.251.120
DB_PORT=1433
DB_USER=apidej
DB_PASSWORD=********
DB_NAME=easymoney_dev
DB_ENCRYPT=disable           # dev มักใช้ disable
```

> ⚠️ `.env` ถูก gitignore (มีรหัสผ่าน) — **ห้าม commit** ส่วน `.env.example` (ไม่มีรหัสจริง) commit ได้
> การต่อ DB ทำที่ `internal/database/database.go → NewSQLServer()` — ตั้ง pool + ping ตรวจตอน startup (fail fast)
> ตรวจว่าต่อติดจริง: `curl http://localhost:8080/readyz` → `{"status":"ready"}`

---

## ขั้นที่ 2 — ตรวจว่าโค้ด Go ปกติ

```powershell
go build ./...            # คอมไพล์ทั้ง module ผ่านไหม (entrypoint อยู่ที่ ./cmd/api)
go vet ./...              # static analysis
go fmt ./...              # จัดรูปแบบโค้ด
go test ./...             # รัน test (ยังไม่มี test ตอนนี้)
```

---

## ขั้นที่ 2.5 — API docs ด้วย Swagger (swaggo)

ใช้ **swaggo/swag** — เขียน annotation (comment พิเศษ) เหนือ handler → gen เป็น Swagger UI
หลักการ: `comment ในโค้ด → swag init → docs/ → Swagger UI ที่ /swagger/`

### วิธีติดตั้งจากศูนย์ (ทำครั้งเดียว)

**1) ติดตั้ง swag CLI** (ตัว gen docs)
```powershell
go install github.com/swaggo/swag/cmd/swag@latest
swag --version          # ยืนยัน เช่น v1.16.4
```
> binary ลงที่ `C:\Users\Hunter\go\bin\swag.exe` — ต้องมีโฟลเดอร์นี้ใน PATH (ที่เดียวกับ air)

**2) เพิ่ม dependency runtime** (ตัวเสิร์ฟ UI + spec)
```powershell
go get github.com/swaggo/http-swagger/v2@latest
go get github.com/swaggo/swag@v1.16.4    # ⚠️ ต้อง "ตรงเวอร์ชัน" กับ CLI ข้อ 1
```
> **ปัญหาที่เจอจริง:** ถ้าปล่อยให้ `go mod tidy` เลือกเอง มันดึง `swag` เวอร์ชันเก่า (v1.8.1)
> แล้ว build พังที่ `docs.go: unknown field LeftDelim` — แก้โดย pin ให้ตรง CLI ตามคำสั่งบน

> ⚠️ **entrypoint ย้ายไป `cmd/api/main.go` แล้ว** — general-info annotation (`@title` ฯลฯ) อยู่ที่นั่น
> ดังนั้น `swag init` ต้องชี้ด้วย `-g cmd/api/main.go` เสมอ (ไม่งั้น swag หา main ไม่เจอ → docs ว่าง)

### วิธีเขียน annotation ใน `cmd/api/main.go`

**General info** — วางเหนือ `func main()`:
```go
// @title        go-api-service API
// @version      1.0
// @description  Hello World API เขียนด้วย Go net/http
// @host         localhost:8080
// @BasePath     /
func main() { ... }
```

**ต่อ endpoint** — วางเหนือ handler แต่ละตัว:
```go
// @Summary      Show greeting
// @Description  Return a greeting message
// @Tags         greeting
// @Produce      json
// @Success      200  {object}  response
// @Router       /hello [get]
func helloHandler(w http.ResponseWriter, r *http.Request) { ... }
```

### เชื่อม Swagger UI เข้า server

เพิ่ม import + route ใน `main.go`:
```go
import (
    _ "github.com/apidet/go-api-service/docs"          // ← docs ที่ gen (blank import)
    httpSwagger "github.com/swaggo/http-swagger/v2"
)

// ใน main():
mux.Handle("GET /swagger/", httpSwagger.Handler(
    httpSwagger.URL("/swagger/doc.json"),
))
```

### gen docs แล้วใช้งาน

```powershell
swag init -g cmd/api/main.go   # อ่าน annotation → สร้าง docs/ (docs.go, swagger.json, swagger.yaml)
air                            # รัน server
```
เปิดเบราว์เซอร์:
```
http://localhost:8080/swagger/index.html
```

### กฎสำคัญ (จำให้ขึ้นใจ)

- ⚠️ **แก้ annotation แล้วต้อง `swag init -g cmd/api/main.go` ใหม่เสมอ** — air rebuild ให้ก็จริง แต่ไม่ gen docs ให้
  (ถ้าอยากให้ auto gen ทุก save: ใส่ `pre_cmd = ["swag init -g cmd/api/main.go"]` ใน `[build]` ของ `.air.toml`
  — แลกกับ reload ช้าขึ้น ~4 วิ)
- ⚠️ ถ้ารัน `gofmt` แล้ว indent ของ comment เปลี่ยน → **`swag init` ซ้ำอีกที** (swag ยังอ่านได้ แต่ gen ใหม่กันพลาด)
- ✅ **commit `docs/` ลง git เสมอ** — เพราะ `main.go` import `docs` package
  ถ้าไม่ commit → Docker / CI build จะพัง (หา package ไม่เจอ)

### workflow ทุกครั้งที่เพิ่ม/แก้ endpoint

```powershell
# 1) แก้โค้ด + annotation ใน handler / cmd/api/main.go
# 2) gen docs ใหม่   ← ห้ามลืม! (air ไม่ gen ให้)
swag init -g cmd/api/main.go
# 3) ให้ server ใช้ docs ใหม่ — air จะ rebuild เองเมื่อ docs/docs.go เปลี่ยน
#    (ถ้า air ค้าง ให้ Ctrl+C แล้วรัน air ใหม่)
# 4) hard refresh หน้า Swagger กัน browser cache
#    Ctrl + F5  ที่ http://localhost:8080/swagger/index.html
```

### ตรวจว่าใช้งานได้ (verify)
```powershell
curl http://localhost:8080/swagger/doc.json     # ควรได้ JSON ที่มี title + paths
curl -o NUL -w "%{http_code}" http://localhost:8080/swagger/index.html   # ควรได้ 200
```

### Troubleshooting — `swag init` แล้ว Swagger ยังไม่อัปเดต

ไล่เช็ก 3 จุดนี้ตามลำดับ:

```powershell
# 1) docs ที่ gen มี endpoint ใหม่ไหม (เช็กไฟล์)
#    ถ้าไม่มี = annotation ผิด/ลืม swag init -> gen ใหม่
type docs\swagger.json | findstr "/your-path"

# 2) server เสิร์ฟ spec ใหม่ไหม
curl http://localhost:8080/swagger/doc.json

# 3) route ใหม่ตอบไหม — ถ้าได้ 404 = server กำลังรัน "binary เก่า"
curl -o NUL -w "%{http_code}" http://localhost:8080/your-path
```

**สาเหตุที่เจอบ่อยสุด:** route ใหม่ตอบ **404** แปลว่ามี **process เก่าค้างยึดพอร์ต 8080**
ทำให้ air build ใหม่เสร็จแต่ bind ไม่ได้ เลยรันตัวเก่าต่อ — หา process แล้วฆ่าทิ้ง:

```powershell
# หาว่าใครยึด 8080
Get-NetTCPConnection -LocalPort 8080 -State Listen | ForEach-Object {
  Get-Process -Id $_.OwningProcess | Select-Object Id, ProcessName, Path
}
# ฆ่าทิ้งด้วย PID ที่ได้
Stop-Process -Id <PID> -Force
```
แล้วสะกิด air ให้ start ใหม่ (เซฟไฟล์ หรือ Ctrl+C → `air`)

> มักเป็น `tmp\main.exe` (ตัว air เอง), `go run` เก่าที่ลืมปิด, หรือ binary ที่เคยรันทดสอบไว้

---

## ขั้นที่ 2.7 — โครง production (Gin + layered) — ทำอะไรไปบ้าง

ย้ายจาก `net/http` → **Gin** และวางเป็น layered architecture ระดับ production

**Dependency ที่ลงเพิ่ม** (`go get` — บันทึกใน `go.mod`):
```powershell
go get github.com/gin-gonic/gin          # web framework
go get github.com/swaggo/gin-swagger github.com/swaggo/files   # Swagger UI สำหรับ Gin
go get github.com/google/uuid            # gen request id
# (มีอยู่แล้วจากก่อนหน้า) gorm.io/gorm, gorm.io/driver/sqlserver, github.com/joho/godotenv
```
> `log/slog` (structured logging) เป็น std lib — ไม่ต้องลง

**ไฟล์/แพ็กเกจที่เพิ่ม:**

| แพ็กเกจ | หน้าที่ |
|---------|---------|
| `internal/server/` | `server.go` (สร้าง Gin engine + middleware stack) + `router.go` (ผูก route, group `/api/v1`) |
| `internal/middleware/` | `requestid` · `logger` (slog) · `recovery` · `cors` · `errorhandler` |
| `internal/dto/` | struct สำหรับ bind request + shape response (แยก API ออกจาก model/DB) |
| `pkg/logger/` | ตั้งค่า slog (json/text ตาม `LOG_FORMAT`) |
| `pkg/apperror/` | typed error (`NotFound`/`BadRequest`/...) → map เป็น HTTP status |
| `pkg/response/` | envelope กลาง `{ success, data, error, meta }` |

**request ไหลยังไง (1 request วิ่งผ่านอะไรบ้าง):**
```
HTTP → Gin engine
     → middleware: Recovery → RequestID → Logger → CORS → ErrorHandler
     → handler (bind/validate query+param ด้วย dto)
     → service (business logic, clamp limit)
     → repository (GORM query SQL Server; not-found → apperror.NotFound)
     → model (map ตาราง customers)
     ← handler ตอบด้วย response.Success / response.Paginated
     ← ถ้า error: handler แค่ c.Error(err) → ErrorHandler แปลงเป็น envelope กลาง
```

**กฎที่ยึด (อย่าทำผิด):**
- handler **ไม่เขียน JSON error เอง** → `c.Error(apperror.NotFound(...))` แล้ว `return` พอ
- response ทุกอันผ่าน `pkg/response` (ห้ามปั้น `gin.H` รูปแบบมั่ว)
- DTO แยกจาก model — list ใช้ DTO ย่อ, detail ใช้ full model (`json:"-"` ซ่อน password/token)
- log ใช้ `slog` ที่ inject เข้ามา — ไม่ใช้ `gin.Default()` / `fmt.Println`

**ทดสอบ (รูป response ใหม่):**
```powershell
curl "http://localhost:8080/api/v1/customers?limit=2"   # { "success":true, "data":[...], "meta":{...} }
curl http://localhost:8080/api/v1/customers/999999       # { "success":false, "error":{ "code":"NOT_FOUND", ... } }
```

---

## ขั้นที่ 2.8 — CLI tools (gen ไฟล์ / scaffold)

> **สำคัญ:** Go **ไม่มี** scaffold generator แบบ `rails generate` / `php artisan make`
> การเพิ่ม resource ใหม่ = สร้างไฟล์ตาม layer เอง (ดู checklist ท้ายหัวข้อ)
> แต่มี CLI เฉพาะทางที่ "gen ไฟล์/โค้ด" ให้ — ลงเฉพาะที่ใช้จริง

| CLI | ใช้ทำอะไร | ลง (ครั้งเดียว) | สั่งใช้ |
|-----|-----------|----------------|---------|
| **air** | live reload ตอน dev | `go install github.com/air-verse/air@latest` | `air` |
| **swag** | gen Swagger docs จาก annotation | `go install github.com/swaggo/swag/cmd/swag@latest` | `swag init -g cmd/api/main.go` |
| **mockgen** | gen mock จาก interface (ไว้เขียน unit test) | `go install go.uber.org/mock/mockgen@latest` | `mockgen -source=internal/repository/customer_repository.go -destination=internal/repository/mocks/customer_repository.go -package=mocks` |
| **migrate** | gen/รัน DB migration (ถ้าจะคุม schema) | `go install -tags 'sqlserver' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` | `migrate create -ext sql -dir migrations <name>` |
| **golangci-lint** | linter รวม (vet+staticcheck+ฯลฯ) | ดู [คู่มือ install](https://golangci-lint.run/welcome/install/) | `golangci-lint run` |

> binary ทั้งหมดลงที่ `C:\Users\Hunter\go\bin\` — โฟลเดอร์นี้ต้องอยู่ใน PATH (ที่เดียวกับ air/swag)
> ✅ ที่จำเป็นต่อ workflow ตอนนี้: **air + swag** (ลงแล้ว) · **mockgen** (เมื่อเริ่มเขียน test)
> ⏳ migrate/golangci-lint = ลงเมื่อต้องใช้

**Checklist เพิ่ม resource ใหม่ (เช่น `orders`) — สร้างไฟล์ตามนี้:**
```
1. internal/model/order.go              # GORM entity (column tag, NULL→pointer, json:"-" ฟิลด์อ่อนไหว)
2. internal/repository/order_repository.go   # interface + GORM impl
3. internal/service/order_service.go    # business logic
4. internal/dto/order_dto.go            # bind request + response DTO
5. internal/handler/order_handler.go    # Gin handler (c.Error + response.*)
6. internal/server/router.go            # เพิ่ม group/route ใน /api/v1
7. cmd/api/main.go                      # wire repo→service→handler เข้า server.Handlers
8. swag init -g cmd/api/main.go         # regen docs
```

---

## ขั้นที่ 3 — ทดสอบด้วย Docker (รันในคอนเทนเนอร์)

> รายละเอียดเต็มอยู่ใน `docker.md` — สรุปคำสั่งหลัก:

```powershell
# 1) build image
docker build -t go-api-service:local .

# 2) ดูว่า image ถูกสร้าง (ควรได้ ~8MB)
docker images go-api-service

# 3) รัน container (host 9090 -> container 8080)
docker run -d --name go-api-local -p 9090:8080 go-api-service:local

# 4) ทดสอบ
curl http://localhost:9090/hello
```

**คำสั่งจัดการ:**
```powershell
docker logs go-api-local        # ดู log
docker stop go-api-local        # หยุด
docker start go-api-local       # เปิดต่อ
docker rm -f go-api-local       # ลบทิ้ง
```

**หลังแก้โค้ดต้อง build image ใหม่เสมอ:**
```powershell
docker rm -f go-api-local
docker build -t go-api-service:local .
docker run -d --name go-api-local -p 9090:8080 go-api-service:local
```

> ใช้พอร์ต 9090 เพราะ 8080 ถูก air ใช้อยู่ — รัน air กับ Docker พร้อมกันได้

---

## Cheat sheet — คำสั่งที่ใช้บ่อยสุด

```powershell
air                                  # dev + live reload (:8080)
swag init -g cmd/api/main.go         # gen API docs ใหม่ (หลังแก้ annotation)
docker build -t go-api-service:local .   # build image
docker run -d --name go-api-local -p 9090:8080 go-api-service:local  # run (:9090)
```
