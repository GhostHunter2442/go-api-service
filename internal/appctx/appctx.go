// Package appctx เก็บ/อ่านค่า request-scoped (เช่น customer_id) บน context.Context มาตรฐาน
//
// หลักการ (Go idiom แบบ Uber/Google):
//   - key เป็น "unexported type" (ctxKey) ไม่ใช่ string → ไม่มี package อื่นสร้าง key ชนกันได้
//     (context.WithValue เทียบ key ด้วย type+value; type ที่ private จึง unique ข้าม package)
//   - เก็บบน context.Context มาตรฐาน ไม่ผูกกับ gin → service/repository อ่านต่อได้โดยไม่รู้จัก HTTP
//   - เปิดเฉพาะ With.../getter ออกไป ปิด key ไว้ภายใน → จุดแก้/จุดอ่านมีที่เดียว
package appctx

import "context"

// ctxKey เป็น type ส่วนตัวของ package — กัน key ชนกับ context value ของ package อื่น
type ctxKey int

const (
	customerIDKey ctxKey = iota
)

// WithCustomerID คืน context ใหม่ที่ฝัง customer_id (ไม่แก้ context เดิม — context เป็น immutable)
func WithCustomerID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, customerIDKey, id)
}

// CustomerID ดึง customer_id จาก context; ok=false ถ้าไม่มี/type ไม่ตรง
func CustomerID(ctx context.Context) (uint, bool) {
	id, ok := ctx.Value(customerIDKey).(uint)
	return id, ok
}
