// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package eval

import "testing"
import "fmt"
import "encoding/json"
import "reflect"
import "sort"

import "github.com/open-policy-agent/opa/opalog"

func TestEvalRef(t *testing.T) {

	var tests = []struct {
		ref      string
		expected interface{}
	}{
		{"c[i][j]", `[
            {"i": 0, "j": "x"},
            {"i": 0, "j": "y"},
            {"i": 0, "j": "z"}
         ]`},
		{"c[i][j][k]", `[
            {"i": 0, "j": "x", "k": 0},
            {"i": 0, "j": "x", "k": 1},
            {"i": 0, "j": "x", "k": 2},
            {"i": 0, "j": "y", "k": 0},
            {"i": 0, "j": "y", "k": 1},
            {"i": 0, "j": "z", "k": "p"},
            {"i": 0, "j": "z", "k": "q"}
        ]`},
		{"d[x][y]", `[
            {"x": "e", "y": 0},
            {"x": "e", "y": 1}
        ]`},
		{`c[i]["x"][k]`, `[
            {"i": 0, "k": 0},
            {"i": 0, "k": 1},
            {"i": 0, "k": 2}
        ]`},
		{"c[i][j][i]", `[
            {"i": 0, "j": "x"},
            {"i": 0, "j": "y"}
        ]`},
		{`c[i]["deadbeef"][k]`, nil},
		{`c[999]`, nil},
	}

	data := loadSmallTestData()

	ctx := &TopDownContext{
		Store:    NewStorageFromJSONObject(data),
		Bindings: make(map[opalog.Var]opalog.Value),
	}

	for i, tc := range tests {

		switch e := tc.expected.(type) {
		case nil:
			var tmp *TopDownContext
			err := evalRef(ctx, parseRef(tc.ref), func(ctx *TopDownContext) error {
				tmp = ctx
				return nil
			})
			if err != nil {
				t.Errorf("Test case (%d): unexpected error: %v", i+1, err)
				continue
			}
			if tmp != nil {
				t.Errorf("Test case (%d): expected no bindings (nil) but got: %v", i+1, tmp)
			}
		case string:
			expected := loadExpectedBindings(e)
			err := evalRef(ctx, parseRef(tc.ref), func(ctx *TopDownContext) error {
				if len(expected) > 0 {
					for j, exp := range expected {
						if reflect.DeepEqual(exp, ctx.Bindings) {
							tmp := expected[:j]
							expected = append(tmp, expected[j+1:]...)
							return nil
						}
					}
				}
				// If there was not a matching expected binding, treat this case as a failure.
				return fmt.Errorf("unexpected bindings: %v", ctx.Bindings)
			})
			if err != nil {
				t.Errorf("Test case %d: expected success but got error: %v", i+1, err)
				continue
			}
			if len(expected) > 0 {
				t.Errorf("Test case %d: missing expected bindings: %v", i+1, expected)
			}
		}

	}
}

func TestEvalTerms(t *testing.T) {

	tests := []struct {
		rule     string
		expected string
	}{
		{"p[x] :- c[i][j][k] = x", `[
            {"i": 0, "j": "x", "k": 0},
            {"i": 0, "j": "x", "k": 1},
            {"i": 0, "j": "x", "k": 2},
            {"i": 0, "j": "y", "k": 0},
            {"i": 0, "j": "y", "k": 1},
            {"i": 0, "j": "z", "k": "p"},
            {"i": 0, "j": "z", "k": "q"}
        ]`},
		{"p[x] :- d[x][y] = a[i]", `[
            {"x": "e", "y": 0, "i": 0},
            {"x": "e", "y": 0, "i": 1},
            {"x": "e", "y": 0, "i": 2},
            {"x": "e", "y": 0, "i": 3},
            {"x": "e", "y": 1, "i": 0},
            {"x": "e", "y": 1, "i": 1},
            {"x": "e", "y": 1, "i": 2},
            {"x": "e", "y": 1, "i": 3}
        ]`},
		{"p[x] :- d[x][y] = z[i]", `[]`},
	}

	data := loadSmallTestData()

	for i, tc := range tests {

		ctx := &TopDownContext{
			Rule:     parseRule(tc.rule),
			Store:    NewStorageFromJSONObject(data),
			Bindings: make(map[opalog.Var]opalog.Value),
		}

		expected := loadExpectedBindings(tc.expected)

		err := evalTerms(ctx, func(ctx *TopDownContext) error {
			if len(expected) > 0 {
				for j, exp := range expected {
					if reflect.DeepEqual(exp, ctx.Bindings) {
						tmp := expected[:j]
						expected = append(tmp, expected[j+1:]...)
						return nil
					}
				}
			}
			// If there was not a matching expected binding, treat this case as a failure.
			return fmt.Errorf("unexpected bindings: %v", ctx.Bindings)
		})
		if err != nil {
			t.Errorf("Test case %d: expected success but got error: %v", i+1, err)
			continue
		}
		if len(expected) > 0 {
			t.Errorf("Test case %d: missing expected bindings: %v", i+1, expected)
		}
	}
}

