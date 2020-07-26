// Copyright 2019 CUE Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package convert_test

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/cockroachdb/apd/v2"
	"github.com/google/go-cmp/cmp"

	_ "cuelang.org/go/cue" // set internal.CoreValue
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/internal/core/adt"
	"cuelang.org/go/internal/core/convert"
	"cuelang.org/go/internal/core/debug"
	"cuelang.org/go/internal/core/eval"
	"cuelang.org/go/internal/core/runtime"
)

func mkBigInt(a int64) (v apd.Decimal) { v.SetInt64(a); return }

func TestConvert(t *testing.T) {
	i34 := big.NewInt(34)
	d35 := mkBigInt(35)
	n36 := mkBigInt(-36)
	f37 := big.NewFloat(37.0000)
	testCases := []struct {
		goVal interface{}
		want  string
	}{{
		nil, "_",
	}, {
		true, "true",
	}, {
		false, "false",
	}, {
		errors.New("oh noes"), "_|_(oh noes)",
	}, {
		"foo", `"foo"`,
	}, {
		"\x80", `_|_(cannot convert result to string: invalid UTF-8)`,
	}, {
		3, "3",
	}, {
		uint(3), "3",
	}, {
		uint8(3), "3",
	}, {
		uint16(3), "3",
	}, {
		uint32(3), "3",
	}, {
		uint64(3), "3",
	}, {
		int8(-3), "-3",
	}, {
		int16(-3), "-3",
	}, {
		int32(-3), "-3",
	}, {
		int64(-3), "-3",
	}, {
		float64(3.1), "3.1",
	}, {
		float32(3.1), "3.1",
	}, {
		uintptr(3), "3",
	}, {
		&i34, "34",
	}, {
		&f37, "37",
	}, {
		&d35, "35",
	}, {
		&n36, "-36",
	}, {
		[]int{1, 2, 3, 4}, `(#list){
  0: (int){ 1 }
  1: (int){ 2 }
  2: (int){ 3 }
  3: (int){ 4 }
}`,
	}, {
		struct {
			A int
			B *int
		}{3, nil},
		"(struct){\n  A: (int){ 3 }\n}",
	}, {
		[]interface{}{}, "(#list){\n}",
	}, {
		[]interface{}{nil}, "(#list){\n  0: (_){ _ }\n}",
	}, {
		map[string]interface{}{"a": 1, "x": nil}, `(struct){
  a: (int){ 1 }
  x: (_){ _ }
}`,
	}, {
		map[string][]int{
			"a": {1},
			"b": {3, 4},
		}, `(struct){
  a: (#list){
    0: (int){ 1 }
  }
  b: (#list){
    0: (int){ 3 }
    1: (int){ 4 }
  }
}`,
	}, {
		map[bool]int{}, "_|_(unsupported Go type for map key (bool))",
	}, {
		map[struct{}]int{{}: 2}, "_|_(unsupported Go type for map key (struct {}))",
	}, {
		map[int]int{1: 2}, `(struct){
  "1": (int){ 2 }
}`,
	}, {
		struct {
			a int
			b int
		}{3, 4},
		"(struct){\n}",
	}, {
		struct {
			A int
			B int `json:"-"`
			C int `json:",omitempty"`
		}{3, 4, 0},
		`(struct){
  A: (int){ 3 }
}`,
	}, {
		struct {
			A int
			B int
		}{3, 4},
		`(struct){
  A: (int){ 3 }
  B: (int){ 4 }
}`,
	}, {
		struct {
			A int `json:"a"`
			B int `yaml:"b"`
		}{3, 4},
		`(struct){
  a: (int){ 3 }
  b: (int){ 4 }
}`,
	}, {
		struct {
			A int `json:"" yaml:"" protobuf:"aa"`
			B int `yaml:"cc" json:"bb" protobuf:"aa"`
		}{3, 4},
		`(struct){
  aa: (int){ 3 }
  bb: (int){ 4 }
}`,
	}, {
		&struct{ A int }{3}, `(struct){
  A: (int){ 3 }
}`,
	}, {
		(*struct{ A int })(nil), "_",
	}, {
		reflect.ValueOf(3), "3",
	}, {
		time.Date(2019, 4, 1, 0, 0, 0, 0, time.UTC), `(string){ "2019-04-01T00:00:00Z" }`,
	}}
	r := runtime.New()
	for _, tc := range testCases {
		e := eval.New(r)
		ctx := adt.NewContext(r, e, &adt.Vertex{})
		t.Run("", func(t *testing.T) {
			v := convert.GoValueToValue(ctx, tc.goVal, true)
			got := debug.NodeString(ctx, v, nil)
			if got != tc.want {
				t.Error(cmp.Diff(got, tc.want))
			}
		})
	}
}

