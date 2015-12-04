// Copyright (c) 2015 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package protocol

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/uber/thriftrw-go/protocol/binary"
	"github.com/uber/thriftrw-go/wire"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
)

type encodeDecodeTest struct {
	value   wire.Value
	encoded []byte
}

// Fully evaluate lazy collections inside a Value.
func evaluate(v wire.Value) error {
	switch v.Type {
	case wire.TBool:
		return nil
	case wire.TByte:
		return nil
	case wire.TDouble:
		return nil
	case wire.TI16:
		return nil
	case wire.TI32:
		return nil
	case wire.TI64:
		return nil
	case wire.TBinary:
		return nil
	case wire.TStruct:
		for _, f := range v.Struct.Fields {
			if err := evaluate(f.Value); err != nil {
				return err
			}
		}
		return nil
	case wire.TMap:
		return v.Map.Items.ForEach(func(item wire.MapItem) error {
			if err := evaluate(item.Key); err != nil {
				return err
			}
			if err := evaluate(item.Value); err != nil {
				return err
			}
			return nil
		})
	case wire.TSet:
		return v.Set.Items.ForEach(evaluate)
	case wire.TList:
		return v.List.Items.ForEach(evaluate)
	default:
		return fmt.Errorf("unknown type %s", v.Type)
	}
}

// Test for primitive encode/decode cases where assert's reflection based
// equals method suffices.
func checkEncodeDecode(t *testing.T, typ wire.Type, tests []encodeDecodeTest) {
	for _, tt := range tests {
		buffer := bytes.Buffer{}

		err := Binary.Encode(tt.value, &buffer)
		if assert.NoError(t, err, "Encode failed:\n%s", tt.value) {
			assert.Equal(t, tt.encoded, buffer.Bytes())
		}

		value, err := Binary.Decode(bytes.NewReader(tt.encoded), typ)
		if assert.NoError(t, err, "Decode failed:\n%s", tt.value) {
			assert.Equal(
				t,
				tt.value,
				value,
				"\n"+strings.Join(pretty.Diff(tt.value, value), "\n"),
			)
		}
	}
}

// Test for encode/decode cases where values must first be normalized using
// ToPrimitive.
func checkEncodeDecodeToPrimitive(t *testing.T, typ wire.Type, tests []encodeDecodeTest) {
	for _, tt := range tests {
		buffer := bytes.Buffer{}

		err := Binary.Encode(tt.value, &buffer)
		if assert.NoError(t, err, "Encode failed:\n%s", tt.value) {
			assert.Equal(t, tt.encoded, buffer.Bytes())
		}

		value, err := Binary.Decode(bytes.NewReader(tt.encoded), typ)
		if assert.NoError(t, err, "Decode failed:\n%s", tt.value) {
			assert.Equal(
				t,
				wire.ToPrimitive(tt.value),
				wire.ToPrimitive(value),
				"\n"+strings.Join(pretty.Diff(tt.value, value), "\n"),
			)
		}

		// encode the decoded Value again
		buffer = bytes.Buffer{}
		err = Binary.Encode(value, &buffer)
		if assert.NoError(t, err, "Encode of decoded value failed:\n%s", tt.value) {
			assert.Equal(t, tt.encoded, buffer.Bytes())
		}
	}
}

type decodeFailureTest struct {
	data []byte
}

func checkDecodeFailure(t *testing.T, typ wire.Type, tests []decodeFailureTest) {
	for _, tt := range tests {
		value, err := Binary.Decode(bytes.NewReader(tt.data), typ)
		if err == nil {
			// lazy collections need to be fully evaluated for the failure to
			// propagate
			err = evaluate(value)
		}
		if assert.Error(t, err, "Expected failure parsing %#v, got %s", tt.data, value) {
			assert.True(
				t,
				binary.IsDecodeError(err),
				"Expected decode error while parsing %#v, got %s",
				tt.data,
				err,
			)
		}
	}
}

func TestBool(t *testing.T) {
	tests := []encodeDecodeTest{
		{vbool(false), []byte{0x00}},
		{vbool(true), []byte{0x01}},
	}

	checkEncodeDecode(t, wire.TBool, tests)
}

func TestBoolDecodeFailure(t *testing.T) {
	tests := []decodeFailureTest{
		{data: []byte{0x02}}, // values outside 0 and 1
	}

	checkDecodeFailure(t, wire.TBool, tests)
}

func TestByte(t *testing.T) {
	tests := []encodeDecodeTest{
		{vbyte(0), []byte{0x00}},
		{vbyte(1), []byte{0x01}},
		{vbyte(-1), []byte{0xff}},
		{vbyte(127), []byte{0x7f}},
		{vbyte(-128), []byte{0x80}},
	}

	checkEncodeDecode(t, wire.TByte, tests)
}

