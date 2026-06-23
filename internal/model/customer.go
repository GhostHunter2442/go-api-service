// Package model เก็บ domain model / GORM entity
package model

import "time"

// Customer map กับตาราง dbo.customers ใน easymoney_dev
//
// หลักการ map:
//   - คอลัมน์ NULL → ใช้ pointer (*string/*int/*time.Time) กัน error ตอน scan ค่า NULL
//   - ใส่ column tag ทุก field (กัน GORM แปลงชื่อย่อเพี้ยน เช่น IDCard, ID)
//   - field อ่อนไหว (password, access_token) ใส่ json:"-" ไม่ให้หลุดออก response
type Customer struct {
	CustomerID                   uint       `gorm:"column:customer_id;primaryKey" json:"customer_id"`
	CustomerCode                 *string    `gorm:"column:customer_code" json:"customer_code"`
	AccpacCustomerCode           *string    `gorm:"column:accpac_customer_code" json:"accpac_customer_code"`
	CustomerType                 *string    `gorm:"column:customer_type" json:"customer_type"`
	PhoneNumber                  string     `gorm:"column:phone_number" json:"phone_number"` // NOT NULL, unique
	Password                     *string    `gorm:"column:password" json:"-"`
	AccessToken                  *string    `gorm:"column:access_token" json:"-"`
	PhoneSN                      *string    `gorm:"column:phone_sn" json:"phone_sn"`
	Firstname                    *string    `gorm:"column:firstname" json:"firstname"`
	Lastname                     *string    `gorm:"column:lastname" json:"lastname"`
	Gender                       *string    `gorm:"column:gender" json:"gender"` // N=ไม่ระบุ, M=ชาย, F=หญิง
	DateOfBirth                  *time.Time `gorm:"column:date_of_birth" json:"date_of_birth"`
	Email                        *string    `gorm:"column:email" json:"email"`
	Status                       *string    `gorm:"column:status" json:"status"`
	TotalPoint                   *int       `gorm:"column:total_point" json:"total_point"`
	LastActive                   *time.Time `gorm:"column:last_active" json:"last_active"`
	LastAccess                   *time.Time `gorm:"column:last_access" json:"last_access"`
	IDCard                       *string    `gorm:"column:id_card" json:"id_card"`
	IsVerify                     *int       `gorm:"column:is_verify" json:"is_verify"`
	CreateDate                   *time.Time `gorm:"column:create_date" json:"create_date"`
	UpdateDate                   *time.Time `gorm:"column:update_date" json:"update_date"`
	PawnAuthenStatus             *string    `gorm:"column:pawn_authen_status" json:"pawn_authen_status"`
	PawnRegisDate                *time.Time `gorm:"column:pawn_regis_date" json:"pawn_regis_date"`
	LastUpdatePassword           *time.Time `gorm:"column:last_update_password" json:"last_update_password"`
	CountUpdatePassword          *int       `gorm:"column:count_update_password" json:"count_update_password"`
	Permission1                  *string    `gorm:"column:permission_1" json:"permission_1"`
	Permission2                  *string    `gorm:"column:permission_2" json:"permission_2"`
	Permission3                  *string    `gorm:"column:permission_3" json:"permission_3"`
	RemarkDelAccountMasterDataID *int       `gorm:"column:remark_del_acccount_master_data_id" json:"remark_del_acccount_master_data_id"`
	RefCustomerID                *int       `gorm:"column:refcustomer_id" json:"refcustomer_id"`
	RefComment                   *string    `gorm:"column:refcomment" json:"refcomment"`
	ProductIDRecommend           *string    `gorm:"column:product_id_recommend" json:"product_id_recommend"`
	NameInFacebook               *string    `gorm:"column:name_in_facebook" json:"name_in_facebook"`
	NameInLine                   *string    `gorm:"column:name_in_line" json:"name_in_line"`
	TicketZone                   *string    `gorm:"column:ticket_zone" json:"ticket_zone"`
	NameInIG                     *string    `gorm:"column:name_in_ig" json:"name_in_ig"`
	EasyID                       *string    `gorm:"column:easy_id" json:"easy_id"`
	DigitFive                    *string    `gorm:"column:digit_five" json:"digit_five"`
	IDAddress                    *string    `gorm:"column:id_address" json:"id_address"`
	PhoneNumberLastModify        *string    `gorm:"column:phone_number_last_modify" json:"phone_number_last_modify"`
	CustomerIDLastModify         *int       `gorm:"column:customer_id_last_modify" json:"customer_id_last_modify"`
	CreditTerm                   *int16     `gorm:"column:credit_term" json:"credit_term"`
	QuestionnaireResults         *string    `gorm:"column:questionnaire_results" json:"questionnaire_results"`
}

// TableName กำหนดชื่อตารางให้ชัด
func (Customer) TableName() string { return "customers" }
