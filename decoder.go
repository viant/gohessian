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

/*
decoder implement hessian 2 protocol, It follows java hessian package standard.
It assume that you using the java name convention
baisca difference between java and go
fully qualify java class name is composed of package + class name
Go assume upper case of field name is exportable and java did not have that constrain
but in general java using camo camlecase. So it did conversion of field name from
the first letter of from upper to lower case
typMap{string]reflect.Type contain full java package+class name and go relfect.Type
must provide in order to correctly decode to galang interface


*/
package hessian

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"reflect"
	"strings"
)

var _ = bytes.MinRead
var _ = reflect.Value{}

// ErrDecoder is returned when the encoder encounters an error.
type ErrDecoder struct {
	Message string
	Err     error
}

type ClassDef struct {
	FullClassName string
	FieldName     []string
}

type decoder struct {
	reader     io.Reader
	typMap     map[string]reflect.Type
	typList    []string
	refList    []interface{}
	clsDefList []ClassDef
}

func NewDecoder(r io.Reader, typ map[string]reflect.Type) *decoder {
	if typ == nil {
		typ = make(map[string]reflect.Type, 17)
	}
	decode := &decoder{r, typ, make([]string, 0, 17), make([]interface{}, 0, 17), make([]ClassDef, 0, 17)}
	return decode
}

func (d *decoder) RegisterType(key string, value reflect.Type) {
	d.typMap[key] = value
}

func (d *decoder) RegisterTypeMap(mp map[string]reflect.Type) {
	d.typMap = mp
}

func (d *decoder) RegisterVal(key string, val interface{}) {
	d.typMap[key] = reflect.TypeOf(val)
}

func (d *decoder) Reset() {
	d.typMap = make(map[string]reflect.Type, 17)
	d.clsDefList = make([]ClassDef, 0, 17)
	d.refList = make([]interface{}, 17)
}

func (e ErrDecoder) Error() string {
	if e.Err == nil {
		return "cannot decode " + e.Message
	}
	return "cannot decode " + e.Message + ": " + e.Err.Error()
}

func newCodecError(dataType string, a ...interface{}) *ErrDecoder {
	var err error
	var format, message string
	var ok bool
	if len(a) == 0 {
		return &ErrDecoder{dataType + ": no reason given", nil}
	}
	// if last item is error: save it
	if err, ok = a[len(a)-1].(error); ok {
		a = a[:len(a)-1] // pop it
	}
	// if items left, first ought to be format string
	if len(a) > 0 {
		if format, ok = a[0].(string); ok {
			a = a[1:] // unshift
			message = fmt.Sprintf(format, a...)
		}
	}
	if message != "" {
		message = ": " + message
	}
	return &ErrDecoder{dataType + message, err}
}

func (d *decoder) readBufByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(d.reader, buf)
	if err != nil {
		return 0, newCodecError("readBufByte", err)
	}
	return buf[0], nil
}

func (d *decoder) readBuf(s int) ([]byte, error) {
	buf := make([]byte, s)
	_, err := io.ReadFull(d.reader, buf)
	if err != nil {
		return nil, newCodecError("readBuf", err)
	}
	return buf, nil
}

//name is option, if it is nil, use type.Name()
func (d *decoder) ReadObjectWithType(typ reflect.Type, name string) (interface{}, error) {
	//register the type if it did exist
	if _, ok := d.typMap[name]; ok {
		log.Println("over write existing type")
	}
	d.typMap[name] = typ
	return d.ReadObject()
}