func TestPlugValue(t *testing.T) {

	a := opalog.Var("a")
	b := opalog.Var("b")
	c := opalog.Var("c")
	k := opalog.Var("k")
	v := opalog.Var("v")
	cs := parseTerm("[c]").Value
	ks := parseTerm(`{k: "world"}`).Value
	vs := parseTerm(`{"hello": v}`).Value
	hello := opalog.String("hello")
	world := opalog.String("world")

	ctx1 := &TopDownContext{Bindings: make(Bindings)}
	ctx1 = ctx1.Bind(a, b)
	ctx1 = ctx1.Bind(b, cs)
	ctx1 = ctx1.Bind(c, ks)
	ctx1 = ctx1.Bind(k, hello)

	ctx2 := &TopDownContext{Bindings: make(Bindings)}
	ctx2 = ctx2.Bind(a, b)
	ctx2 = ctx2.Bind(b, cs)
	ctx2 = ctx2.Bind(c, vs)
	ctx2 = ctx2.Bind(v, world)

	expected := parseTerm(`[{"hello": "world"}]`).Value

	r1 := plugValue(a, ctx1.Bindings)

	if !expected.Equal(r1) {
		t.Errorf("Expected %v but got %v", expected, r1)
		return
	}

	r2 := plugValue(a, ctx2.Bindings)

	if !expected.Equal(r2) {
		t.Errorf("Expected %v but got %v", expected, r2)
	}
}

func TestTopDownScalarDoc(t *testing.T) {
	tests := []struct {
		note     string
		rule     string
		expected string
	}{
		{"undefined", "p = null :- false", ""}, // "" will be converted to Undefined
		{"null", "p = null :- true", "null"},
		{"bool: true", "p = true :- true", "true"},
		{"bool: false", "p = false :- true", "false"},
		{"number: 3", "p = 3 :- true", "3"},
		{"number: 3.0", "p = 3.0 :- true", "3.0"},
		{"number: 66.66667", "p = 66.66667 :- true", "66.66667"},
		{`string: "hello"`, `p = "hello" :- true`, `"hello"`},
		{`string: ""`, `p = "" :- true`, `""`},
		{"array: [1,2,3,4]", "p = [1,2,3,4] :- true", "[1,2,3,4]"},
		{"array: []", "p = [] :- true", "[]"},
		{`object/nested composites: {"a": [1], "b": [2], "c": [3]}`,
			`p = {"a": [1], "b": [2], "c": [3]} :- true`,
			`{"a": [1], "b": [2], "c": [3]}`},
	}

	data := loadSmallTestData()

	for i, tc := range tests {

		ctx := &TopDownContext{
			Rule:     parseRule(tc.rule),
			Store:    NewStorageFromJSONObject(data),
			Bindings: make(map[opalog.Var]opalog.Value),
		}

		expected := loadExpectedResult(tc.expected)
		result, err := TopDownQuery(ctx)

		if err != nil {
			t.Errorf("Test case %d (%v): unexpected error: %v", i+1, tc.note, err)
			continue
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Test case %d (%v): expected %v but got: %v", i+1, tc.note, expected, result)
		}
	}

}

