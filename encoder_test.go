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
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type P struct {
	X, Y, Z int
	Name    string
}

type Box struct {
	Width  int
	Height int
	Color  string
	Open   bool
}

type BB struct {
	Name   string
	List   []string
	Mp     map[int32]string
	Pst    P
	Number int32
}

func TestSerilazer(t *testing.T) {
	ts4 := []string{"t1", "t2", "t3"}
	mp1 := map[int32]string{1: "test1", 2: "test2", 3: "test3"}
	p := P{1, 2, 3, "ABC"}
	bb := BB{"AB", ts4, mp1, p, 4}
	gh := NewGoHessian(nil, nil)
	bt, _ := gh.ToBytes(bb)
	fmt.Println("bt", string(bt))

}

func TestDecoder_Instance(t *testing.T) {
	ts4 := []string{"t1", "t2", "t3"}
	mp1 := map[int32]string{1: "test1", 2: "test2", 3: "test3"}
	p := P{1, 2, 3, "ABC"}
	bb := BB{"AB", ts4, mp1, p, 4}
	br := bytes.NewBuffer(nil)
	e := NewEncoder(br, nil)
	e.WriteObject(bb)
	fmt.Println("e", br)
	bt := bytes.NewReader(br.Bytes())
	d := NewDecoder(bt, nil)
	d.RegisterType("BB", reflect.TypeOf(BB{}))
	d.RegisterType("P", reflect.TypeOf(P{}))
	it, err := d.ReadObject()
	if err != nil {
		fmt.Println("err", err)
	}
	fmt.Println("decode t", it, "bt len", len(br.Bytes()))
}

func TestEncoder_WriteObject(t *testing.T) {
	mp2 := make(map[int]string)
	mp2[1] = "test1"
	mp2[2] = "test2"
	mp2[3] = "test3"
	br := bytes.NewBuffer(nil)
	e7 := NewEncoder(br, nil)
	e7.WriteObject(mp2)
	fmt.Println("encode map buf->", string(br.Bytes()), len(br.Bytes()), br.Bytes())
	bt2 := br.Bytes()
	br2 := bytes.NewReader(br.Bytes())
	d7 := NewDecoder(br2, nil)
	t7, _ := d7.ReadObject()
	fmt.Println("decode map", t7, "bt2", len(bt2))

}

func TestEncoder_WriteList(t *testing.T) {
	ts3 := [3]string{"t1", "t2", "t3"}
	br := bytes.NewBuffer(nil)
	e7 := NewEncoder(br, nil)
	e7.WriteObject(ts3)
	fmt.Println("encode array buf->", string(br.Bytes()), len(br.Bytes()), br.Bytes())
	bt2 := br.Bytes()
	br2 := bytes.NewReader(br.Bytes())
	d7 := NewDecoder(br2, nil)
	t7, _ := d7.ReadObject()
	fmt.Println("decode array", t7, "bt2", len(bt2))

}

func TestEncoder_WriteString(t *testing.T) {
	str := "HessianSerializer"
	br := bytes.NewBuffer(nil)
	e7 := NewEncoder(br, nil)
	e7.WriteObject(str)
	bt2 := br.Bytes()
	br2 := bytes.NewReader(br.Bytes())
	d7 := NewDecoder(br2, nil)
	t7, _ := d7.ReadObject()
	fmt.Println("decode string", t7, "bt2", len(bt2))
	if strings.Compare(str, t7.(string)) == 0 {
		fmt.Println("succes for ", str)
	}
}