func (d *decoder) readInt(flag int32) (interface{}, error) {
	var tag byte
	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readBufByte()
	}

	switch {
	//direct integer
	case tag >= 0x80 && tag <= 0xbf:
		return int32(tag - BC_INT_ZERO), nil
	case tag >= 0xc0 && tag <= 0xcf:
		bf := make([]byte, 1)
		if _, err := io.ReadFull(d.reader, bf); err != nil {
			return nil, newCodecError("short integer", err)
		}
		return int32(tag-BC_INT_BYTE_ZERO)<<8 + int32(bf[0]), nil
	case tag >= 0xd0 && tag <= 0xd7:
		bf := make([]byte, 2)
		if _, err := io.ReadFull(d.reader, bf); err != nil {
			return nil, newCodecError("short integer", err)
		}
		i := int32(tag-BC_INT_SHORT_ZERO)<<16 + int32(bf[1])<<8 + int32(bf[0])
		return i, nil
	case tag == BC_INT:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(d.reader, buf); err != nil {
			return nil, newCodecError("parse int", err)
		}
		i := int32(buf[0])<<24 + int32(buf[1])<<16 + int32(buf[2])<<8 + int32(buf[3])
		return i, nil
	default:
		return nil, newCodecError("integer wrong tag ", tag)

	}
}

func (d *decoder) readLong(flag int32) (interface{}, error) {
	var tag byte
	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readBufByte()
	}

	switch {
	case tag >= 0xd8 && tag <= 0xef:
		return int64(tag - BC_LONG_ZERO), nil
	case tag >= 0xf4 && tag <= 0xff:

		bf := make([]byte, 1)
		if _, err := io.ReadFull(d.reader, bf); err != nil {
			return nil, newCodecError("short integer", err)
		}
		i := int64(tag-BC_LONG_BYTE_ZERO)<<8 + int64(bf[0])
		return i, nil
	case tag >= 0x38 && tag <= 0x3f:
		bf := make([]byte, 2)
		if _, err := io.ReadFull(d.reader, bf); err != nil {
			return nil, newCodecError("short integer", err)
		}

		i := int64(tag-BC_LONG_SHORT_ZERO)<<16 + int64(bf[1])<<8 + int64(bf[0])
		return i, nil
	case tag == BC_LONG:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(d.reader, buf); err != nil {
			return nil, newCodecError("parse long", err)
		}
		i := int64(buf[0])<<56 + int64(buf[1])<<48 + int64(buf[2]) + int64(buf[3]) +
			int64(buf[4])<<24 + int64(buf[5])<<16 + int64(buf[6])<<8 + int64(buf[7])
		return i, nil
	case tag == BC_LONG_INT: //add by CSE
		buf := make([]byte, 4)
		if _, err := io.ReadFull(d.reader, buf); err != nil {
			return nil, newCodecError("parse integer", err)
		}
		i := int32(buf[0])<<24 + int32(buf[1])<<16 + int32(buf[2])<<8 + int32(buf[3])
		return i, nil
	default:
		return nil, newCodecError("long wrong tag " + string(tag))
	}

}

func (d *decoder) readDouble(flag int32) (interface{}, error) {
	var tag byte
	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readBufByte()
	}
	switch tag {
	case BC_LONG_INT:
		return d.readInt(TAG_READ)
	case BC_DOUBLE_ZERO:
		return float64(0), nil
	case BC_DOUBLE_ONE:
		return float64(1), nil
	case BC_DOUBLE_BYTE:
		bt, _ := d.readBufByte()
		return float64(bt), nil
	case BC_DOUBLE_SHORT:
		bf, _ := d.readBuf(2)
		return float64(int(bf[0])*256 + int(bf[1])), nil
	case BC_DOUBLE_MILL:
		i, _ := d.readInt(TAG_READ)
		return float64(i.(int32)), nil
	case BC_DOUBLE:
		buf, _ := d.readBuf(8)
		bits := binary.BigEndian.Uint64(buf)
		datum := math.Float64frombits(bits)
		return datum, nil
	}
	return nil, newCodecError("parse double wrong tag " + string(tag))
}

