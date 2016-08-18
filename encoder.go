/*
 *
 *  * Copyright 2012-2016 Viant.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 *  * use this file except in compliance with the License. You may obtain a copy of
 *  * the License at
 *  *
 *  * http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 *  * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 *  * License for the specific language governing permissions and limitations under
 *  * the License.
 *
 */
 
package hessian

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
)

type encoder struct {
	writer     io.Writer
	clsDefList []ClassDef
	nameMap    map[string]string
}

func NewEncoder(w io.Writer, np map[string]string) *encoder {
	if w == nil {
		return nil
	}
	if np == nil {
		np = make(map[string]string, 17)
	}
	encoder := &encoder{w, make([]ClassDef, 0, 17), np}
	return encoder
}

func (e *encoder) RegisterNameType(key string, javaClsName string) {
	e.nameMap[key] = javaClsName
}

func (e *encoder) RegisterNameMap(mp map[string]string) {
	e.nameMap = mp
}

func (e *encoder) Reset() {
	e.nameMap = make(map[string]string, 17)
	e.clsDefList = make([]ClassDef, 0, 17)
}

func (e *encoder) WriteObject(data interface{}) (int, error) {
	if data == nil {
		io.WriteString(e.writer, "N")
		return 1, nil
	}
	typ := reflect.TypeOf(data)

	switch typ.Kind() {
	case reflect.String:
		value := data.(string)
		return e.writeString(value)
	case reflect.Int32:
		value := data.(int32)
		return e.writeInt(value)
	case reflect.Int64:
		value := data.(int64)
		return e.writeLong(value)
	case reflect.Slice, reflect.Array:
		return e.writeList(data)
	case reflect.Float32:
		value := data.(float32)
		return e.writeDouble(float64(value))
	case reflect.Float64:
		value := data.(float64)
		return e.writeDouble(value)
	//case reflect.Uint8:
	case reflect.Map:
		return e.writeMap(data)
	case reflect.Bool:
		value := data.(bool)
		return e.writeBoolean(value)
	case reflect.Struct:
		return e.writeInstance(data)
	}
	fmt.Println("error WriteObject", data, "kind", typ.Kind(), typ)
	return 0, nil
}

func (e *encoder) writeDouble(value float64) (int, error) {
	v := float64(int64(value))
	if v == value {
		iv := int64(value)
		switch iv {
		case 0:
			return e.writeBT(BC_DOUBLE_ZERO)
		case 1:
			return e.writeBT(BC_DOUBLE_ONE)
		}
		if iv >= -0x80 && iv < 0x80 {
			e.writeBT(BC_DOUBLE_BYTE, byte(iv))
		} else if iv >= -0x8000 && iv < 0x8000 {
			e.writeBT(BC_DOUBLE_BYTE, byte(iv>>8), byte(iv))
		}
	} else {
		bits := uint64(math.Float64bits(value))
		e.writeBT(BC_DOUBLE)
		e.writeBT(byte(bits>>56), byte(bits>>48), byte(bits>>40), byte(bits>>32), byte(bits>>24),
			byte(bits>>16), byte(bits>>8), byte(bits))
	}
	return 8, nil

}

func (e *encoder) writeMap(data interface{}) (int, error) {
	e.writeBT(BC_MAP_UNTYPED)
	vv := reflect.ValueOf(data)
	typ := reflect.TypeOf(data).Key()
	keys := vv.MapKeys()
	for i := 0; i < len(keys); i++ {
		k := buildKey(keys[i], typ)
		e.WriteObject(k)
		//e.WriteObject(keys[i].Interface())
		//v := buildValue(vv.MapIndex(keys[i]))
		//e.WriteObject(v)
		e.WriteObject(vv.MapIndex(keys[i]).Interface())
	}
	e.writeBT(BC_END)
	return len(keys), nil
}

func buildKey(key reflect.Value, typ reflect.Type) interface{} {
	switch typ.Kind() {
	case reflect.String:
		return key.String()
	case reflect.Bool:
		return key.Bool()
	case reflect.Int:
		return int32(key.Int())
	case reflect.Int8:
		return int8(key.Int())
	case reflect.Int16:
	case reflect.Int32:
		return int32(key.Int())
	case reflect.Int64:
		return key.Int()
	case reflect.Uint8:
		return byte(key.Uint())
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return key.Uint()
	}
	return newCodecError("unsuport key kind " + typ.Kind().String())
}

