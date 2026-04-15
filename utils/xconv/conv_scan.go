package xconv

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

func Scan(b []byte, any any) error {
	switch v := any.(type) {
	case nil:
		return fmt.Errorf("http: Scan(nil)")
	case *string:
		*v = String(b)
		return nil
	case *[]byte:
		*v = b
		return nil
	case *int:
		var err error
		*v, err = strconv.Atoi(String(b))
		return err
	case *int8:
		n, err := strconv.ParseInt(String(b), 10, 8)
		if err != nil {
			return err
		}
		*v = int8(n)
		return nil
	case *int16:
		n, err := strconv.ParseInt(String(b), 10, 16)
		if err != nil {
			return err
		}
		*v = int16(n)
		return nil
	case *int32:
		n, err := strconv.ParseInt(String(b), 10, 32)
		if err != nil {
			return err
		}
		*v = int32(n)
		return nil
	case *int64:
		n, err := strconv.ParseInt(String(b), 10, 64)
		if err != nil {
			return err
		}
		*v = n
		return nil
	case *uint:
		n, err := strconv.ParseUint(String(b), 10, 64)
		if err != nil {
			return err
		}
		*v = uint(n)
		return nil
	case *uint8:
		n, err := strconv.ParseUint(String(b), 10, 8)
		if err != nil {
			return err
		}
		*v = uint8(n)
		return nil
	case *uint16:
		n, err := strconv.ParseUint(String(b), 10, 16)
		if err != nil {
			return err
		}
		*v = uint16(n)
		return nil
	case *uint32:
		n, err := strconv.ParseUint(String(b), 10, 32)
		if err != nil {
			return err
		}
		*v = uint32(n)
		return nil
	case *uint64:
		n, err := strconv.ParseUint(String(b), 10, 64)
		if err != nil {
			return err
		}
		*v = n
		return nil
	case *float32:
		n, err := strconv.ParseFloat(String(b), 32)
		if err != nil {
			return err
		}
		*v = float32(n)
		return err
	case *float64:
		var err error
		*v, err = strconv.ParseFloat(String(b), 64)
		return err
	case *bool:
		*v = len(b) == 1 && b[0] == '1'
		return nil
	case *time.Time:
		var err error
		*v, err = time.Parse(time.RFC3339Nano, String(b))
		return err
	case encoding.BinaryUnmarshaler:
		return v.UnmarshalBinary(b)
	default:
		var (
			rv   = reflect.ValueOf(v)
			kind = rv.Kind()
		)

		if kind != reflect.Ptr {
			return fmt.Errorf("can't unmarshal %T", v)
		}

		switch kind = rv.Elem().Kind(); kind {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			return json.Unmarshal(b, v)
		}

		return fmt.Errorf("can't unmarshal %T", v)
	}
}
