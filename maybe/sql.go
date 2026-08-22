package maybe

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
)

type valuer interface {
	sql.Scanner
	driver.Valuer
}

//nolint:nestif
func (m *Maybe[T]) Scan(src any) error {
	if src == nil {
		// DB column is present but null - explicit None, not Absent
		m.state = stateNone
		return nil
	}

	if scanner, ok := convertToScanner(&m.v); ok {
		err := scanner.Scan(src)
		if err != nil {
			return err
		}
		m.state = stateJust
		return err
	}

	if scanner, ok := convertToScanner(m.v); ok {
		err := scanner.Scan(src)
		if err != nil {
			return err
		}
		m.state = stateJust
		return err
	}

	var v T
	if nullObject := sqlNullObject[T](v, false); nullObject != nil {
		err := nullObject.Scan(src)
		if err != nil {
			return err
		}

		value, err := nullObject.Value()
		if err != nil {
			return err
		}

		var ok bool
		v, ok = value.(T)
		if !ok {
			castedV, casted := cast(value, v)
			if !casted {
				return fmt.Errorf("failed to type cast %T value to %T by returned from driver.Valuer %T", v, value, nullObject)
			}

			v, ok = castedV.(T)
			if !ok {
				return fmt.Errorf("failed to type assert %T value to %T by returned from reflect.Convert %T", v, value, castedV)
			}
		}

		m.v, m.state = v, stateJust
		return nil
	}

	v, ok := src.(T)
	if !ok {
		return fmt.Errorf("failed to type cast from %T to %T", src, v)
	}

	m.v, m.state = v, stateJust
	return nil
}

func (m Maybe[T]) Value() (driver.Value, error) {
	if m.state != stateJust {
		return nil, nil
	}

	// Pass as pointer since value method may be implemented on pointer receiver
	if v, ok := convertToValuer(&m.v); ok {
		return v.Value()
	}

	// Pass v itself since value method may be implemented on non-pointer receiver
	if v, ok := convertToValuer(m.v); ok {
		return v.Value()
	}

	if nullObject := sqlNullObject[T](m.v, true); nullObject != nil {
		return nullObject.Value()
	}

	return m.v, nil
}

func convertToScanner(i any) (v sql.Scanner, ok bool) {
	v, ok = i.(sql.Scanner)
	return
}

func convertToValuer(i any) (v driver.Valuer, ok bool) {
	v, ok = i.(driver.Valuer)
	return
}

// sqlNullObject converts empty interface to sql null object to be able to use more options of conversion types from database
// For driver compatible types uses sql.Null
// Some types like int uses sql.NullInt64 to be driver compatible
func sqlNullObject[T any](t any, valid bool) valuer {
	if driver.IsValue(t) {
		v := t.(T) //nolint:forcetypeassert // callers always pass a T value boxed in any
		return &sql.Null[T]{
			V:     v,
			Valid: valid,
		}
	}

	switch e := t.(type) {
	// types in this case is not mentioned in driver.IsValue
	// but unmarshalling for them supported by sql.Null types in convertAssign
	case byte,
		int16,
		int32:
		v := t.(T) //nolint:forcetypeassert // callers always pass a T value boxed in any
		return &sql.Null[T]{
			V:     v,
			Valid: valid,
		}
	// support of null int via NullInt64
	// since go int has size between 32 and 64 use in64 to do not lose values
	case int:
		return &sql.NullInt64{
			Int64: int64(e),
			Valid: valid,
		}
	default:
		return nil
	}
}

func cast(v, t any) (any, bool) {
	rv := reflect.ValueOf(v)
	tType := reflect.TypeOf(t)

	// check that can convert v to T
	if !rv.Type().ConvertibleTo(tType) {
		return nil, false
	}

	value := rv.Convert(tType)
	return value.Interface(), true
}
