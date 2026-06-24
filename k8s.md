# Kubernetes — คู่มือ deploy go-api-service แบบไม่ให้ล่ม (local)

ทำให้ API รันหลาย replica + ซ่อมตัวเอง + อัปเดตแบบ zero-downtime บน Kubernetes
ใช้ **Kubernetes ที่มากับ Docker Desktop** (ไม่ต้องลงอะไรเพิ่ม)

> **อ่านก่อน — ขอบเขตของ local cluster:**
> เครื่องเดียว (single-node) ทำให้ได้ **กลไกกัน downtime ระดับ pod**: หลาย replica, self-healing,
> readiness/liveness probe, rolling update — เทสต์ได้ครบ
> แต่ **"node ดับแล้วไม่ล่ม" จริงๆ ต้องมีหลาย node** (cloud: GKE/EKS/AKS หรือ on-prem cluster)
> เพราะถ้า Docker Desktop ดับ ทุกอย่างก็ดับตาม — local ไว้เรียน/เทสต์กลไก, production ค่อยขึ้น multi-node

---

## ⚡ Quick Reference — คำสั่งที่ใช้บ่อย

> namespace = `go-api` · deployment = `go-api` · port = `30080` · build ผ่าน `docker compose build` (แบบ A: อ่าน `Dockerfile` แยก)

**เปิดใช้งาน (หลังเปิดเครื่อง / Docker Desktop):**
```powershell
kubectl config use-context docker-desktop                  # เผื่อ context หลุด
kubectl config set-context --current --namespace=go-api    # เผื่อ namespace หลุด
kubectl get pods                                           # ต้องเห็น READY 1/1 Running
```

**เข้าใช้งาน (ลิงก์):**
```
http://localhost:30080/readyz
http://localhost:30080/api/v1/customers?limit=2
http://localhost:30080/swagger/index.html
```

**ดูสถานะ / log / debug:**
```powershell
kubectl get pods                       # pod ทั้งหมด + RESTARTS
kubectl get deploy,svc                 # deployment + service
kubectl logs -f -l app=go-api          # log รวมทุก pod (realtime)
kubectl describe pod <ชื่อ-pod>        # ดีบั๊กตอน pod ไม่ขึ้น (ดู Events)
```

**หลังแก้โค้ด Go → deploy เวอร์ชันใหม่ (zero-downtime):**
```powershell
docker compose build                       # build image ใหม่จาก Dockerfile → tag go-api-service:local
kubectl rollout restart deployment/go-api
kubectl rollout status deployment/go-api
```

**ปรับจำนวน replica:**
```powershell
kubectl scale deployment/go-api --replicas=4   # เพิ่ม
kubectl scale deployment/go-api --replicas=2   # ลดกลับ
```

**เทส "ไม่ล่ม":**
```powershell
kubectl delete pod <ชื่อ-pod>          # ฆ่า pod → Deployment สร้างใหม่เอง
kubectl get pods -w                    # ดูตัวใหม่เกิด (Ctrl+C ออก)
```

**ปิด / ล้าง:**
```powershell
kubectl scale deployment/go-api --replicas=0   # หยุดชั่วคราว (ไม่ลบ config)
kubectl delete -f k8s/                          # ลบทุก resource ที่ apply
kubectl delete namespace go-api                 # ลบทั้ง namespace ทีเดียว
```

> **3 คำสั่งที่ใช้บ่อยสุด:** `kubectl get pods` · `kubectl logs -f -l app=go-api` · `docker compose build && kubectl rollout restart deployment/go-api`

---

## 🚀 เริ่มใหม่จากศูนย์ (clean start) — ทีละ step + อธิบายทุกคำสั่ง

> ใช้ตอนเคลียร์หมดแล้ว (ไม่มี image/container/namespace ของ project) แต่ **k8s engine ยังเปิดอยู่**
> รันทีละ step ใน **PowerShell** ที่โฟลเดอร์ `go-api-service`

### Step 1 — เช็ก k8s พร้อม
```powershell
kubectl config use-context docker-desktop
kubectl get nodes
```
| คำสั่ง | อธิบาย |
|--------|--------|
| `kubectl config use-context docker-desktop` | ชี้ kubectl ไปที่ cluster ของ Docker Desktop (กันยิงผิด cluster ถ้าเครื่องมีหลายอัน) |
| `kubectl get nodes` | ดู node ใน cluster — ต้องเห็น `docker-desktop` STATUS `Ready` = cluster พร้อมรับงาน |

