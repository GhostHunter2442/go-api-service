# Docker — คู่มือ build & run บนเครื่อง (local)

คู่มือนี้สำหรับ build & run `go-api-service` ด้วย Docker บนเครื่องตัวเอง
อ่านครั้งเดียวจบ ใช้ซ้ำได้ทุกครั้งที่กลับมาทำงาน

> สรุปสถาปัตยกรรม: Go (`net/http`) ฟังพอร์ต **8080** ในคอนเทนเนอร์เสมอ
> เราแมปออกมาที่พอร์ตเครื่องผ่าน `-p <host>:8080`

---

## 0. เตรียมก่อนเริ่ม (ตรวจครั้งเดียว)

ต้องมี **Docker Desktop** เปิดทำงานอยู่ (ไอคอนวาฬมุมขวาล่างต้องเป็นสีปกติ ไม่ใช่ "starting")

ตรวจว่า Docker พร้อม:
```powershell
docker version
```
ถ้าเห็นทั้งส่วน `Client:` และ `Server:` แปลว่า daemon ทำงานแล้ว — ไปต่อได้

---

## 1. Build image

จากโฟลเดอร์โปรเจกต์ (ที่มี `Dockerfile`):
```powershell
docker build -t go-api-service:local .
```

| ส่วน | ความหมาย |
|------|----------|
| `docker build` | สั่งสร้าง image |
| `-t go-api-service:local` | ตั้งชื่อ:แท็ก (`ชื่อ:tag`) ให้ image |
| `.` | ใช้ `Dockerfile` ในโฟลเดอร์ปัจจุบันเป็นต้นแบบ |

- ครั้งแรกช้า (~3 นาที) เพราะต้องดาวน์โหลด base image `golang:1.26`
- ครั้งต่อไปเร็วมาก เพราะ Docker cache layer ไว้ จะ build ใหม่เฉพาะส่วนที่โค้ดเปลี่ยน

**Dockerfile นี้เป็นแบบ multi-stage:**
1. stage `build` — ใช้ `golang:1.26` คอมไพล์เป็น static binary (`CGO_ENABLED=0`)
2. stage runtime — ใช้ `distroless/static` (ไม่มี shell, รันด้วย non-root) ก๊อปแค่ binary เข้าไป

ผลลัพธ์: image เล็กมาก (~8 MB) และปลอดภัยกว่า image ที่มี OS เต็มๆ

---

## 2. ตรวจว่า image ถูกสร้าง

```powershell
docker images go-api-service
```
ควรเห็นแถว `go-api-service  local  ...  ~8MB`

---

## 3. รัน container

```powershell
docker run -d --name go-api-local -p 9090:8080 go-api-service:local
```

| ส่วน | ความหมาย |
|------|----------|
| `-d` | รันเบื้องหลัง (detached) ไม่ค้าง terminal |
| `--name go-api-local` | ตั้งชื่อ container ให้สั่งงานต่อง่าย |
| `-p 9090:8080` | `<พอร์ตบนเครื่อง>:<พอร์ตในคอนเทนเนอร์>` |
| `go-api-service:local` | image ที่จะรัน |

> **ทำไมใช้ 9090?** พอร์ต 8080 บนเครื่องนี้ถูกโปรเซสอื่นยึดอยู่ (เช่น `go run .` ที่ค้าง)
> เลยแมปออกมาที่ 9090 แทน ถ้าพอร์ต 8080 ว่างก็ใช้ `-p 8080:8080` ได้ตามปกติ

ตรวจว่ารันอยู่:
```powershell
docker ps --filter name=go-api-local
```

---

## 4. ทดสอบ endpoint

```powershell
curl http://localhost:9090/hello
```
ควรได้:
```json
{"message":"Hello, World!"}
```

---

## 5. คำสั่งจัดการ container ที่ใช้บ่อย

```powershell
docker logs go-api-local        # ดู log
docker logs -f go-api-local     # ดู log แบบ realtime (Ctrl+C เพื่อออก)
docker stop go-api-local        # หยุดชั่วคราว
docker start go-api-local       # เปิดอันเดิมต่อ
docker rm -f go-api-local       # ลบ container ทิ้ง (บังคับแม้ยังรันอยู่)
docker ps                       # ดู container ที่กำลังรัน
docker ps -a                    # ดูทั้งหมด รวมที่หยุดแล้ว
```

---

## 6. หลังแก้โค้ด Go → build ใหม่

Docker image เป็น snapshot ของโค้ด ณ ตอน build — แก้โค้ดแล้วต้อง build ใหม่เสมอ
รัน 3 บรรทัดนี้ติดกัน (ลบตัวเก่า → build → รันใหม่):
```powershell
docker rm -f go-api-local
docker build -t go-api-service:local .
docker run -d --name go-api-local -p 9090:8080 go-api-service:local
```

---

## 7. ล้างของที่ไม่ใช้ (เป็นครั้งคราว)

```powershell
docker rm -f go-api-local          # ลบ container นี้
docker rmi go-api-service:local    # ลบ image นี้
docker image prune -f              # ลบ image ที่ไม่มีชื่อ (dangling) คืนพื้นที่
```

---

## ปัญหาที่เจอบ่อย

| อาการ | สาเหตุ / วิธีแก้ |
|-------|----------------|
| `bind: Only one usage of each socket address...` | พอร์ตบนเครื่องถูกใช้อยู่ → เปลี่ยนพอร์ตซ้าย เช่น `-p 9091:8080` |
| `Cannot connect to the Docker daemon` | Docker Desktop ยังไม่เปิด → เปิดโปรแกรมรอจนพร้อม |
| `curl` ไม่ตอบ | เช็ก `docker ps` ว่า container ยังรัน + ดู `docker logs go-api-local` |
| แก้โค้ดแล้วผลไม่เปลี่ยน | ลืม build ใหม่ → ทำตามข้อ 6 |
| `name "go-api-local" is already in use` | มี container ชื่อนี้ค้าง → `docker rm -f go-api-local` ก่อน |