func (e *encoder) writeString(value string) (int, error) {
	dataBys := []byte(value)
	l := len(dataBys)
	sub := 0x8000
	begin := 0
	for l > sub {
		buf := make([]byte, 3)
		buf[0] = BC_STRING_CHUNK
		buf[1] = byte(sub >> 8)
		buf[2] = byte(sub)
		_, err := e.writer.Write(buf)
		if err != nil {
			return 0, newCodecError("writeString", err)
		}
		buf = make([]byte, sub)
		copy(buf, dataBys[begin:begin+sub])
		_, err = e.writer.Write(buf)
		if err != nil {
			return 0, newCodecError("writeString", err)
		}
		l -= sub
		begin += sub
	}
	var buf []byte
	if l == 0 {
		return len(dataBys), nil
	} else if l <= int(STRING_DIRECT_MAX) {
		buf = make([]byte, 1)
		buf[0] = byte(l + int(BC_STRING_DIRECT))
	} else if l <= int(STRING_SHORT_MAX) {
		buf = make([]byte, 2)
		buf[0] = byte((l >> 8) + int(BC_STRING_SHORT))
		buf[1] = byte(l)
	} else {
		buf = make([]byte, 3)
		buf[0] = BC_STRING
		buf[0] = byte(l >> 8)
		buf[1] = byte(l)
	}
	bs := make([]byte, l+len(buf))
	copy(bs[0:], buf)
	copy(bs[len(buf):], dataBys[begin:])
	_, err := e.writer.Write(bs)
	if err != nil {
		return 0, newCodecError("writeString", err)
	}
	return l, nil
}

func (e *encoder) writeList(data interface{}) (int, error) {

	vv := reflect.ValueOf(data)
	e.writeBT(BC_LIST_FIXED_UNTYPED)
	e.writeInt(int32(vv.Len()))
	fmt.Println("list", vv.Len(), data)
	for i := 0; i < vv.Len(); i++ {
		e.WriteObject(vv.Index(i).Interface())
	}
	e.writeBT(BC_END)
	return vv.Len(), nil
}

func (e *encoder) writeInt(value int32) (int, error) {
	var buf []byte
	if int32(INT_DIRECT_MIN) <= value && value <= int32(INT_DIRECT_MAX) {
		buf = make([]byte, 1)
		buf[0] = byte(value + int32(BC_INT_ZERO))
	} else if int32(INT_BYTE_MIN) <= value && value <= int32(INT_BYTE_MAX) {
		buf = make([]byte, 2)
		buf[0] = byte(int32(BC_INT_BYTE_ZERO) + value>>8)
		buf[1] = byte(value)
	} else if int32(INT_SHORT_MIN) <= value && value <= int32(INT_SHORT_MAX) {
		buf = make([]byte, 3)
		buf[0] = byte(value>>16 + int32(BC_INT_SHORT_ZERO))
		buf[1] = byte(value >> 8)
		buf[2] = byte(value)
	} else {
		buf = make([]byte, 5)
		buf[0] = byte('I')
		buf[1] = byte(value >> 24)
		buf[2] = byte(value >> 16)
		buf[3] = byte(value >> 8)
		buf[4] = byte(value)
	}
	l, err := e.writer.Write(buf)
	if err != nil {
		return 0, newCodecError("WriteInt", err)
	}
	return l, nil
}

func (e *encoder) writeLong(value int64) (int, error) {
	var buf []byte
	if int64(LONG_DIRECT_MIN) <= value && value <= int64(LONG_DIRECT_MAX) {
		buf = make([]byte, 1)
		buf[0] = byte(value - int64(BC_LONG_ZERO))
	} else if int64(LONG_BYTE_MIN) <= value && value <= int64(LONG_BYTE_MAX) {
		buf = make([]byte, 2)
		buf[0] = byte(int64(BC_LONG_BYTE_ZERO) + (value >> 8))
		buf[1] = byte(value)
	} else if int64(LONG_SHORT_MIN) <= value && value <= int64(LONG_SHORT_MAX) {
		buf = make([]byte, 3)
		buf[0] = byte(int64(BC_LONG_SHORT_ZERO) + (value >> 16))
		buf[1] = byte(value >> 8)
		buf[2] = byte(value)
	} else if 0x80000000 <= value && value <= 0x7fffffff {
		buf = make([]byte, 5)
		buf[0] = BC_LONG_INT
		buf[1] = byte(value >> 24)
		buf[2] = byte(value >> 16)
		buf[3] = byte(value >> 8)
		buf[4] = byte(value)
	} else {
		buf = make([]byte, 9)
		buf[0] = 'L'
		buf[1] = byte(value >> 56)
		buf[2] = byte(value >> 48)
		buf[3] = byte(value >> 32)
		buf[4] = byte(value >> 24)
		buf[5] = byte(value >> 16)
		buf[6] = byte(value >> 8)
		buf[7] = byte(value)
	}
	l, err := e.writer.Write(buf)
	if err != nil {
		return 0, newCodecError("WriteLong", err)
	}
	return l, nil
}

func (e *encoder) writeBoolean(value bool) (int, error) {
	buf := make([]byte, 1)
	if value {
		buf[0] = BC_TRUE
	} else {
		buf[0] = BC_FALSE
	}
	l, err := e.writer.Write(buf)
	if err != nil {
		return 0, newCodecError("WriteBoolean", err)
	}
	return l, nil
}