func (d *decoder) readString(flag int32) (interface{}, error) {
	var tag byte
	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readBufByte()
	}
	last := true
	var len int32
	if (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) || (tag >= 0x30 && tag <= 0x33) || (tag == BC_STRING_CHUNK || tag == BC_STRING) {
		//fmt.Println("inside ", tag)
		if tag == BC_STRING_CHUNK {
			last = false
		} else {
			last = true
		}
		l, err := d.getStrLen(tag)
		if err != nil {
			return nil, newCodecError("getStrLen", err)
		}
		len = l
		data := make([]byte, len)
		for i := 0; ; {
			if int32(i) == len {
				if last {
					//fmt.Println("last ", last, "i", i, "len", len)
					return string(data), nil
				}

				buf := make([]byte, 1)
				_, err := io.ReadFull(d.reader, buf)

				if err != nil {
					return nil, newCodecError("byte1 integer", err)
				}
				b := buf[0]
				switch {
				case b == BC_STRING_CHUNK || b == BC_STRING:
					if b == BC_STRING_CHUNK {
						last = false
					} else {
						last = true
					}
					l, err := d.getStrLen(b)
					if err != nil {
						return nil, newCodecError("getStrLen", err)
					}
					len += l
					bs := make([]byte, 0, len)
					copy(bs, data)
					data = bs
				default:
					return nil, newCodecError("tag error ", err)
				}
			} else {
				buf := make([]byte, 1)
				_, err := io.ReadFull(d.reader, buf)
				if err != nil {
					return nil, newCodecError("byte2 integer", err)
				}
				data[i] = buf[0]
				i++
			}
		}
		return string(data), nil
	} else {
		return nil, newCodecError("byte3 integer")
	}

}

func (d *decoder) getStrLen(tag byte) (int32, error) {
	switch {
	case tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX:
		return int32(tag - 0x00), nil
	case tag >= 0x30 && tag <= 0x33:
		buf := make([]byte, 1)
		_, err := io.ReadFull(d.reader, buf)
		if err != nil {
			return -1, newCodecError("byte4 integer", err)
		}
		len := int32(tag-0x30)<<8 + int32(buf[0])
		return len, nil

	case tag == BC_STRING_CHUNK || tag == BC_STRING:
		buf := make([]byte, 1)
		_, err := io.ReadFull(d.reader, buf)
		if err != nil {
			return -1, newCodecError("byte5 integer", err)
		}
		len := int32(tag)<<8 + int32(buf[0])
		return len, nil
	default:
		return -1, newCodecError("getStrLen")
	}
}
func (d *decoder) readInstance2(cls ClassDef) (interface{}, error) {
	var tmpMemberMap = make(map[string]interface{})

	for i := 0; i < len(cls.FieldName); i++ {
		fldName := cls.FieldName[i]
		obj, err := d.ReadObject()
		if err != nil {
			fmt.Println("struct error", err)
		}
		tmpMemberMap[fldName] = obj
	}
	return tmpMemberMap, nil
}

