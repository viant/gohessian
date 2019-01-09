/*
 *
 *  * Copyright 2012=2016 Viant.
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
	"testing"
)

func makeInt(ps []int32) []interface{} {
	rt := make([]interface{}, len(ps))
	for i, t := range ps {
		rt[i] = t
	}
	return rt
}

/*
Commented out at Michael's request.  Delete this or not?
func TestDecoder_Invoker(t *testing.T) {
	file3, err := os.Open("/Users/mhsieh/test/files/msg.hs")
	if err != nil {
		log.Fatal(err)
	}
	d := NewDecoder(file3, nil)
	iv := message.Invoker{}
	d.RegisterType("com.sm.message.Invoker", reflect.TypeOf(iv))
	ts1, _ := d.ReadObject()
	fmt.Println("ts1 ", ts1)
	pv := []int32{1, 2}
	iv = message.Invoker{"Math", "Add", makeInt(pv)}
	br := bytes.NewBuffer(nil)
	e := NewEncoder(br, message.DefaultNameMap())
	e.WriteObject(iv)
	bt := bytes.NewReader(br.Bytes())
	d = NewDecoder(bt, message.DefaultTypeMap())
	ts1, _ = d.ReadObject()
	fmt.Println("ts1 ", ts1)
}
*/

/*
Commented out at michael's request.  Delete this or not?
func TestDecoder_Double(t *testing.T) {
	file3, err := os.Open("/Users/mhsieh/test/files/dbl.hs")
	if err != nil {
		log.Fatal(err)
	}
	d := NewDecoder(file3, nil)
	ts1, _ := d.ReadObject()
	fmt.Println("ts1 ", ts1)
	ts2, _ := d.ReadObject()
	fmt.Println("ts2 ", ts2)
	//ts3, _ := d.ReadObject()
	ts3, _ := d.readDouble(TAG_READ)
	fmt.Println("ts3 ", ts3)
	fmt.Println(reflect.TypeOf(ts2), reflect.ValueOf(ts2))
}
*/

func TestMethod_pkg(t *testing.T) {
	br := bytes.NewBuffer(nil)
	e := NewEncoder(br, nil)
	typ := reflect.TypeOf(e)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		fmt.Println("pkg", method.PkgPath, len(method.PkgPath), "type", mtype, mname)
	}

}
