package binaryencoding

import (
	"github.com/tetratelabs/wazero/internal/leb128"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// EncodeFunctionType returns the wasm.FunctionType encoded in WebAssembly binary
// format. Plain function types use the legacy 0x60 shorthand; struct (0x5F) and
// array (0x5E) shorthands serialise GC composite types. If SuperTypeIndex is
// non-nil, the entry is wrapped in a sub or sub-final form.
func EncodeFunctionType(t *wasm.FunctionType) []byte {
	var data []byte
	if t.SuperTypeIndex != nil {
		if t.Final {
			data = append(data, 0x4F) // sub final
		} else {
			data = append(data, 0x50) // sub
		}
		data = append(data, 1) // one supertype
		data = append(data, leb128.EncodeUint32(*t.SuperTypeIndex)...)
	}
	switch t.Form {
	case wasm.CompositeFormStruct:
		data = append(data, 0x5F)
		data = append(data, leb128.EncodeUint32(uint32(len(t.Fields)))...)
		for i := range t.Fields {
			data = append(data, encodeFieldType(t.Fields[i])...)
		}
	case wasm.CompositeFormArray:
		data = append(data, 0x5E)
		data = append(data, encodeFieldType(t.ArrayField)...)
	default: // CompositeFormFunc
		data = append(data, 0x60)
		data = append(data, EncodeValTypes(t.Params)...)
		data = append(data, EncodeValTypes(t.Results)...)
	}
	return data
}

// EncodeTypeSection serialises a slice of FunctionType, grouping consecutive
// entries whose RecGroupSize > 1 (with matching RecGroupPosition) into a
// rec-group prefix (0x4E).
func EncodeTypeSection(types []wasm.FunctionType) []byte {
	var out []byte
	groupCount := uint32(0)
	for i := 0; i < len(types); {
		t := &types[i]
		if t.RecGroupSize > 1 && t.RecGroupPosition == 0 {
			out = append(out, 0x4E)
			out = append(out, leb128.EncodeUint32(uint32(t.RecGroupSize))...)
			for j := 0; j < t.RecGroupSize && i+j < len(types); j++ {
				out = append(out, EncodeFunctionType(&types[i+j])...)
			}
			i += t.RecGroupSize
			groupCount++
		} else {
			out = append(out, EncodeFunctionType(t)...)
			i++
			groupCount++
		}
	}
	return append(leb128.EncodeUint32(groupCount), out...)
}

func encodeFieldType(f wasm.FieldType) []byte {
	var b []byte
	if f.Packed != wasm.PackedTypeNone {
		switch f.Packed {
		case wasm.PackedTypeI8:
			b = append(b, wasm.PackedTypeI8Byte)
		case wasm.PackedTypeI16:
			b = append(b, wasm.PackedTypeI16Byte)
		}
	} else {
		b = append(b, f.ValueType.Kind())
	}
	if f.Mutable {
		b = append(b, 0x01)
	} else {
		b = append(b, 0x00)
	}
	return b
}
