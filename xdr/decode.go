package xdr

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/fdawg4l/nfs/util"
)

var (
	EncoderDebug = false
)

func debugf(fmt string, args ...interface{}) {
	if !EncoderDebug {
		return
	}

	util.Debugf(fmt, args...)
}

func Uint32(b []byte) (uint32, []byte) {
	return binary.BigEndian.Uint32(b[0:4]), b[4:]
}

func Opaque(b []byte) ([]byte, []byte) {
	l, b := Uint32(b)
	return b[:l], b[l:]
}

func Uint32List(b []byte) ([]uint32, []byte) {
	l, b := Uint32(b)
	v := make([]uint32, l)
	for i := 0; i < int(l); i++ {
		v[i], b = Uint32(b)
	}
	return v, b
}

func Read(r io.Reader, val interface{}) error {
	if err := read(r, reflect.ValueOf(val)); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return nil
}

func read(r io.Reader, v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch t := v.Type(); t.Kind() {
	case reflect.String:
		// the length is the first element
		var l uint32
		if err := binary.Read(r, binary.BigEndian, &l); err != nil {
			return err
		}

		b := make([]byte, l, l)
		if err := binary.Read(r, binary.BigEndian, b); err != nil {
			return err
		}

		// drain the fill
		var n int
		if l%4 > 0 {
			fillBytes := make([]byte, 4-l%4)
			n, _ = r.Read(fillBytes)
		}

		debugf("len=%d string=%s pad=%d", l, string(b), n)
		v.SetString(string(b))

	case reflect.Struct:
		debugf("enter struct")
		for i := 0; i < v.NumField(); i++ {
			if err := read(r, v.Field(i)); err != nil {
				return err
			}
		}
		debugf("exit struct")

	case reflect.Uint8, reflect.Uint32:
		var val uint32
		if err := binary.Read(r, binary.BigEndian, &val); err != nil {
			return err
		}
		debugf("8|32 = 0x%x", val)
		v.SetUint(uint64(val))

	case reflect.Uint64:
		var val uint64
		if err := binary.Read(r, binary.BigEndian, &val); err != nil {
			return err
		}

		debugf("64 = 0x%x", val)
		v.SetUint(val)

	case reflect.Array:
		debugf("enter array")
		for idx := 0; idx < v.Len(); idx++ {
			val := v.Index(idx)
			if err := read(r, val); err != nil {
				return err
			}
		}
		debugf("close array")

	case reflect.Slice:
		debugf("ENTER slice")
		switch t.Elem().Kind() {
		case reflect.Uint8:
			var l uint32
			if err := binary.Read(r, binary.BigEndian, &l); err != nil {
				return err
			}

			b := make([]byte, l, l)
			if err := binary.Read(r, binary.BigEndian, b); err != nil {
				return err
			}

			// drain the fill
			if l%4 > 0 {
				fillBytes := make([]byte, 4-l%4)
				_, _ = r.Read(fillBytes)
			}

			if v.Cap() < int(l) {
				v.Set(reflect.MakeSlice(v.Type(), int(l), int(l)))
			}

			v.SetBytes(b)

		case reflect.Struct:
			slc := reflect.MakeSlice(v.Type(), 0, 0)
			for {
				val := reflect.New(v.Type().Elem())
				if val.Kind() == reflect.Ptr {
					val = val.Elem()
				}
				if err := read(r, val); err != nil {
					return err
				}

				slc = reflect.Append(slc, val)

				// slices have a "follows" 32b integer that is non-zero when an elem exists
				var follows uint32
				if err := binary.Read(r, binary.BigEndian, &follows); err != nil {
					return err
				}

				if follows == 0 {
					break
				}
			}
			v.Set(slc)

		default:
			return fmt.Errorf("rpc.read: invalid type: %v ", t.String())
		}

		debugf("EXIT slice")

	default:
		return fmt.Errorf("rpc.read: invalid type: %v ", t.String())
	}
	return nil
}