### Step 2 — build image (แบบ A: Dockerfile แยก)
```powershell
docker compose build
docker images go-api-service
```
| คำสั่ง | อธิบาย |
|--------|--------|
| `docker compose build` | อ่าน `docker-compose.yml` → เรียก `Dockerfile` build เป็น image แล้ว tag `go-api-service:local` ให้อัตโนมัติ (จาก `image:` ใน compose) |
| `docker images go-api-service` | ลิสต์ image ชื่อนี้ — ยืนยัน build สำเร็จ เห็น tag `local` |

> `deployment.yaml` ตั้ง `imagePullPolicy: Never` → k8s ใช้ image จาก local store ตรงๆ
> จึง **ต้อง build ให้มี `go-api-service:local` ในเครื่องก่อน** ไม่งั้น pod ขึ้น `ErrImageNeverPull`

### Step 3 — เช็ก image ก่อนขึ้น k8s (ออปชัน แต่แนะนำ)
```powershell
docker compose up -d
curl http://localhost:8080/readyz
docker compose down
```
| คำสั่ง | อธิบาย |
|--------|--------|
| `docker compose up -d` | รัน container เดี่ยวจาก image (`-d` = เบื้องหลัง) ที่พอร์ต 8080 — ลองว่า binary รันได้จริง |
| `curl .../readyz` | เรียก readiness probe — ได้ `status:ready` = แอป start ขึ้น + ต่อ DB ติด |
| `docker compose down` | ลบ container ทิ้ง — k8s ไม่ใช้ container ตัวนี้ ใช้แค่ **image** ที่ build ไว้ |

### Step 4 — สร้าง namespace + ชี้ context
```powershell
kubectl create namespace go-api
kubectl config set-context --current --namespace=go-api
```
| คำสั่ง | อธิบาย |
|--------|--------|
| `kubectl create namespace go-api` | สร้างห้องแยกชื่อ `go-api` ให้ resource ของ project นี้ (ไม่ปนของอื่น) |
| `kubectl config set-context --current --namespace=go-api` | ตั้ง namespace ปริยายของ context ปัจจุบัน = `go-api` → คำสั่ง kubectl ถัดไปไม่ต้องเติม `-n go-api` |

### Step 5 — apply manifest ทั้งหมด
```powershell
kubectl apply -f k8s/
```
| คำสั่ง | อธิบาย |
|--------|--------|
| `kubectl apply -f k8s/` | อ่านทุกไฟล์ `.yaml` ในโฟลเดอร์ `k8s/` → สร้าง/อัปเดต ConfigMap + Secret + Deployment + Service + PDB เข้า cluster (`-f` = from file/dir; apply = สร้างถ้ายังไม่มี / แก้ถ้ามีแล้ว) |

> `k8s/secret.yaml` ถูก gitignore แต่ยังอยู่ในเครื่อง → apply กลับมาเองได้ ไม่ต้องสร้างใหม่

### Step 6 — รอ pod พร้อม
```powershell
kubectl get pods -w
kubectl rollout status deployment/go-api
```
| คำสั่ง | อธิบาย |
|--------|--------|
| `kubectl get pods -w` | ดู pod แบบ realtime (`-w` = watch) — รอจน 2 ตัวเป็น READY `1/1` STATUS `Running` (Ctrl+C ออก) |
| `kubectl rollout status deployment/go-api` | รอ/ยืนยันว่า deployment ปล่อย pod ครบ — ได้ `successfully rolled out` |

### Step 7 — ยืนยัน k8s ทำงาน
```powershell
curl http://localhost:30080/readyz
curl "http://localhost:30080/api/v1/customers?limit=2"
```
| คำสั่ง | อธิบาย |
|--------|--------|
| `curl .../readyz` | ยิงผ่าน Service NodePort `30080` → load balance ไป 1 ใน 2 pod — ได้ 200 = ครบวงจร |
| `curl .../customers?limit=2` | ดึงข้อมูลจริงจาก DB ผ่าน k8s (limit 2 แถว) |

Swagger UI: `http://localhost:30080/swagger/index.html`

---

## ภาพรวมไฟล์ (`k8s/`)

| ไฟล์ | คืออะไร | บทบาทกัน downtime |
|------|---------|-------------------|
| `configmap.yaml` | ค่า config ไม่ลับ (APP_ENV, DB_HOST...) | — |
| `secret.yaml` | ค่าลับ (DB_USER/DB_PASSWORD) — **gitignore** | — |
| `deployment.yaml` | คุม pod: 2 replica, probe, rolling update | ⭐ หัวใจ |
| `service.yaml` | endpoint คงที่ + load balance (NodePort 30080) | ⭐ ส่ง traffic เฉพาะ pod ที่ ready |
| `pdb.yaml` | PodDisruptionBudget — เหลือ pod ≥1 เสมอ | ⭐ ตอน maintenance |

**map กับ endpoint ที่มีอยู่:** `/healthz` → liveness (ไม่แตะ DB) · `/readyz` → readiness (ping DB)

