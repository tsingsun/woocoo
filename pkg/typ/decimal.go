package typ

import (
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"io"
)

//Decimal is extended from github.com/shopspring/decimal to support ent ORM
type Decimal struct {
	decimal.Decimal
}

func (d *Decimal) Scan(value interface{}) error {
	if v, ok := value.(decimal.Decimal); ok {
		d.Decimal = v
		return nil
	}
	return d.Decimal.Scan(value)
}

// MarshalGQL support gengql
func (d Decimal) MarshalGQL(w io.Writer) {
	io.WriteString(w, d.String())
}

func (d *Decimal) UnmarshalGQL(v interface{}) (err error) {
	switch v := v.(type) {
	case string:
		d.Decimal, err = decimal.NewFromString(v)
		return
	case int:
		d.Decimal = decimal.NewFromInt(int64(v))
		return
	case int64:
		d.Decimal = decimal.NewFromInt(v)
		return
	case float64:
		d.Decimal = decimal.NewFromFloat(v)
		return
	case json.Number:
		d.Decimal, err = decimal.NewFromString(string(v))
		return
	default:
		return fmt.Errorf("%T is not an decimal", v)
	}
}