func (d *decoder) readInstance(typ reflect.Type, cls ClassDef) (interface{}, error) {
	if typ.Kind() != reflect.Struct {
		return nil, newCodecError("wrong type expect Struct but get " + typ.String())
	}
	vv := reflect.New(typ)
	st := reflect.ValueOf(vv.Interface()).Elem()
	for i := 0; i < len(cls.FieldName); i++ {
		fldName := cls.FieldName[i]
		index, err := findField(fldName, typ)
		if err != nil {
			log.Printf("%s is not found, will ski type ->p %v", fldName, typ)
			continue
		}
		fldValue := st.Field(index)
		//fmt.Println("fld", fldName, fldValue, fldValue.Kind())
		if !fldValue.CanSet() {
			return nil, newCodecError("CanSet false for " + fldName)
		}
		kind := fldValue.Kind()
		switch {
		case kind == reflect.String:
			str, err := d.readString(TAG_READ)
			if err != nil {
				return nil, newCodecError("ReadString "+fldName, err)
			}
			fldValue.SetString(str.(string))
		case kind == reflect.Int32 || kind == reflect.Int || kind == reflect.Int16:
			i, err := d.readInt(TAG_READ)
			if err != nil {
				return nil, newCodecError("ParseInt"+fldName, err)
			}
			v := int64(i.(int32))
			fldValue.SetInt(v)
		case kind == reflect.Int64 || kind == reflect.Uint64:
			i, err := d.readLong(TAG_READ)
			if err != nil {
				return nil, newCodecError("decode error "+fldName, err)
			}
			fldValue.SetInt(i.(int64))
		case kind == reflect.Bool:
			b, err := d.ReadObject()
			if err != nil {
				return nil, newCodecError("decode error "+fldName, err)
			}
			fldValue.SetBool(b.(bool))
		case kind == reflect.Float32 || kind == reflect.Float64:
			d, err := d.readDouble(TAG_READ)
			if err != nil {
				return nil, newCodecError("decode error "+fldName, err)
			}
			fldValue.SetFloat(d.(float64))
		case kind == reflect.Struct:
			s, err := d.ReadObject()
			if err != nil {
				fmt.Println("struct error", err)
			}
			fldValue.Set(reflect.Indirect(s.(reflect.Value)))
			//fmt.Println("s with struct", s)
		case kind == reflect.Map:
			//m, _ := d.ReadObject()
			//fmt.Println("struct map", m)
			d.readMap(fldValue)
		case kind == reflect.Slice || kind == reflect.Array:
			m, _ := d.ReadObject()
			v := reflect.ValueOf(m)
			if v.Len() > 0 {
				sl := reflect.MakeSlice(fldValue.Type(), v.Len(), v.Len())
				for i := 0; i < v.Len(); i++ {
					sl.Index(i).Set(reflect.ValueOf(v.Index(i).Interface()))
				}
				fldValue.Set(sl)
			}
		}

	}
	return vv, nil
}

func (d *decoder) readMap(value reflect.Value) error {
	tag, _ := d.readBufByte()
	if tag == BC_MAP {
		d.readString(TAG_READ)
	} else if tag == BC_MAP_UNTYPED {
		//do nothing
	} else {
		return newCodecError("wrong header BC_MAP_UNTYPED")
	}
	m := reflect.MakeMap(value.Type())
	//read key and value
	for {
		key, err := d.ReadObject()
		if err != nil {
			if err == io.EOF {
				//fmt.Println("endMamp")
				break
			} else {
				return newCodecError("ReadType", err)
			}
		}
		vl, err := d.ReadObject()
		//fmt.Println(key, vl)
		m.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(vl))
	}
	value.Set(m)
	return nil
}

func (d *decoder) readSlice(value reflect.Value) (interface{}, error) {
	tag, _ := d.readBufByte()
	var i int
	if tag >= BC_LIST_DIRECT_UNTYPED && tag <= 0x7f {
		i = int(tag - BC_LIST_DIRECT_UNTYPED)
	} else {
		ii, err := d.readInt(TAG_READ)
		if err != nil {
			return nil, newCodecError("ReadType", err)
		}
		i = int(ii.(int32))
	}
	//fmt.Println("list len ", i)
	ary := reflect.MakeSlice(value.Type(), i, i)
	for j := 0; j < i; j++ {
		it, err := d.ReadObject()
		if err != nil {
			return nil, newCodecError("ReadList", err)
		}
		ary.Index(j).Set(reflect.ValueOf(it))
		//fmt.Println("j", j, "it", it)
	}
	d.readBufByte()
	//fmt.Println("endList", bt)
	value.Set(ary)
	return ary, nil
}

func CapitalizeName(name string) string {
	if name[0] >= 'A' && name[0] <= 'Z' {
		return name
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		bs := make([]byte, len(name))
		bs[0] = byte(name[0] - ASCII_GAP)
		copy(bs[1:], name[1:])
		return string(bs)
	} else {
		return name
	}

}

func findField(name string, typ reflect.Type) (int, error) {
	for i := 0; i < typ.NumField(); i++ {
		str := typ.Field(i).Name
		if strings.Compare(str, name) == 0 {
			return i, nil
		}
		str1 := CapitalizeName(name)
		if strings.Compare(str, str1) == 0 {
			return i, nil
		}
	}
	return 0, newCodecError("findField")
}