---

## Step 1 — เปิด Kubernetes ใน Docker Desktop (ทำครั้งเดียว, GUI)

1. เปิด **Docker Desktop** → ไอคอน ⚙️ **Settings**
2. เมนู **Kubernetes** → ติ๊ก **Enable Kubernetes** → **Apply & Restart**
3. รอ ~2–5 นาที (ดาวน์โหลด component) จนสถานะ Kubernetes มุมล่างซ้ายเป็น **เขียว (running)**

> ปุ่มนี้กดได้เฉพาะใน GUI — CLI เปิดให้ไม่ได้

ตรวจว่าพร้อม:
```powershell
kubectl config use-context docker-desktop   # ชี้ context ไปที่ cluster ของ Docker Desktop
kubectl get nodes                            # ต้องเห็น node 'docker-desktop' STATUS=Ready
```

---

## Step 2 — build image ให้ k8s เห็น

k8s ของ Docker Desktop ใช้ image store เดียวกับ Docker → แค่ build ไว้ในเครื่องก็พอ (ไม่ต้อง push registry)
```powershell
docker compose build                          # build จาก Dockerfile → tag go-api-service:local อัตโนมัติ
docker images go-api-service                   # ต้องเห็น tag local
```
> โปรเจกต์ใช้ **แบบ A** (มีไฟล์ `Dockerfile` แยก + `docker-compose.yml`) → build ได้ทั้ง
> `docker compose build` (tag `go-api-service:local` ให้อัตโนมัติ — แนะนำ) หรือ `docker build -t go-api-service:local .`
> `deployment.yaml` ตั้ง `imagePullPolicy: Never` = ใช้ image ในเครื่องเท่านั้น ไม่ไปโหลดจากเน็ต

---

## Step 3 — สร้าง namespace

แยกของโปรเจกต์นี้ไว้ใน namespace `go-api` (ไม่ปนกับของอื่น)
```powershell
kubectl create namespace go-api
kubectl config set-context --current --namespace=go-api   # ตั้งให้คำสั่งถัดไปอยู่ใน go-api อัตโนมัติ
```

---

## Step 4 — สร้าง secret (ถ้ายังไม่มี `k8s/secret.yaml`)

ไฟล์ `k8s/secret.yaml` ถูก gitignore (มีรหัสผ่าน) — ในเครื่องนี้สร้างไว้แล้ว
ถ้า clone ใหม่/ไฟล์หาย สร้างจาก `.env` ได้ 2 วิธี:

**วิธี A — สร้างไฟล์ (ตามแบบที่ใช้อยู่):** สร้าง `k8s/secret.yaml`
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: go-api-secret
type: Opaque
stringData:
  DB_USER: "apidej"
  DB_PASSWORD: "Hunter+2369"
```

**วิธี B — สร้างตรงจาก CLI (ไม่มีไฟล์ค้างในเครื่อง):**
```powershell
kubectl create secret generic go-api-secret -n go-api `
  --from-literal=DB_USER=apidej `
  --from-literal=DB_PASSWORD='Hunter+2369'
```

---

## Step 5 — apply ทุก manifest

```powershell
kubectl apply -f k8s/
```
สั่งครั้งเดียวสร้าง ConfigMap + Secret + Deployment + Service + PDB ครบ

---

## Step 6 — ตรวจสถานะ

```powershell
kubectl get pods -w           # รอจน 2 pod เป็น READY 1/1, STATUS Running (Ctrl+C ออก)
kubectl get deploy,svc,pdb    # ดูภาพรวม
kubectl rollout status deployment/go-api   # ยืนยัน deploy สำเร็จ
```
ถ้า pod ไม่ READY → `kubectl describe pod <ชื่อ>` ดู Events / `kubectl logs <ชื่อ>` ดู log

---

## Step 7 — ทดสอบ endpoint

```powershell
curl http://localhost:30080/readyz                       # ผ่าน Service (NodePort) → load balance 2 pod
curl "http://localhost:30080/api/v1/customers?limit=2"
```

---

## Step 8 — ทดสอบ "ไม่ down" (สำคัญสุด)

### 8.1 Self-healing — ฆ่า pod แล้วมันเกิดใหม่เอง
```powershell
kubectl get pods                          # จดชื่อ pod ตัวนึง
kubectl delete pod <ชื่อ-pod>             # จำลอง pod ตาย
kubectl get pods -w                       # เห็นตัวใหม่ถูกสร้างแทนทันที (Deployment คุมจำนวนไว้ที่ 2)
```
ระหว่างนั้น `curl http://localhost:30080/readyz` ยัง **ตอบ 200 ตลอด** เพราะอีก pod รับแทน

