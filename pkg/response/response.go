// Package response นิยามรูปแบบ JSON response กลางของทั้ง API (envelope)
// ทุก endpoint ตอบในรูปนี้เพื่อให้ฝั่ง client parse แบบเดียวกันได้ทั้งหมด
package response

// Body คือ envelope มาตรฐาน
//
//	สำเร็จ : { "success": true,  "data": ..., "meta": ... }
//	ล้มเหลว: { "success": false, "error": { code, message, request_id } }
type Body struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
	Meta    any        `json:"meta,omitempty"`
}

// ErrorInfo รายละเอียด error ที่ส่งให้ client
type ErrorInfo struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// PageMeta metadata สำหรับ list แบบ paginate
type PageMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

// Success ห่อ data สำเร็จ
func Success(data any) Body {
	return Body{Success: true, Data: data}
}

// Paginated ห่อ list + meta
func Paginated(data any, meta PageMeta) Body {
	return Body{Success: true, Data: data, Meta: meta}
}

// Failure ห่อ error
func Failure(code, message, requestID string) Body {
	return Body{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message, RequestID: requestID},
	}
}