func TestI16(t *testing.T) {
	tests := []encodeDecodeTest{
		{vi16(1), []byte{0x00, 0x01}},
		{vi16(255), []byte{0x00, 0xff}},
		{vi16(256), []byte{0x01, 0x00}},
		{vi16(257), []byte{0x01, 0x01}},
		{vi16(32767), []byte{0x7f, 0xff}},
		{vi16(-1), []byte{0xff, 0xff}},
		{vi16(-2), []byte{0xff, 0xfe}},
		{vi16(-256), []byte{0xff, 0x00}},
		{vi16(-255), []byte{0xff, 0x01}},
		{vi16(-32768), []byte{0x80, 0x00}},
	}

	checkEncodeDecode(t, wire.TI16, tests)
}

func TestI32(t *testing.T) {
	tests := []encodeDecodeTest{
		{vi32(1), []byte{0x00, 0x00, 0x00, 0x01}},
		{vi32(255), []byte{0x00, 0x00, 0x00, 0xff}},
		{vi32(65535), []byte{0x00, 0x00, 0xff, 0xff}},
		{vi32(16777215), []byte{0x00, 0xff, 0xff, 0xff}},
		{vi32(2147483647), []byte{0x7f, 0xff, 0xff, 0xff}},
		{vi32(-1), []byte{0xff, 0xff, 0xff, 0xff}},
		{vi32(-256), []byte{0xff, 0xff, 0xff, 0x00}},
		{vi32(-65536), []byte{0xff, 0xff, 0x00, 0x00}},
		{vi32(-16777216), []byte{0xff, 0x00, 0x00, 0x00}},
		{vi32(-2147483648), []byte{0x80, 0x00, 0x00, 0x00}},
	}

	checkEncodeDecode(t, wire.TI32, tests)
}