func TestTopDownSetDoc(t *testing.T) {

	tests := []struct {
		note     string
		rule     string
		expected string
	}{
		{"array values", "p[x] :- a[i] = x", `[1, 2, 3, 4]`},
		{"array indices", "p[x] :- a[x] = _", `[0, 1, 2, 3]`},
		{"object keys", "p[x] :- b[x] = _", `["v1", "v2"]`},
		{"object values", "p[x] :- b[i] = x", `["hello", "goodbye"]`},
		{"nested composites", "p[x] :- f[i] = x", `[{"xs": [1.0], "ys": [2.0]}, {"xs": [2.0], "ys": [3.0]}]`},
		{"deep ref/heterogeneous", "p[x] :- c[i][j][k] = x", `[null, 3.14159, true, false, true, false, "foo"]`},
	}

	data := loadSmallTestData()

	for i, tc := range tests {
		ctx := &TopDownContext{
			Rule:     parseRule(tc.rule),
			Store:    NewStorageFromJSONObject(data),
			Bindings: make(map[opalog.Var]opalog.Value),
		}

		expected := loadExpectedResult(tc.expected)
		result, err := TopDownQuery(ctx)

		if err != nil {
			t.Errorf("Test case %d (%v): unexpected error: %v", i+1, tc.note, err)
			continue
		}

		sort.Sort(ResultSet(result.([]interface{})))

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Test case %d (%v): expected %v but got: %v", i+1, tc.note, expected, result)
		}
	}

}

func TestTopDownObjectDoc(t *testing.T) {
	tests := []struct {
		note     string
		rule     string
		expected interface{}
	}{
		{"identity", "p[k] = v :- b[k] = v", `{"v1": "hello", "v2": "goodbye"}`},
		{"composites", "p[k] = v :- d[k] = v", `{"e": ["bar", "baz"]}`},
		{"non-string key", "p[k] = v :- a[k] = v", fmt.Errorf("cannot produce object with non-string key: 0")},
	}

	data := loadSmallTestData()

	for i, tc := range tests {

		ctx := &TopDownContext{
			Rule:     parseRule(tc.rule),
			Store:    NewStorageFromJSONObject(data),
			Bindings: make(map[opalog.Var]opalog.Value),
		}

		switch e := tc.expected.(type) {
		case string:
			expected := loadExpectedResult(e)
			result, err := TopDownQuery(ctx)
			if err != nil {
				t.Errorf("Test case %d (%v): unexpected error: %v", i+1, tc.note, err)
				continue
			}
			if !reflect.DeepEqual(result, expected) {
				t.Errorf("Test case %d (%v): expected %v but got %v", i+1, tc.note, expected, result)
			}
		case error:
			_, err := TopDownQuery(ctx)
			if !reflect.DeepEqual(err, e) {
				t.Errorf("Test case %d (%v): expected error %v but got %v", i+1, tc.note, e, err)
			}
		}

	}
}

