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
	"reflect"
	"testing"
)

type Person struct {
	firstName string
	lastName  string
}

func TestSerializer(t *testing.T) {
	p := Person{"John", "Doe"}
	gh := NewGoHessian(nil, nil)
	bt, _ := gh.ToBytes(p)
	p_new, _ := gh.ToObject(bt)
	fmt.Println("bt", string(bt))
	reflect.DeepEqual(p, p_new)
}
