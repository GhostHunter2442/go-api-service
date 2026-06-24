# Docker — คู่มือ build & run บนเครื่อง (local)

คู่มือนี้สำหรับ build & run `go-api-service` ด้วย Docker บนเครื่องตัวเอง
อ่านครั้งเดียวจบ ใช้ซ้ำได้ทุกครั้งที่กลับมาทำงาน

> สรุปสถาปัตยกรรม: Go (`net/http`) ฟังพอร์ต **8080** ในคอนเทนเนอร์เสมอ
> เราแมปออกมาที่พอร์ตเครื่องผ่าน `ports: "<host>:8080"`

---

## 0. เตรียมก่อนเริ่ม (ตรวจครั้งเดียว)

ต้องมี **Docker Desktop** เปิดทำงานอยู่ (ไอคอนวาฬมุมขวาล่างต้องเป็นสีปกติ ไม่ใช่ "starting")

ตรวจว่า Docker พร้อม:
```powershell
docker version          # เห็นทั้ง Client: และ Server: = daemon ทำงานแล้ว
docker compose version  # v2 (มากับ Docker Desktop) — เรียก `docker compose` ได้
```

---

## แนวคิด: Dockerfile vs docker-compose.yml

มันคนละหน้าที่ ใช้คู่กัน **ไม่ใช่แทนกัน**

| | **Dockerfile** | **docker-compose.yml** |
|---|---|---|
| ตอบว่า | image นี้ **สร้างยังไง** (compile/package) | container **รันยังไง** (ชื่อ/พอร์ต/env) |
| แทนอะไร | ขั้นตอน `docker build` | คำสั่ง `docker run ...` ยาวๆ |

มี **2 แบบ** ในการจัดวาง (ด้านล่างเป็นตัวอย่างทั้งคู่) — **โปรเจกต์นี้ใช้แบบ A**

---

## แบบ A (ที่โปรเจกต์นี้ใช้): แยกไฟล์ `Dockerfile` + `docker-compose.yml`

แบบมาตรฐานทั่วไป มี 2 ไฟล์ ได้ syntax highlight + lint (hadolint) ของ Dockerfile
แก้ขั้นตอน build แยกจาก config การรัน container ชัดเจน

**ไฟล์ `Dockerfile` (จริงในโปรเจกต์):**
```dockerfile
# ---- build stage ----
FROM golang:1.26 AS build
WORKDIR /src

# ก๊อป go.mod/go.sum ก่อน → cache layer ของ dependency (เร็วเวลา build ซ้ำ)
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# build แบบ static (CGO ปิด) สำหรับ linux/amd64 — entrypoint อยู่ที่ ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app ./cmd/api/...

# ---- runtime stage ----
# distroless = image เล็ก (~8MB), ไม่มี shell, รันด้วย non-root
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app /app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

**ไฟล์ `docker-compose.yml` (จริงในโปรเจกต์):** (`build: .` ชี้ไปหา `Dockerfile` ข้างบน)
```yaml
services:
  api:
    build: .                      # = docker build .  (เรียก Dockerfile ในโฟลเดอร์นี้)
    image: go-api-service:local   # ตั้งชื่อ:แท็ก ให้ image ที่ build ได้
    container_name: go-api-local  # = --name go-api-local
    env_file: .env                # = --env-file .env  (compose อ่านให้อัตโนมัติ)
    ports:
      - "8080:8080"               # = -p 8080:8080  (8080 ใช้ Swagger ได้ — ดูหมายเหตุ)
    restart: unless-stopped       # ล่ม/รีบูต → รีสตาร์ทเอง (ไม่รีสตาร์ทถ้าเราสั่ง stop เอง)
```

---

## แบบ B (ทางเลือก): รวมทุกอย่างในไฟล์ `.yml` เดียว

เนื้อหา Dockerfile ถูกฝังใน `docker-compose.yml` ผ่าน `build.dockerfile_inline`
→ **ไม่มีไฟล์ `Dockerfile` แยก** เหลือไฟล์เดียวจบ (ต้องใช้ Compose v2.17+)

**ไฟล์ `docker-compose.yml`:**
```yaml
services:
  api:
    image: go-api-service:local   # = -t go-api-service:local
    container_name: go-api-local  # = --name go-api-local
    env_file: .env                # = --env-file .env  (compose อ่านให้เอง)
    ports:
      - "8080:8080"               # = -p 8080:8080  (8080 ใช้ Swagger ได้ — ดูหมายเหตุ)
    restart: unless-stopped       # ล่ม/รีบูต → รีสตาร์ทเอง (ไม่รีสตาร์ทถ้าเราสั่ง stop เอง)
    build:
      context: .                  # build context
      dockerfile_inline: |        # เนื้อหา Dockerfile multi-stage ฝังตรงนี้
        FROM golang:1.26 AS build
        WORKDIR /src
        COPY go.mod go.sum ./
        RUN go mod download
        COPY . .
        RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
            go build -trimpath -ldflags="-s -w" -o /app ./cmd/api/...
        FROM gcr.io/distroless/static-debian12:nonroot
        COPY --from=build /app /app
        EXPOSE 8080
        USER nonroot:nonroot
        ENTRYPOINT ["/app"]
