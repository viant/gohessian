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
	"reflect"
)

type Serializer interface {
	ToBytes(interface{}) ([]byte, error)
	ToObject([]byte) (interface{}, error)
}

type goHessian struct {
	typMap  map[string]reflect.Type
	nameMap map[string]string
}

func NewGoHessian(typMap map[string]reflect.Type, nmeMap map[string]string) Serializer {
	if typMap == nil {
		typMap = make(map[string]reflect.Type, 17)
	}
	if nmeMap == nil {
		nmeMap = make(map[string]string, 17)
	}
	return &goHessian{typMap, nmeMap}
}

func (gh *goHessian) ToBytes(object interface{}) ([]byte, error) {
	btBufs := bytes.NewBuffer(nil)
	e := NewEncoder(btBufs, gh.nameMap)
	_, err := e.WriteObject(object)
	if err != nil {
		return nil, err
	} else {
		return btBufs.Bytes(), nil
	}
}

func (gh *goHessian) ToObject(ins []byte) (interface{}, error) {
	btBufs := bytes.NewReader(ins)
	d := NewDecoder(btBufs, gh.typMap)
	return d.ReadObject()
}