func (d *decoder) readType() (interface{}, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(d.reader, buf)
	if err != nil {
		return nil, newCodecError("reading tag", err)
	}
	tag := buf[0]
	if (tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) || (tag >= 0x30 && tag <= 0x33) || (tag == BC_STRING || tag == BC_STRING_CHUNK) {
		return d.readString(int32(tag))
	} else {
		i, err := d.readInt(TAG_READ)
		if err != nil {
			return nil, newCodecError("reading tag", err)
		}
		index := int(i.(int32))
		return d.typList[index], nil
	}
}

func (d *decoder) ReadObject() (interface{}, error) {
	tag, err := d.readBufByte()
	if err != nil {
		return nil, newCodecError("reading tag", err)
	}
	//fmt.Println("tag ", tag)
	switch {
	case tag == BC_END:
		return nil, io.EOF
	case tag == BC_NULL:
		return nil, nil
	case tag == BC_TRUE:
		return true, nil
	case tag == BC_FALSE:
		return false, nil
		//direct integer
	case tag == BC_MAP:
		_, err := d.readType()
		if err != nil {
			return nil, newCodecError("ReadType", err)
		}
		m := make(map[interface{}]interface{})
		//read key and value
		key, _ := d.ReadObject()
		value, _ := d.ReadObject()
		m[key] = value
		return m, nil
	case tag == BC_MAP_UNTYPED:
		m := make(map[string]interface{}) //这里强制只支持key为string的map类型, cse只支持字符串key
		//read key and value
		for {
			key, err := d.ReadObject()
			if err != nil {
				if err == io.EOF {
					//fmt.Println("endMamp")
					return m, nil
				} else {
					return nil, newCodecError("ReadType", err)
				}
			}
			value, err := d.ReadObject()
			m[key.(string)] = value
		}
	case tag == BC_OBJECT_DEF:
		//fmt.Println("BC_OBJECT_DEF")
		clsDef, err := d.readClassDef()
		if err != nil {
			return nil, newCodecError("byte double", err)
		}
		clsD, _ := clsDef.(ClassDef)
		//add to slice
		d.clsDefList = append(d.clsDefList, clsD)
		//fmt.Println("clsD", clsD)
		//read from refList of ClassDef
		return d.ReadObject()
	case tag == BC_OBJECT:
		//fmt.Println("BC_OBJECT ")
		i, _ := d.readInt(TAG_READ)
		idx := int(i.(int32))
		clsD := d.clsDefList[idx]
		typ, ok := d.typMap[clsD.FullClassName]
		if !ok { //add by cse
			return d.readInstance2(clsD)
			//return nil, newCodecError("undefine type for "+clsD.FullClassName, err)
		} else {
			return d.readInstance(typ, clsD)
		}
	case (tag >= 0x80 && tag <= 0xbf) || (tag >= 0xc0 && tag <= 0xcf) ||
		(tag >= 0xd0 && tag <= 0xd7) || (tag == BC_INT):
		return d.readInt(int32(tag))

	case (tag >= 0xd8 && tag <= 0xef) || (tag >= 0xf4 && tag <= 0xff) ||
		(tag >= 0x38 && tag <= 0x3f) || (tag == BC_LONG_INT) ||
		(tag == BC_LONG):
		return d.readLong(int32(tag))
	case tag == BC_DOUBLE_ZERO:
		return float64(0), nil
	case tag == BC_DOUBLE_ONE:
		return float64(1), nil
	case tag == BC_DOUBLE_BYTE:
		bf1 := make([]byte, 1)
		if _, err := io.ReadFull(d.reader, bf1); err != nil {
			return nil, newCodecError("byte double", err)
		}
		i := float64(bf1[0])
		return i, nil
	case tag == BC_DOUBLE_SHORT:
		bf1 := make([]byte, 2)
		if _, err := io.ReadFull(d.reader, bf1); err != nil {
			return nil, newCodecError("short long", err)
		}
		i := float64(bf1[0])*256 + float64(bf1[0])
		return i, nil
	case tag == BC_DOUBLE_MILL:
		t, err := d.readInt(int32(tag))
		if err == nil {
			return t, err
		} else {
			return nil, newCodecError("double mill", err)
		}
	case tag == BC_DOUBLE:
		return d.readDouble(int32(tag))
	case tag == BC_DATE:
		_, err := d.readLong(int32(tag))
		if err != nil {
			return nil, newCodecError("date", err)
		} else {
			return nil, newCodecError("not yet implementd")
		}
	case tag == BC_DATE_MINUTE:
		return nil, newCodecError("not yet implementd")
	case (tag == BC_STRING_CHUNK || tag == BC_STRING) ||
		(tag >= BC_STRING_DIRECT && tag <= STRING_DIRECT_MAX) ||
		(tag >= 0x30 && tag <= 0x33):
		return d.readString(int32(tag))
	case (tag >= 0x60 && tag <= 0x6f):
		//fmt.Println("ReadInstance")
		i := int(tag - 0x60)
		clsD := d.clsDefList[i]
		typ, ok := d.typMap[clsD.FullClassName]
		if !ok { //add by cse
			return d.readInstance2(clsD)
			//return nil, newCodecError("undefine type for "+clsD.FullClassName, err)
		} else {
			return d.readInstance(typ, clsD)
		}

	case (tag == BC_BINARY || tag == BC_BINARY_CHUNK) || (tag >= 0x20 && tag <= 0x2f):
		return d.readBinary(int32(tag))
	case (tag >= BC_LIST_DIRECT && tag <= 0x77) || (tag == BC_LIST_FIXED || tag == BC_LIST_VARIABLE):
		str, err := d.readType()
		if err != nil {
			return nil, newCodecError("ReadType", err)
		}
		var i int
		if tag >= BC_LIST_DIRECT && tag <= 0x77 {
			i = int(tag - BC_LIST_DIRECT)
		} else {
			ii, err := d.readInt(TAG_READ)
			if err != nil {
				return nil, newCodecError("ReadType", err)
			}
			i = int(ii.(int32))
		}
		ary := make([]interface{}, i)
		bl := isBuildInType(str.(string))
		if bl == false {
			for j := 0; j < i; j++ {
				it, err := d.ReadObject()
				if err != nil {
					return nil, newCodecError("ReadList", err)
				}
				ary[j] = it
				//fmt.Println("j", j, "it", it)
			}
		} else {
			for j := 0; j < i; j++ {
				it, err := d.ReadObject()
				if err != nil {
					return nil, newCodecError("ReadList", err)
				}
				ary[j] = it
				//fmt.Println("j", j, "it", it)
			}
		}
		return ary, nil
	case (tag >= BC_LIST_DIRECT_UNTYPED && tag <= 0x7f) || (tag == BC_LIST_FIXED_UNTYPED || tag == BC_LIST_VARIABLE_UNTYPED):
		var i int
		if tag >= BC_LIST_DIRECT_UNTYPED && tag <= 0x7f {
			i = int(tag - BC_LIST_DIRECT_UNTYPED)
		} else {
			ii, err := d.readInt(TAG_READ)
			if err != nil {
				return nil, newCodecError("ReadType", err)
			}
			i = int(ii.(int32))
		}
		//fmt.Println("list len ", i)
		ary := make([]interface{}, i)
		for j := 0; j < i; j++ {
			it, err := d.ReadObject()
			if err != nil {
				return nil, newCodecError("ReadList", err)
			}
			ary[j] = it
			//fmt.Println("j", j, "it", it)
		}
		//read the endbyte of list
		//d.readBufByte()  cse 暂时注释掉,dubbo的hessian编码没有这个结尾标记符
		//fmt.Println("endList", bt)
		return ary, nil
	default:
		return nil, newCodecError("unkonw tag")
	}
	return nil, newCodecError("wrong tag")
}