### 8.2 Rolling update — อัปเดตแบบไม่มี downtime
เปิดอีกหน้าต่างยิง loop ค้างไว้:
```powershell
while ($true) { (curl -s -o /dev/null -w "%{http_code} " http://localhost:30080/healthz); Start-Sleep -Milliseconds 300 }
```
อีกหน้าต่างสั่ง redeploy:
```powershell
kubectl rollout restart deployment/go-api
kubectl rollout status deployment/go-api
```
loop ควรเห็น **200 รัวๆ ไม่มี 000/5xx** (เพราะ `maxUnavailable: 0` + readiness probe)

### 8.3 Scale — เพิ่ม/ลดจำนวนก๊อปสดๆ
```powershell
kubectl scale deployment/go-api --replicas=4
kubectl get pods            # เห็น 4 ตัว
kubectl scale deployment/go-api --replicas=2
```

### 8.4 readiness ทำงานจริง — DB ต่อไม่ติด pod ถูกถอดจาก traffic
ถ้า DB (`192.168.251.120`) ล่ม → `/readyz` fail → k8s ถอด pod ออกจาก Service เอง (ไม่ส่ง request ไปเจอ error)
ดูได้จาก `kubectl get pods` คอลัมน์ READY จะกลายเป็น `0/1` ชั่วคราว แล้วกลับมาเมื่อ DB คืนชีพ

---

## Step 9 — คำสั่งจัดการที่ใช้บ่อย

```powershell
kubectl get pods                       # ดู pod
kubectl logs -f <ชื่อ-pod>             # ดู log realtime
kubectl logs -l app=go-api --tail=50   # log รวมทุก pod ที่ label app=go-api
kubectl describe pod <ชื่อ-pod>        # รายละเอียด + Events (ดีบั๊ก)
kubectl exec -it <ชื่อ-pod> -- /app    # (distroless ไม่มี shell — ส่วนใหญ่ใช้ logs/describe แทน)
kubectl rollout undo deployment/go-api # ย้อนเวอร์ชันก่อนหน้า
kubectl apply -f k8s/                  # apply ใหม่หลังแก้ manifest
```

**หลังแก้โค้ด Go:** build image ใหม่ → restart rollout
```powershell
docker compose build                       # build จาก Dockerfile → tag go-api-service:local
kubectl rollout restart deployment/go-api
```

---

## Step 10 — ลบทิ้ง / cleanup

```powershell
kubectl delete -f k8s/                 # ลบเฉพาะ resource ที่เรา apply
# หรือลบทั้ง namespace ทีเดียว:
kubectl delete namespace go-api
```
ปิด Kubernetes: Docker Desktop → Settings → Kubernetes → ติ๊ก Enable ออก → Apply & Restart

---

## ข้อจำกัด & ทางไป production

| อยากได้ | local (Docker Desktop) | production |
|---------|------------------------|------------|
| pod ตาย→ซ่อมเอง | ✅ ได้ | ✅ |
| zero-downtime deploy | ✅ ได้ | ✅ |
| **node ดับ→ไม่ล่ม** | ❌ (มี node เดียว) | ✅ ต้อง **หลาย node** (cloud/on-prem) |
| auto-scale ตามโหลด (HPA) | ⚠️ ต้องลง metrics-server เพิ่ม | ✅ |
| external load balancer/ingress | ใช้ NodePort/port-forward | LoadBalancer + Ingress + TLS |

ขั้นถัดไปเมื่อจะขึ้นจริง: push image ขึ้น registry (เลิกใช้ `imagePullPolicy: Never`),
ใช้ managed k8s หลาย node, เพิ่ม Ingress + cert, และ HorizontalPodAutoscaler

---

## ปัญหาที่เจอบ่อย

| อาการ | สาเหตุ / วิธีแก้ |
|-------|----------------|
| `ErrImageNeverPull` | k8s ไม่เห็น image → `docker compose build` ก่อน แล้วเช็ก `docker images` |
| pod `CrashLoopBackOff` | แอป start ไม่ขึ้น → `kubectl logs <pod>` ดู error (มัก env/DB) |
| READY `0/1` ค้าง | readiness `/readyz` ไม่ผ่าน = ต่อ DB ไม่ติด → เช็ก `DB_HOST` ใน ConfigMap + network ถึง `192.168.251.120` |
| `kubectl` ต่อ cluster ไม่ได้ | ยังไม่เปิด k8s ใน Docker Desktop หรือ context ผิด → `kubectl config use-context docker-desktop` |
| `localhost:30080` ไม่ตอบ | Service ยังไม่ขึ้น/pod ไม่ ready → `kubectl get svc,pods` |
| pod อยู่ namespace อื่น | ลืมตั้ง ns → เติม `-n go-api` หรือ `kubectl config set-context --current --namespace=go-api` |
