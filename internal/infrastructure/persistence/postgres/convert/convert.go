// Package convert translates application row/parameter structs to and from
// sqlc and pgx value types at the persistence boundary.
package convert

import (
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	appstaff "github.com/ren0503/tycoon/internal/application/staff"
)

var (
	dateType        = reflect.TypeOf(pgtype.Date{})
	timeType        = reflect.TypeOf(pgtype.Time{})
	decimalNullType = reflect.TypeOf(decimal.NullDecimal{})
	timePtrType     = reflect.TypeOf((*time.Time)(nil))
	timeOfDayType   = reflect.TypeOf(appstaff.TimeOfDay{})
)

// To copies src into a new destination value by matching struct field names.
func To[D any](src any) D {
	var dst D
	if src == nil {
		return dst
	}
	copyValue(reflect.ValueOf(&dst).Elem(), reflect.ValueOf(src))
	return dst
}

// Slice copies a slice of structs field-for-field.
func Slice[D any, S any](src []S) []D {
	out := make([]D, len(src))
	for i := range src {
		out[i] = To[D](src[i])
	}
	return out
}

func copyValue(dst, src reflect.Value) {
	if !dst.CanSet() || !src.IsValid() {
		return
	}
	st, dt := src.Type(), dst.Type()

	if st == dateType && dt == timePtrType {
		date := src.Interface().(pgtype.Date)
		if date.Valid {
			t := date.Time
			dst.Set(reflect.ValueOf(&t))
		}
		return
	}
	if st == timePtrType && dt == dateType {
		if src.IsNil() {
			dst.Set(reflect.Zero(dt))
			return
		}
		t := src.Interface().(*time.Time)
		dst.Set(reflect.ValueOf(pgtype.Date{Time: *t, Valid: true}))
		return
	}
	if st == timeType && dt == timeOfDayType {
		t := src.Interface().(pgtype.Time)
		dst.Set(reflect.ValueOf(appstaff.TimeOfDay{Microseconds: t.Microseconds, Valid: t.Valid}))
		return
	}
	if st == timeOfDayType && dt == timeType {
		t := src.Interface().(appstaff.TimeOfDay)
		dst.Set(reflect.ValueOf(pgtype.Time{Microseconds: t.Microseconds, Valid: t.Valid}))
		return
	}
	if st == decimalNullType && dt.Kind() == reflect.Pointer && dt.Elem().Kind() == reflect.Float64 {
		d := src.Interface().(decimal.NullDecimal)
		if d.Valid {
			f := d.Decimal.InexactFloat64()
			dst.Set(reflect.ValueOf(&f))
		}
		return
	}
	if st.Kind() == reflect.Pointer && st.Elem().Kind() == reflect.Float64 && dt == decimalNullType {
		if src.IsNil() {
			dst.Set(reflect.Zero(dt))
			return
		}
		f := src.Interface().(*float64)
		dst.Set(reflect.ValueOf(decimal.NullDecimal{Decimal: decimal.NewFromFloat(*f), Valid: true}))
		return
	}

	switch dst.Kind() {
	case reflect.Pointer:
		if src.Kind() == reflect.Pointer {
			if src.IsNil() {
				return
			}
			if src.Type().AssignableTo(dst.Type()) {
				dst.Set(src)
				return
			}
			if src.Type().Elem().AssignableTo(dst.Type().Elem()) {
				p := reflect.New(dst.Type().Elem())
				copyValue(p.Elem(), src.Elem())
				dst.Set(p)
				return
			}
		}
		p := reflect.New(dst.Type().Elem())
		copyValue(p.Elem(), src)
		dst.Set(p)
	case reflect.Struct:
		if src.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < dst.NumField(); i++ {
			df := dst.Field(i)
			sf := src.FieldByName(dst.Type().Field(i).Name)
			if sf.IsValid() {
				copyValue(df, sf)
			}
		}
	case reflect.Slice:
		if src.Type().AssignableTo(dst.Type()) {
			dst.Set(src)
		}
	default:
		if src.Type().AssignableTo(dst.Type()) {
			dst.Set(src)
		} else if src.Type().ConvertibleTo(dst.Type()) {
			dst.Set(src.Convert(dst.Type()))
		}
	}
}