```

> **เทียบ 2 แบบ:** แบบ A อ่าน/แก้ Dockerfile ง่ายกว่า (มี tooling: syntax highlight + hadolint)
> เหมาะกับ build ซับซ้อน — โปรเจกต์นี้เลือกแบบ A ·
> แบบ B เหลือไฟล์เดียว สะอาดตา เหมาะกับโปรเจกต์เล็ก แต่ไม่มี tooling ของ Dockerfile

---

## 1. Build + run (เหมือนกันทั้ง 2 แบบ)

```powershell
docker compose up -d --build    # build + run ครบในบรรทัดเดียว
```

| ส่วน | ความหมาย |
|------|----------|
| `up` | สร้าง + start container ตาม compose |
| `-d` | รันเบื้องหลัง (detached) ไม่ค้าง terminal |
| `--build` | บังคับ build image ใหม่ก่อนรัน (ครั้งแรก/หลังแก้โค้ด) |

- ครั้งแรกช้า (~3 นาที) เพราะดาวน์โหลด base `golang:1.26`; ครั้งต่อไปเร็วเพราะ cache layer
- ครั้งต่อไปถ้าโค้ดไม่เปลี่ยน ใช้ `docker compose up -d` เฉยๆ (ไม่ต้อง `--build`)

> ⚠️ **`env_file: .env` ขาดไม่ได้** — image เป็น distroless ก๊อปแค่ binary ไม่มี `.env` ข้างใน
> (`.env` ถูก gitignore เพราะมีรหัสผ่าน) ถ้าขาด container จะ fall back `DB_HOST=localhost`
> → ต่อ DB ไม่ติด → `Exited (1)` ทันที

> ⚠️ **เรื่องพอร์ตกับ Swagger:** `docs/swagger.json` ฮาร์ดโค้ด `"host": "localhost:8080"`
> เวลากด **Try it out** Swagger จะยิงไป `localhost:8080` เสมอ — ดังนั้นต้อง map `"8080:8080"`
> ถ้า map พอร์ตอื่น (เช่น 9090) จะขึ้น **`Failed to fetch`** (ไม่ใช่ CORS — CORS เปิด `*` แล้ว แค่พอร์ตไม่ตรง)
> *(แก้หายขาด: ลบ `@host` ใน `cmd/api/main.go` แล้ว `swag init` ใหม่ → Swagger ยิงแบบ relative ถูกทุกพอร์ต)*
> ถ้าพอร์ต 8080 บนเครื่องถูกยึด (เช่น `go run .` ค้าง) เปลี่ยนเป็น `"9090:8080"` ได้ (แต่ Swagger จะใช้ไม่ได้)

---

## 2. ตรวจ + ทดสอบ

```powershell
docker compose ps               # ดูสถานะ container
curl http://localhost:8080/readyz                       # ต่อ DB ติด → {"success":true,"data":{"status":"ready"}}
curl "http://localhost:8080/api/v1/customers?limit=1"   # ดึงข้อมูลจริง
```
Swagger UI: `http://localhost:8080/swagger/index.html` (ต้อง map 8080:8080)

---

## 3. คำสั่งที่ใช้บ่อย

```powershell
docker compose up -d --build    # build + run (หลังแก้โค้ด Go ใช้อันนี้บรรทัดเดียวจบ)
docker compose up -d            # run เฉยๆ ถ้าโค้ดไม่เปลี่ยน
docker compose ps               # ดูสถานะ
docker compose logs -f api      # ดู log realtime (Ctrl+C เพื่อออก)
docker compose stop             # หยุดชั่วคราว
docker compose start            # เปิดต่อ
docker compose down             # stop + ลบ container + network
docker compose down --rmi local # ลบ image ด้วย (เคลียร์หมด)
```

**ตารางเทียบ** คำสั่ง raw docker เดิม → compose:

| งาน | raw docker (เดิม) | compose (ตอนนี้) |
|------|------------------|------------------|
| build | `docker build -t go-api-service:local .` | `docker compose build` |
| run | `docker run -d --name ... --env-file .env -p 8080:8080 ...` | `docker compose up -d` |
| build + run | (2 คำสั่ง) | `docker compose up -d --build` |
| ดู log | `docker logs -f go-api-local` | `docker compose logs -f api` |
| stop | `docker stop go-api-local` | `docker compose stop` |
| ลบ | `docker rm -f go-api-local` | `docker compose down` |
| สถานะ | `docker ps` | `docker compose ps` |

---

## 4. หมายเหตุ

- `.env` ยังถูก gitignore และ **ไม่ถูก build เข้า image** — compose อ่านส่งเข้า container ตอน run เท่านั้น
- ใช้ `docker compose` (เว้นวรรค = v2 มากับ Docker Desktop) ไม่ใช่ `docker-compose` (ขีด = ของเก่า)
- วันหลังถ้าจะเพิ่ม SQL Server / Redis เป็น container ก็เพิ่ม service ใหม่ในไฟล์เดิมได้เลย
- ถ้า build cache เพี้ยน (Windows/WSL): `docker builder prune -f` แล้ว `wsl --shutdown` ก่อน build ใหม่

---

## ปัญหาที่เจอบ่อย

| อาการ | สาเหตุ / วิธีแก้ |
|-------|----------------|
| `bind: Only one usage of each socket address...` | พอร์ต host ถูกใช้อยู่ → เปลี่ยนเลขซ้ายใน `ports` เช่น `"9090:8080"` |
| `Cannot connect to the Docker daemon` | Docker Desktop ยังไม่เปิด → เปิดโปรแกรมรอจนพร้อม |
| `curl` ไม่ตอบ | เช็ก `docker compose ps` ว่ายังรัน + ดู `docker compose logs api` |
| แก้โค้ดแล้วผลไม่เปลี่ยน | ลืม build ใหม่ → `docker compose up -d --build` |
| `dockerfile_inline` ไม่รู้จัก | Compose เวอร์ชันเก่า → อัปเดต Docker Desktop (ต้อง v2.17+) |
| Swagger `Failed to fetch` | ไม่ได้ map `8080:8080` → ดูหมายเหตุพอร์ตข้อ 1 |
