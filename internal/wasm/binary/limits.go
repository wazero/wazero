package binary

import (
	"bytes"
	"fmt"

	"github.com/tetratelabs/wazero/internal/leb128"
)

// decodeLimitsType returns the `limitsType` (min, max) decoded with the WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#limits%E2%91%A6
//
// Extended in threads proposal: https://webassembly.github.io/threads/core/binary/types.html#limits
func decodeLimitsType(r *bytes.Reader) (min uint64, max *uint64, shared bool, is64 bool, err error) {
	var flag byte
	if flag, err = r.ReadByte(); err != nil {
		err = fmt.Errorf("read leading byte: %v", err)
		return
	}

	is64 = flag&0x04 != 0
	// Ignore the memory64 bit for decoding structure.
	flag &^= 0x04

	readMin := func() (uint64, error) {
		if is64 {
			v, _, err := leb128.DecodeUint64(r)
			if err != nil {
				return 0, fmt.Errorf("read min of limit: %v", err)
			}
			return v, nil
		}
		v, _, err := leb128.DecodeUint32(r)
		if err != nil {
			return 0, fmt.Errorf("read min of limit: %v", err)
		}
		return uint64(v), nil
	}

	readMax := func() (uint64, error) {
		if is64 {
			v, _, err := leb128.DecodeUint64(r)
			if err != nil {
				return 0, fmt.Errorf("read max of limit: %v", err)
			}
			return v, nil
		}
		v, _, err := leb128.DecodeUint32(r)
		if err != nil {
			return 0, fmt.Errorf("read max of limit: %v", err)
		}
		return uint64(v), nil
	}

	switch flag {
	case 0x00, 0x02:
		min, err = readMin()
	case 0x01, 0x03:
		min, err = readMin()
		if err != nil {
			return
		}
		var m uint64
		m, err = readMax()
		if err != nil {
			return
		}
		max = &m
	case 0x04, 0x06: // Invalid for tables, handled by caller.
		min, err = readMin()
	case 0x05, 0x07:
		min, err = readMin()
		if err != nil {
			return
		}
		var m uint64
		m, err = readMax()
		if err != nil {
			return
		}
		max = &m
	default:
		err = fmt.Errorf("%v for limits: %#x not in (0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07)", ErrInvalidByte, flag)
	}

	shared = flag == 0x02 || flag == 0x03 || flag == 0x06 || flag == 0x07

	return
}
