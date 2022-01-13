package types

import (
	"database/sql/driver"
	"fmt"
)

// YesNo is bool format by using "Y" or "N"
type YesNo bool

const (
	BoolCharYes string = "Y"
	BoolCharNo  string = "N"
)

func (b YesNo) String() string {
	if b {
		return BoolCharYes
	}
	return BoolCharNo
}

func (b YesNo) Value() (driver.Value, error) {
	return b, nil
}

func (b *YesNo) Scan(val interface{}) error {
	var s string
	switch v := val.(type) {
	case string:
		s = v
	case []uint8:
		s = string(v)
	}
	switch s {
	case "Y":
		*b = true
	case "N":
		*b = false
	default:
		return fmt.Errorf("%v is not bool type", val)
	}
	return nil
}
