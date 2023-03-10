package models

type Medicines struct {
	Base
	Uuid     string `json:"uuid" gorm:"primaryKey;autoIncrement:false"`
	Name     string `json:"name,omitempty"`
	Dosage   string `json:"dosage,omitempty"`
	Types    string `json:"types"`
	Interval int    `json:"interval"`
}