func isBuildInType(typeStr string) bool {
	switch typeStr {
	case ARRAY_STRING:
		return true
	case ARRAY_INT:
		return true
	case ARRAY_FLOAT:
		return true
	case ARRAY_DOUBLE:
		return true
	case ARRAY_BOOL:
		return true
	case ARRAY_LONG:
		return true
	default:
		return false
	}
}

func (d *decoder) readBinary(flag int32) (interface{}, error) {
	var tag byte
	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readBufByte()
	}
	last := true
	var len int32
	if (tag >= BC_BINARY_DIRECT && tag <= INT_DIRECT_MAX) || (tag == BC_BINARY || tag == BC_BINARY_CHUNK) {
		if tag == BC_BINARY_CHUNK {
			last = false
		} else {
			last = true
		}
		l, err := d.getBinLen(tag)
		if err != nil {
			return nil, newCodecError("getStrLen", err)
		}
		len = int32(l)
		data := make([]byte, len)
		for i := 0; ; {
			if int32(i) == len {
				if last {
					//fmt.Println("last ", last, "i", i, "len", len)
					return string(data), nil
				}

				buf := make([]byte, 1)
				_, err := io.ReadFull(d.reader, buf)

				if err != nil {
					return nil, newCodecError("byte1 integer", err)
				}
				b := buf[0]
				switch {
				case b == BC_BINARY_CHUNK || b == BC_BINARY:
					if b == BC_BINARY_CHUNK {
						last = false
					} else {
						last = true
					}
					l, err := d.getStrLen(b)
					if err != nil {
						return nil, newCodecError("getStrLen", err)
					}
					len += l
					bs := make([]byte, 0, len)
					copy(bs, data)
					data = bs
				default:
					return nil, newCodecError("tag error ", err)
				}
			} else {
				buf := make([]byte, 1)
				_, err := io.ReadFull(d.reader, buf)

				if err != nil {
					return nil, newCodecError("byte2 integer", err)
				}
				data[i] = buf[0]
				i++
			}
		}
		return data, nil
	} else {
		//fmt.Println(tag, len, last)
		return nil, newCodecError("byte3 integer")
	}

}

