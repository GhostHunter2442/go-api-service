package httpclient

import "context"

// ctxKey เป็น type ส่วนตัว — กัน key ชนกับ context value ของ package อื่น (Go idiom)
type ctxKey int

const requestIDKey ctxKey = iota

// WithRequestID ฝัง request id ลง context เพื่อให้ทุก outbound call แนบ header X-Request-ID อัตโนมัติ
// เรียกจาก middleware ขอบนอก (requestid) ครั้งเดียว แล้ว layer ล่างไม่ต้องสนใจอีก
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// requestIDFromContext อ่าน request id (ว่างถ้าไม่มี) — ใช้ภายในตอน set header
func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
