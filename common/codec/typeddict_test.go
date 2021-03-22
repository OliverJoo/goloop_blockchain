/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func mustEncodeAny(v interface{}) *TypedObj {
	ev, err := EncodeAny(nil, v)
	if err != nil {
		panic(err)
	}
	return ev
}

func TestTypedDict_Encodings(t *testing.T) {
	data := map[string]*TypedObj{
		"a": mustEncodeAny("value A"),
		"b": mustEncodeAny("value B"),
		"c": mustEncodeAny("value C"),
	}
	keys := []string{"b", "c", "a"}

	t.Run("Unordered", func(t *testing.T) {
		// d1 := &TypedDict {
		// 	Map: data,
		// }
		for _, c := range []Codec{MP, RLP} {
			t.Run(c.Name(), func(t *testing.T) {
				bs, err := c.MarshalToBytes(data)
				assert.NoError(t, err)

				var d1 map[string]*TypedObj
				_, err = c.UnmarshalFromBytes(bs, &d1)
				assert.NoError(t, err)
				assert.Equal(t, data, d1)

				var d2 *TypedDict
				_, err = c.UnmarshalFromBytes(bs, &d2)
				assert.NoError(t, err)
				assert.Equal(t, data, d2.Map)
			})
		}
	})
	t.Run("Ordered", func(t *testing.T) {
		d1 := &TypedDict{
			Keys: keys,
			Map:  data,
		}
		for _, c := range []Codec{MP, RLP} {
			t.Run(c.Name(), func(t *testing.T) {
				bs, err := c.MarshalToBytes(d1)
				assert.NoError(t, err)

				var d2 *TypedDict
				_, err = c.UnmarshalFromBytes(bs, &d2)
				assert.NoError(t, err)
				assert.Equal(t, data, d2.Map)
				assert.Equal(t, keys, d2.Keys)

				var d3 map[string]*TypedObj
				_, err = c.UnmarshalFromBytes(bs, &d3)
				assert.NoError(t, err)
				assert.Equal(t, data, d3)
			})
		}
	})
}