func (d *decoder) getBinLen(tag byte) (int, error) {
	if tag >= BC_BINARY_DIRECT && tag <= INT_DIRECT_MAX {
		return int(tag - BC_BINARY_DIRECT), nil
	}
	bs := make([]byte, 2)
	_, err := io.ReadFull(d.reader, bs)
	if err != nil {
		return 0, newCodecError("parse binary", err)
	}
	return int(bs[0]<<8 + bs[1]), nil
}

func (d *decoder) readClassDef() (interface{}, error) {
	f, err := d.readString(TAG_READ)
	if err != nil {
		return nil, newCodecError("ReadClassDef", err)
	}
	clsName, ok := f.(string)
	if !ok {
		return nil, newCodecError("wrong type")
	}
	n, err := d.readInt(TAG_READ)
	if err != nil {
		return nil, newCodecError("ReadClassDef", err)
	}
	no, ok := n.(int32)
	fields := make([]string, no)
	for i := 0; i < int(no); i++ {
		s, err := d.readString(TAG_READ)
		if err != nil {
			return nil, newCodecError("ReadClassDef", err)
		}
		s1, ok := s.(string)
		if !ok {
			return nil, newCodecError("wrong type")
		}
		fields[i] = s1
	}
	cls := ClassDef{clsName, fields}
	return cls, nil
}

//func floatDecoder(r io.Reader) (interface{}, error) {
//	buf := make([]byte, 4)
//	if _, err := io.ReadFull(r, buf); err != nil {
//		return nil, newCodecError("float", err)
//	}
//	bits := binary.LittleEndian.Uint32(buf)
//	datum := math.Float32frombits(bits)
//	return datum, nil
//}