func TestTopDownEqExpr(t *testing.T) {

	tests := []struct {
		note     string
		rule     string
		expected interface{}
	}{
		// undefined cases
		{"undefined: same type", `p = true :- true = false`, ""},
		{"undefined: diff type", `p = true :- 42 = "hello"`, ""},
		{"undefined: array order", `p = true :- [1,2,3] = [1,3,2]`, ""},
		{"undefined: ref value", "p = true :- a[3] = 9999", ""},
		{"undefined: ref values", "p = true :- a[i] = 9999", ""},
		{"undefined: ground var", "p = true :- a[3] = x, x = 3", ""},
		{"undefined: array var 1", "p = true :- [1,x,x] = [1,2,3]", ""},
		{"undefined: array var 2", "p = true :- [1,x,3] = [1,2,x]", ""},
		{"undefined: object var 1", `p = true :- {"a": 1, "b": 2} = {"a": a, "b": a}`, ""},
		{"undefined: array deep var 1", "p = true :- [[1,x],[3,x]] = [[1,2],[3,4]]", ""},
		{"undefined: array deep var 2", "p = true :- [[1,x],[3,4]] = [[1,2],[x,4]]", ""},
		{"undefined: array uneven", `p = true :- [true, false, "foo", "deadbeef"] = c[i][j]`, ""},
		{"undefined: object uneven", `p = true :- {"a": 1, "b": 2} = {"a": 1}`, ""},
		{"undefined: occurs 1", "p = true :- [y,x] = [[x],y]", ""},
		{"undefined: occurs 2", "p = true :- [y,x] = [{\"a\": x}, y]", ""},

		// ground terms
		{"ground: bool", `p = true :- true = true`, "true"},
		{"ground: string", `p = true :- "string" = "string"`, "true"},
		{"ground: number", `p = true :- 17 = 17`, "true"},
		{"ground: null", `p = true :- null = null`, "true"},
		{"ground: array", `p = true :- [1,2,3] = [1,2,3]`, "true"},
		{"ground: object", `p = true :- {"b": false, "a": [1,2,3]} = {"a": [1,2,3], "b": false}`, "true"},
		{"ground: ref 1", `p = true :- a[2] = 3`, "true"},
		{"ground: ref 2", `p = true :- b["v2"] = "goodbye"`, "true"},
		{"ground: ref 3", `p = true :- d["e"] = ["bar", "baz"]`, "true"},
		{"ground: ref 4", `p = true :- c[0].x[1] = c[0].z["q"]`, "true"},

		// variables
		{"var: a=b=c", "p[a] :- a = b, c = 42, b = c", "[42]"},
		{"var: ref value", "p = true :- a[3] = x, x = 4", "true"},
		{"var: ref values", "p = true :- a[i] = x, x = 2", "true"},
		{"var: ref key", "p = true :- a[i] = 4, x = 3", "true"},
		{"var: ref keys", "p = true :- a[i] = x, i = 2", "true"},
		{"var: ref ground var", "p[x] :- i = 2, a[i] = x", "[3]"},
		{"var: ref ref", "p[x] :- c[0].x[i] = c[0].z[j], x = [i, j]", `[[0, "p"], [1, "q"]]`},

		// arrays and variables
		{"pattern: array", "p[x] :- [1,x,3] = [1,2,3]", "[2]"},
		{"pattern: array 2", "p[x] :- [[1,x],[3,4]] = [[1,2],[3,4]]", "[2]"},
		{"pattern: array same var", "p[x] :- [2,x,3] = [x,2,3]", "[2]"},
		{"pattern: array multiple vars", "p[z] :- [1,x,y] = [1,2,3], z = [x, y]", "[[2, 3]]"},
		{"pattern: array multiple vars 2", "p[z] :- [1,x,3] = [y,2,3], z = [x, y]", "[[2, 1]]"},
		{"pattern: array ref", "p[x] :- [1,2,3,x] = [a[0], a[1], a[2], a[3]]", "[4]"},
		{"pattern: array = ref", "p[x] :- [true, false, x] = c[i][j]", `["foo"]`},
		{"pattern: array = ref (reversed)", "p[x] :-  c[i][j] = [true, false, x]", `["foo"]`},
		{"pattern: array = var", "p[y] :- [1,2,x] = y, x = 3", "[[1,2,3]]"},

		// objects and variables
		{"pattern: object val", `p[y] :- {"x": y} = {"x": "y"}`, `["y"]`},
		{"pattern: var key error 1", `p[x] :- {x: "y"} = {"x": "y"}`, fmt.Errorf("cannot unify object with variable key: x")},
		{"pattern: var key error 2", `p[x] :- {"x": "y"} = {x: "y"}`, fmt.Errorf("cannot unify object with variable key: x")},
		{"pattern: var key error 3", `p = true :- {"x": [{y: "z"}]} = {"x": [{"y": "z"}]}`, fmt.Errorf("cannot unify object with variable key: y")},
		{"pattern: object same var", `p[x] :- {"x": x, "y": x} = {"x": 1, "y": 1}`, "[1]"},
		{"pattern: object multiple vars", `p[z] :- {"x": x, "y": y} = {"x": 1, "y": 2}, z = [x, y]`, "[[1, 2]]"},
		{"pattern: object multiple vars 2", `p[z] :- {"x": x, "y": 2} = {"x": 1, "y": y}, z = [x, y]`, "[[1, 2]]"},
		{"pattern: object ref", `p[x] :- {"p": c[0].x[0], "q": x} = c[i][j]`, `[false]`},
		{"pattern: object = ref", `t[x] :- {"p": p, "q": q} = c[i][j], x = [i, j, p, q]`, `[[0, "z", true, false]]`},
		{"pattern: object = ref (reversed)", `t[x] :- c[i][j] = {"p": p, "q": q}, x = [i, j, p, q]`, `[[0, "z", true, false]]`},
		{"pattern: object = var", `p[x] :- {"a": 1, "b": b} = x, b = 2`, `[{"a": 1, "b": 2}]`},
		{"pattern: object/array nested", `p[ys] :- f[i] = {"xs": [2.0], "ys": ys}`, `[[3.0]]`},
	}

	data := loadSmallTestData()

	for i, tc := range tests {

		ctx := &TopDownContext{
			Rule:     parseRule(tc.rule),
			Store:    NewStorageFromJSONObject(data),
			Bindings: make(map[opalog.Var]opalog.Value),
		}

		switch e := tc.expected.(type) {
		case error:
			_, err := TopDownQuery(ctx)
			if !reflect.DeepEqual(err, e) {
				t.Errorf("Test case %d (%v): expected error %v but got %v", i+1, tc.note, e, err)
			}
		case string:
			expected := loadExpectedResult(e)
			result, err := TopDownQuery(ctx)
			if err != nil {
				t.Errorf("Test case %d (%v): unexpected error: %v", i+1, tc.note, err)
				continue
			}
			if !reflect.DeepEqual(result, expected) {
				t.Errorf("Test case %d (%v): expected %v but got: %v", i+1, tc.note, expected, result)
			}
		}
	}

}