func (e *encoder) writeBytes(value []byte) (int, error) {
	sub := CHUNK_SIZE
	l := len(value)
	begin := 0

	for l > sub {
		buf := make([]byte, 3+CHUNK_SIZE)
		buf[0] = byte(BC_BINARY_CHUNK)
		buf[1] = byte(sub >> 8)
		buf[2] = byte(sub)

		copy(buf[3:], value[begin:begin+CHUNK_SIZE])
		_, err := e.writer.Write(buf)
		if err != nil {
			return 0, newCodecError("WriteBytes", err)
		}
		l -= CHUNK_SIZE
		begin += CHUNK_SIZE
	}
	var buf []byte
	if l == 0 {
		return len(value), nil
	} else if l <= int(BINARY_DIRECT_MAX) {
		buf := make([]byte, 1)
		buf[0] = byte(int(BC_BINARY_DIRECT) + l)
	} else if l <= int(BINARY_SHORT_MAX) {
		buf := make([]byte, 2)
		buf[0] = byte(int(BC_BINARY_SHORT) + l>>8)
		buf[1] = byte(l)
	} else {
		buf := make([]byte, 3)
		buf[0] = byte(BC_BINARY)
		buf[1] = byte(l >> 8)
		buf[2] = byte(l)
	}
	bs := make([]byte, l+len(buf))
	copy(bs[0:], buf)
	copy(bs[len(buf):], value[begin:])
	_, err := e.writer.Write(bs)
	if err != nil {
		return 0, newCodecError("WriteBytes", err)
	}
	return len(value), nil
}

func (e *encoder) writeBT(bs ...byte) (int, error) {
	return e.writer.Write(bs)
}

func (e *encoder) writeInstance(data interface{}) (int, error) {
	fmt.Println("struct", data)
	typ := reflect.TypeOf(data)
	vv := reflect.ValueOf(data)

	clsName, ok := e.nameMap[typ.Name()]
	if !ok {
		clsName = typ.Name()
		e.nameMap[clsName] = clsName
	}
	//fmt.Println("ok",ok, clsName)
	l, ok := e.existClassDef(clsName)
	//fmt.Println("ok 2 ->",ok, l)
	if !ok {
		l, _ = e.writeClsDef(typ, clsName)
	}
	if byte(l) <= OBJECT_DIRECT_MAX {
		e.writeBT(byte(l) + BC_OBJECT_DIRECT)
	} else {
		e.writeBT(BC_OBJECT)
		e.writeInt(int32(l))
	}
	for i := 0; i < vv.NumField(); i++ {
		//if field.Field(i).Type().PkgPath() != "" {
		//	continue
		//}
		e.writeField(vv.Field(i), vv.Field(i).Type().Kind())
	}
	return vv.NumField(), nil
}

func (e *encoder) writeField(field reflect.Value, kind reflect.Kind) error {
	switch kind {
	case reflect.Struct:
		//fmt.Println("StructFiled", field.Interface(), "num", field.NumField())
		e.WriteObject(field.Interface())
	case reflect.String:
		v := field.String()
		e.writeString(v)
	case reflect.Int32:
		v := int32(field.Int())
		e.writeInt(v)
	case reflect.Int64:
		v := int64(field.Int())
		e.writeLong(v)
	case reflect.Bool:
		v := field.Bool()
		e.writeBoolean(v)
	case reflect.Map:
		v := field.Interface()
		//fmt.Println("Map", field.Type().Kind(), field.Interface())
		e.writeMap(v)
	case reflect.Array, reflect.Slice:
		v := field.Interface()
		//fmt.Println("Slice", field.Type().Kind(), field.Interface())
		e.WriteObject(v)
	case reflect.Int:
		v := int32(field.Int())
		e.writeInt(v)
	}
	return newCodecError("writeField unspport kind " + kind.String())
}

func (e *encoder) writeClsDef(typ reflect.Type, clsName string) (int, error) {
	e.writeBT(BC_OBJECT_DEF)
	e.writeString(clsName)
	fldList := make([]string, typ.NumField())
	e.writeInt(int32(len(fldList)))
	for i := 0; i < len(fldList); i++ {
		str, _ := lowerName(typ.Field(i).Name)
		fldList[i] = str
		e.writeString(fldList[i])
	}
	clsDef := ClassDef{clsName, fldList}
	l := len(e.clsDefList)
	e.clsDefList = append(e.clsDefList, clsDef)
	return l, nil
}

func lowerName(name string) (string, error) {
	if name[0] >= 'a' && name[0] <= 'c' {
		return name, nil
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		bs := make([]byte, len(name))
		bs[0] = byte(name[0] + ASCII_GAP)
		copy(bs[1:], name[1:])
		return string(bs), nil
	} else {
		return name, nil
	}

}

func (e *encoder) existClassDef(clsName string) (int, bool) {
	for i := 0; i < len(e.clsDefList); i++ {
		if strings.Compare(clsName, e.clsDefList[i].FullClassName) == 0 {
			return i, true
		}
	}
	return 0, false
}
