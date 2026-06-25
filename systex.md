# GORM Finisher & Method สรุป

สรุป method ของ GORM สำหรับโปรเจกต์ `go-api-service` (**read-only** — ไม่แตะ schema)

---

## หลักการ: build vs finisher

GORM method แบ่งเป็น 2 ประเภท ต่อกันด้วย method chaining:

```go
err := r.db.WithContext(ctx).   // build  → สะสมเงื่อนไข (ยังไม่ยิง DB)
	Select(...).                // build  → สะสมเงื่อนไข
	First(&customer, id).       // finisher → ยิง SQL จริง + scan ลงตัวแปร
	Error                       // ดึง error ออกมา
```

- **build** = แค่ "จดโน้ต" ว่าจะทำอะไร — ยังไม่แตะ DB
- **finisher** = ตัวที่ประกอบ SQL จริง ยิงไป DB แล้ว scan ผลลง pointer ที่ส่งเข้าไป
- data ไหลเข้า **ตัวแปร** (ผ่าน `&x`), error ไหลออก **`.Error`** — คนละทางกัน

---

## คู่หลัก: First vs Find

```go
// ดึงแถวเดียว → ใส่ struct
var customer model.Customer
r.db.First(&customer, id)

// ดึงหลายแถว → ใส่ slice
var customers []model.Customer
r.db.Find(&customers)
```

| Method | ดึง | ปลายทาง | ไม่เจอข้อมูล |
|--------|-----|---------|-------------|
| `First` | 1 แถว | `*struct` (`&customer`) | ❌ error `ErrRecordNotFound` |
| `Find`  | หลายแถว | `*slice` (`&customers`) | ✅ ไม่ error — ได้ slice ว่าง `[]` |

> ⚠️ **`Find` ไม่ error เมื่อไม่เจอ** — คืน slice ว่าง (`len == 0`)
> ถ้าอยากรู้ว่าเจอไหม ต้องเช็ค `len(customers)` หรือ `RowsAffected` เอง

---

## Finisher ทั้งหมด (ดึงข้อมูล)

| Method | ทำอะไร | ใช้ในโปรเจกต์ |
|--------|--------|--------------|
| **`First(&x, id)`** | ดึงแถวแรก เรียงตาม PK; ไม่เจอ → error | `GetByID`, `GetByPhone` |
| **`Find(&xs)`** | ดึงทุกแถวที่ตรงเงื่อนไข ลง slice | `List` |
| `Take(&x)` | ดึง 1 แถว **ไม่เรียง** (เร็วกว่า First นิดหน่อย) | — |
| `Last(&x)` | ดึงแถวสุดท้าย (เรียง PK มาก→น้อย) | — |
| `Count(&n)` | นับจำนวนแถว (`int64`) — ไม่ดึง data | ทำ total ของ pagination |
| `Scan(&x)` | ดึงลง struct ที่ไม่ใช่ model (raw query / aggregate) | — |
| `Pluck("col", &xs)` | ดึงคอลัมน์เดียวลง slice เช่น `[]string` | — |

---

## Build (ต่อหน้า finisher — ไม่ยิง DB เอง)

| Method | ทำอะไร | ตัวอย่าง |
|--------|--------|---------|
| `WithContext(ctx)` | ผูก request context (timeout/cancel) | — |
| `Where(...)` | เงื่อนไข WHERE | `Where("status = ?", 1)` |
| `Select(...)` | เลือกคอลัมน์ (เลี่ยง field อ่อนไหว + ลด payload) | `Select("customer_id", "firstname")` |
| `Order(...)` | เรียงลำดับ | `Order("customer_id ASC")` |
| `Limit(n)` | จำกัดจำนวน | `Limit(50)` |
| `Offset(n)` | ข้ามแถว (paginate) | `Offset(100)` |

---

## ตัวอย่างเทียบ — ดึงหนึ่ง vs ดึงหลาย

```go
// ── ดึงคนเดียว (GetByID) ──
var customer model.Customer
err := r.db.WithContext(ctx).
	First(&customer, id).Error
// ไม่เจอ → err = ErrRecordNotFound

// ── ดึงหลายคน (List) ──
var customers []model.Customer
err := r.db.WithContext(ctx).
	Order("customer_id ASC").
	Limit(limit).
	Offset(offset).
	Find(&customers).Error          // Find + slice
// ไม่เจอ → err = nil, customers = [] (ว่าง ไม่ error)
```

---

## field สำคัญของ `*gorm.DB` (ดูได้หลัง query)

| Field | ได้อะไร |
|-------|---------|
| `.Error` | error (`nil` = สำเร็จ) |
| `.RowsAffected` | จำนวนแถวที่กระทบ (`int64`) |
| `.Statement.SQL` | SQL string ที่ gen ออกมา |

---

## ⚠️ กลุ่มเขียน DB — โปรเจกต์นี้ "ไม่ใช้"

โปรเจกต์เป็น **read-only ห้ามแตะ schema** (ดู CLAUDE.md) — method พวกนี้ไม่ควรใช้:

`Create` · `Save` · `Update` · `Updates` · `Delete` · `AutoMigrate`

---

## Debug query

```go
// เปิด log SQL ทุก query (ตอน dev) — ที่ database.go
Logger: logger.Default.LogMode(logger.Info),   // เปลี่ยนกลับเป็น Warn ตอน prod

// หรือ debug เฉพาะ query เดียว — แทรกใน chain
r.db.WithContext(ctx).Debug().First(&customer, id)
```

ดูที่ console/terminal ที่รัน `air` หรือ `go run`:
- `[1.234ms]` → query ใช้เวลาเท่าไร
- `[rows:1]` → ได้กี่แถว (`0` = ไม่เจอ)
- SQL → เช็ค WHERE/Select ตรงที่ตั้งใจไหม
