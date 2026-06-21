# CLAUDE.md

แนวทางสำหรับ Claude Code เมื่อทำงานกับ repo นี้

## ภาพรวมโปรเจกต์

`go-api-service` — REST API service เขียนด้วย Go ปัจจุบันอยู่ในขั้น Hello World API
โดยใช้ `net/http` standard library (ยังไม่ใช้ framework ใดๆ)

- **Module:** `github.com/apidet/go-api-service`
- **Go version:** 1.26 (ดู directive ใน `go.mod`)
- **Remote:** https://github.com/GhostHunter2442/go-api-service (branch หลัก: `main`)

## โครงสร้างปัจจุบัน

```
go-api-service/
├── main.go         # entrypoint + HTTP server (port :8080)
├── go.mod          # module definition (ยังไม่มี dependency ภายนอก)
├── .gitignore
├── CLAUDE.md
└── README.md
```

**Endpoint ที่มี:**
- `GET /hello` → `{"message":"Hello, World!"}`

## คำสั่งที่ใช้บ่อย

```powershell
go run .                      # รัน dev mode (server ที่ :8080)
go build -o bin/app.exe .     # build binary (bin/ ถูก gitignore)
go vet ./...                  # static analysis
go fmt ./...                  # format
go test ./...                 # รัน test ทั้งหมด
go mod tidy                   # sync dependency
```

**ทดสอบ endpoint:**
```powershell
curl http://localhost:8080/hello
```

## ข้อตกลง / Conventions

- ใช้ **method-aware routing** ของ `http.ServeMux` (Go 1.22+) เช่น `mux.HandleFunc("GET /hello", h)`
  — อย่าถอยกลับไปเช็ค `r.Method` เองในแต่ละ handler
- JSON response ใช้ struct + `encoding/json` (ตั้ง `Content-Type: application/json` เสมอ)
- ตั้ง field tag เป็น `snake_case` (`json:"..."`)
- เมื่อโปรเจกต์โตขึ้น แนะนำแยกเป็น `cmd/api/` (entrypoint) + `internal/` (handler, service, repository)
  — โค้ดใน `internal/` import จากนอก module ไม่ได้ เหมาะกับ encapsulation

## หมายเหตุ Environment (Windows)

- Shell หลักคือ **PowerShell** — เลี่ยง here-string ที่มีอักขระพิเศษ (`{}`, `!`, `:`) กับ
  `git commit -m`; ใช้ `git commit -F <file>` แทนเพื่อกัน quoting พัง
- git ตั้ง `credential.helper = manager` (GCM) ไว้แล้ว push/pull ไม่ต้อง login ซ้ำ
- การรัน git ผ่าน PowerShell อาจโชว์ stderr เป็น `RemoteException` ทั้งที่สำเร็จ
  — ดูบรรทัดผลลัพธ์จริง (เช่น `xxx..yyy main -> main`) เป็นหลัก

## งานที่ยังทำต่อได้ (backlog ที่คุยกันไว้)

- [ ] แยกโครงสร้างเป็น `cmd/internal`
- [ ] เพิ่ม graceful shutdown (`signal.NotifyContext` + `server.Shutdown`)
- [ ] เพิ่ม unit test ให้ handler (`net/http/httptest`)
- [ ] พิจารณา framework (Gin / Echo / chi) เมื่อ routing ซับซ้อนขึ้น