func TestX(t *testing.T) {
	t.Skip()

	x := []string{}

	r := runtime.New()
	e := eval.New(r)
	ctx := adt.NewContext(r, e, &adt.Vertex{})

	v := convert.GoValueToValue(ctx, x, false)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	got := debug.NodeString(ctx, v, nil)
	t.Error(got)
}

func TestConvertType(t *testing.T) {
	testCases := []struct {
		goTyp interface{}
		want  string
	}{{
		struct {
			A int      `cue:">=0&<100"`
			B *big.Int `cue:">=0"`
			C *big.Int
			D big.Int
			F *big.Float
		}{},
		// TODO: indicate that B is explicitly an int only.
		`{
  A: (((int & >=-9223372036854775808) & <=9223372036854775807) & (>=0 & <100))
  B: (int & >=0)
  C?: int
  D: int
  F?: number
}`,
	}, {
		&struct {
			A int16 `cue:">=0&<100"`
			B error `json:"b,"`
			C string
			D bool
			F float64
			L []byte
			T time.Time
			G func()
		}{},
		`(*null|{
  A: (((int & >=-32768) & <=32767) & (>=0 & <100))
  b: null
  C: string
  D: bool
  F: number
  L?: (*null|bytes)
  T: _
})`,
	}, {
		struct {
			A int `cue:"<"` // invalid
		}{},
		"_|_(invalid tag \"<\" for field \"A\": expected operand, found 'EOF' (and 1 more errors))",
	}, {
		struct {
			A int `json:"-"` // skip
			D *apd.Decimal
			P ***apd.Decimal
			I interface{ Foo() }
			T string `cue:""` // allowed
			h int
		}{},
		`{
  D?: number
  P?: (*null|number)
  I?: _
  T: (string & _)
}`,
	}, {
		struct {
			A int8 `cue:"C-B"`
			B int8 `cue:"C-A,opt"`
			C int8 `cue:"A+B"`
		}{},
		// TODO: should B be marked as optional?
		`{
  A: (((int & >=-128) & <=127) & (〈0;C〉 - 〈0;B〉))
  B?: (((int & >=-128) & <=127) & (〈0;C〉 - 〈0;A〉))
  C: (((int & >=-128) & <=127) & (〈0;A〉 + 〈0;B〉))
}`,
	}, {
		[]string{},
		`(*null|[
  ...string,
])`,
	}, {
		[4]string{},
		`(4 * [
  string,
])`,
	}, {
		[]func(){},
		"_|_(unsupported Go type (func()))",
	}, {
		map[string]struct{ A map[string]uint }{},
		`(*null|{
  [string]: {
    A?: (*null|{
      [string]: ((int & >=0) & <=18446744073709551615)
    })
  }
})`,
	}, {
		map[float32]int{},
		`_|_(unsupported Go type for map key (float32))`,
	}, {
		map[int]map[float32]int{},
		`_|_(unsupported Go type for map key (float32))`,
	}, {
		map[int]func(){},
		`_|_(unsupported Go type (func()))`,
	}, {
		time.Now, // a function
		"_|_(unsupported Go type (func() time.Time))",
	}}

	r := runtime.New()

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			e := eval.New(r)
			ctx := adt.NewContext(r, e, &adt.Vertex{})
			v, _ := convert.GoTypeToExpr(ctx, tc.goTyp)
			got := debug.NodeString(ctx, v, nil)
			if got != tc.want {
				t.Errorf("\n got %q;\nwant %q", got, tc.want)
			}
		})
	}
}