// TODO(tsandall): cover dereferencing of variables.

func loadExpectedBindings(input string) []Bindings {
	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		panic(err)
	}
	var expected []Bindings
	for _, bindings := range data {
		buf := make(Bindings)
		for k, v := range bindings {
			switch v := v.(type) {
			case string:
				buf[opalog.Var(k)] = opalog.String(v)
			case float64:
				buf[opalog.Var(k)] = opalog.Number(v)
			default:
				panic("unreachable")
			}
		}
		expected = append(expected, buf)
	}

	return expected
}

func loadExpectedResult(input string) interface{} {
	if len(input) == 0 {
		return Undefined{}
	}
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		panic(err)
	}
	switch data := data.(type) {
	case []interface{}:
		sort.Sort(ResultSet(data))
		return data
	default:
		return data
	}
}

func loadSmallTestData() map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(`{
        "a": [1,2,3,4],
        "b": {
            "v1": "hello",
            "v2": "goodbye"
        },
        "c": [{
            "x": [true, false, "foo"],
            "y": [null, 3.14159],
            "z": {"p": true, "q": false}
        }],
        "d": {
            "e": ["bar", "baz"]
        },
        "f": [
            {"xs": [1.0], "ys": [2.0]},
            {"xs": [2.0], "ys": [3.0]}
        ],
        "z": []
    }`), &data)
	if err != nil {
		panic(err)
	}
	return data
}

func parseRef(input string) opalog.Ref {
	body := opalog.MustParseStatement(input).(opalog.Body)
	return body[0].Terms.(*opalog.Term).Value.(opalog.Ref)
}

func parseRule(input string) *opalog.Rule {
	return opalog.MustParseStatement(input).(*opalog.Rule)
}

func parseTerm(input string) *opalog.Term {
	return opalog.MustParseStatement(input).(opalog.Body)[0].Terms.(*opalog.Term)
}

// ResultSet is used to sort set documents produeced by rules for comparison purposes.
type ResultSet []interface{}

// Less returns true if the i'th index of resultSet is less than the j'th index.
func (resultSet ResultSet) Less(i, j int) bool {
	return Compare(resultSet[i], resultSet[j]) < 0
}

// Swap exchanges the i'th and j'th values in resultSet.
func (resultSet ResultSet) Swap(i, j int) {
	tmp := resultSet[i]
	resultSet[i] = resultSet[j]
	resultSet[j] = tmp
}

// Len returns the size of the resultSet.
func (resultSet ResultSet) Len() int {
	return len(resultSet)
}