func TestI64(t *testing.T) {
	tests := []encodeDecodeTest{
		{vi64(1), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}},
		{vi64(4294967295), []byte{0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}},
		{vi64(1099511627775), []byte{0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{vi64(281474976710655), []byte{0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{vi64(72057594037927935), []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{vi64(9223372036854775807), []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{vi64(-1), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{vi64(-4294967296), []byte{0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00}},
		{vi64(-1099511627776), []byte{0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{vi64(-281474976710656), []byte{0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{vi64(-72057594037927936), []byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{vi64(-9223372036854775808), []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
	}

	checkEncodeDecode(t, wire.TI64, tests)
}

func TestDouble(t *testing.T) {
	tests := []encodeDecodeTest{
		{vdouble(0.0), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{vdouble(1.0), []byte{0x3f, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{vdouble(1.0000000001), []byte{0x3f, 0xf0, 0x0, 0x0, 0x0, 0x6, 0xdf, 0x38}},
		{vdouble(1.1), []byte{0x3f, 0xf1, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9a}},
		{vdouble(-1.1), []byte{0xbf, 0xf1, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9a}},
		{vdouble(3.141592653589793), []byte{0x40, 0x9, 0x21, 0xfb, 0x54, 0x44, 0x2d, 0x18}},
		{vdouble(-1.0000000001), []byte{0xbf, 0xf0, 0x0, 0x0, 0x0, 0x6, 0xdf, 0x38}},
		{vdouble(math.Inf(0)), []byte{0x7f, 0xf0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}},
		{vdouble(math.Inf(-1)), []byte{0xff, 0xf0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}},
	}

	checkEncodeDecode(t, wire.TDouble, tests)
}

func TestDoubleNaN(t *testing.T) {
	value := vdouble(math.NaN())
	encoded := []byte{0x7f, 0xf8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}

	buffer := bytes.Buffer{}
	err := Binary.Encode(value, &buffer)
	if assert.NoError(t, err, "Encode failed:\n%s", value) {
		assert.Equal(t, encoded, buffer.Bytes())
	}

	v, err := Binary.Decode(bytes.NewReader(encoded), wire.TDouble)
	if assert.NoError(t, err, "Decode failed:\n%s", value) {
		assert.Equal(t, wire.TDouble, v.Type)
		assert.True(t, math.IsNaN(v.Double))
	}
}

func TestBinary(t *testing.T) {
	tests := []encodeDecodeTest{
		{vbinary(""), []byte{0x00, 0x00, 0x00, 0x00}},
		{vbinary("hello"), []byte{
			0x00, 0x00, 0x00, 0x05, // len:4 = 5
			0x68, 0x65, 0x6c, 0x6c, 0x6f, // 'h', 'e', 'l', 'l', 'o'
		}},
	}

	checkEncodeDecode(t, wire.TBinary, tests)
}

func TestBinaryDecodeFailure(t *testing.T) {
	tests := []decodeFailureTest{
		{data: []byte{0xff, 0x30, 0x30, 0x30}}, // negative length
	}

	checkDecodeFailure(t, wire.TBinary, tests)
}

func TestStruct(t *testing.T) {
	tests := []encodeDecodeTest{
		{vstruct(), []byte{0x00}},
		{vstruct(vfield(1, vbool(true))), []byte{
			0x02,       // type:1 = bool
			0x00, 0x01, // id:2 = 1
			0x01, // value = true
			0x00, // stop
		}},
		{
			vstruct(
				vfield(1, vi16(42)),
				vfield(2, vlist(wire.TBinary, vbinary("foo"), vbinary("bar"))),
				vfield(3, vset(wire.TBinary, vbinary("baz"), vbinary("qux"))),
			), []byte{
				0x06,       // type:1 = i16
				0x00, 0x01, // id:2 = 1
				0x00, 0x2a, // value = 42

				0x0F,       // type:1 = list
				0x00, 0x02, // id:2 = 2

				// <list>
				0x0B,                   // type:1 = binary
				0x00, 0x00, 0x00, 0x02, // size:4 = 2
				// <binary>
				0x00, 0x00, 0x00, 0x03, // len:4 = 3
				0x66, 0x6f, 0x6f, // 'f', 'o', 'o'
				// </binary>
				// <binary>
				0x00, 0x00, 0x00, 0x03, // len:4 = 3
				0x62, 0x61, 0x72, // 'b', 'a', 'r'
				// </binary>
				// </list>

				0x0E,       // type = set
				0x00, 0x03, // id = 3

				// <set>
				0x0B,                   // type:1 = binary
				0x00, 0x00, 0x00, 0x02, // size:4 = 2
				// <binary>
				0x00, 0x00, 0x00, 0x03, // len:4 = 3
				0x62, 0x61, 0x7a, // 'b', 'a', 'z'
				// </binary>
				// <binary>
				0x00, 0x00, 0x00, 0x03, // len:4 = 3
				0x71, 0x75, 0x78, // 'q', 'u', 'x'
				// </binary>
				// </set>

				0x00, // stop
			},
		},
	}

	checkEncodeDecodeToPrimitive(t, wire.TStruct, tests)
}

func TestMap(t *testing.T) {
	tests := []encodeDecodeTest{
		{vmap(wire.TI64, wire.TBinary), []byte{0x0A, 0x0B, 0x00, 0x00, 0x00, 0x00}},
		{
			vmap(
				wire.TBinary, wire.TList,
				vitem(vbinary("a"), vlist(wire.TI16, vi16(1))),
				vitem(vbinary("b"), vlist(wire.TI16, vi16(2), vi16(3))),
			), []byte{
				0x0B,                   // ktype = binary
				0x0F,                   // vtype = list
				0x00, 0x00, 0x00, 0x02, // count:4 = 2

				// <item>
				// <key>
				0x00, 0x00, 0x00, 0x01, // len:4 = 1
				0x61, // 'a'
				// </key>
				// <value>
				0x06,                   // type:1 = i16
				0x00, 0x00, 0x00, 0x01, // count:4 = 1
				0x00, 0x01, // 1
				// </value>
				// </item>

				// <item>
				// <key>
				0x00, 0x00, 0x00, 0x01, // len:4 = 1
				0x62, // 'b'
				// </key>
				// <value>
				0x06,                   // type:1 = i16
				0x00, 0x00, 0x00, 0x02, // count:4 = 2
				0x00, 0x02, // 2
				0x00, 0x03, // 3
				// </value>
				// </item>
			},
		},
	}

	checkEncodeDecodeToPrimitive(t, wire.TMap, tests)
}

func TestMapDecodeFailure(t *testing.T) {
	tests := []decodeFailureTest{
		{data: []byte{
			0x08, 0x0B, // key: i32, value: binary
			0xff, 0x00, 0x00, 0x30, // negative length
		}},
	}

	checkDecodeFailure(t, wire.TMap, tests)
}

func TestSet(t *testing.T) {
	tests := []encodeDecodeTest{
		{vset(wire.TBool), []byte{0x02, 0x00, 0x00, 0x00, 0x00}},
		{
			vset(wire.TBool, vbool(true), vbool(false), vbool(true)),
			[]byte{0x02, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x01},
		},
	}

	checkEncodeDecodeToPrimitive(t, wire.TSet, tests)
}

func TestSetDecodeFailure(t *testing.T) {
	tests := []decodeFailureTest{
		{data: []byte{
			0x08,                   // type: i32
			0xff, 0x00, 0x30, 0x30, // negative length
		}},
	}

	checkDecodeFailure(t, wire.TSet, tests)
}

func TestList(t *testing.T) {
	tests := []encodeDecodeTest{
		{vlist(wire.TStruct), []byte{0x0C, 0x00, 0x00, 0x00, 0x00}},
		{
			vlist(
				wire.TStruct,
				vstruct(
					vfield(1, vi16(1)),
					vfield(2, vi32(2)),
				),
				vstruct(
					vfield(1, vi16(3)),
					vfield(2, vi32(4)),
				),
			),
			[]byte{
				0x0C,                   // vtype:1 = struct
				0x00, 0x00, 0x00, 0x02, // count:4 = 2

				// <struct>
				0x06,       // type:1 = i16
				0x00, 0x01, // id:2 = 1
				0x00, 0x01, // value = 1

				0x08,       // type:1 = i32
				0x00, 0x02, // id:2 = 2
				0x00, 0x00, 0x00, 0x02, // value = 2

				0x00, // stop
				// </struct>

				// <struct>
				0x06,       // type:1 = i16
				0x00, 0x01, // id:2 = 1
				0x00, 0x03, // value = 3

				0x08,       // type:1 = i32
				0x00, 0x02, // id:2 = 2
				0x00, 0x00, 0x00, 0x04, // value = 4

				0x00, // stop
				// </struct>
			},
		},
	}

	checkEncodeDecodeToPrimitive(t, wire.TList, tests)
}

func TestListDecodeFailure(t *testing.T) {
	tests := []decodeFailureTest{
		{data: []byte{
			0x0B,                   // type: i32
			0xff, 0x00, 0x30, 0x00, // negative length
		}},
		{data: []byte{
			0x02, // type: bool
			0x00, 0x00, 0x00, 0x01,
			0x10, // invalid bool
		}},
	}

	checkDecodeFailure(t, wire.TList, tests)
}

func TestStructOfContainers(t *testing.T) {
	tests := []encodeDecodeTest{
		{
			vstruct(
				vfield(1, vlist(
					wire.TMap,
					vmap(
						wire.TI32, wire.TSet,
						vitem(vi32(1), vset(
							wire.TBinary,
							vbinary("a"), vbinary("b"), vbinary("c"),
						)),
						vitem(vi32(2), vset(wire.TBinary)),
						vitem(vi32(3), vset(
							wire.TBinary,
							vbinary("d"), vbinary("e"), vbinary("f"),
						)),
					),
					vmap(
						wire.TI32, wire.TSet,
						vitem(vi32(4), vset(wire.TBinary, vbinary("g"))),
					),
				)),
				vfield(2, vlist(wire.TI16, vi16(1), vi16(2), vi16(3))),
			),
			[]byte{
				0x0f,       // type:list
				0x00, 0x01, // field ID 1

				0x0d,                   // type: map
				0x00, 0x00, 0x00, 0x02, // length: 2

				// <map-1>
				0x08, 0x0e, // ktype: i32, vtype: set
				0x00, 0x00, 0x00, 0x03, // length: 3

				// 1: {"a", "b", "c"}
				0x00, 0x00, 0x00, 0x01, // 1
				0x0B,                   // type: binary
				0x00, 0x00, 0x00, 0x03, // length: 3
				0x00, 0x00, 0x00, 0x01, 0x61, // 'a'
				0x00, 0x00, 0x00, 0x01, 0x62, // 'b'
				0x00, 0x00, 0x00, 0x01, 0x63, // 'c'

				// 2: {}
				0x00, 0x00, 0x00, 0x02, // 2
				0x0B,                   // type: binary
				0x00, 0x00, 0x00, 0x00, // length: 0

				// 3: {"d", "e", "f"}
				0x00, 0x00, 0x00, 0x03, // 3
				0x0B,                   // type: binary
				0x00, 0x00, 0x00, 0x03, // length: 3
				0x00, 0x00, 0x00, 0x01, 0x64, // 'd'
				0x00, 0x00, 0x00, 0x01, 0x65, // 'e'
				0x00, 0x00, 0x00, 0x01, 0x66, // 'f'

				// </map-1>

				// <map-2>
				0x08, 0x0e, // ktype: i32, vtype: set
				0x00, 0x00, 0x00, 0x01, // length: 1

				// 4: {"g"}
				0x00, 0x00, 0x00, 0x04, // 3
				0x0B,                   // type: binary
				0x00, 0x00, 0x00, 0x01, // length: 1
				0x00, 0x00, 0x00, 0x01, 0x67, // 'g'

				// </map-2>

				0x0f,       // type: list
				0x00, 0x02, // field ID 2

				0x06,                   // type: i16
				0x00, 0x00, 0x00, 0x03, // length 3
				0x00, 0x01, 0x00, 0x02, 0x00, 0x03, // [1,2,3]

				0x00,
			},
		},
	}

	checkEncodeDecodeToPrimitive(t, wire.TStruct, tests)
}

// TODO test input too short errors
