package sjson

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
)

const (
	setRaw    = 1
	setBool   = 2
	setInt    = 3
	setFloat  = 4
	setString = 5
	setDelete = 6
)

func sortJSON(json string) string {
	opts := pretty.Options{SortKeys: true}
	return string(pretty.Ugly(pretty.PrettyOptions([]byte(json), &opts)))
}

func testRaw(t *testing.T, kind int, expect, json, path string, value interface{}) {
	t.Helper()
	expect = sortJSON(expect)
	var json2 string
	var err error
	switch kind {
	default:
		json2, err = Set(json, path, value)
	case setRaw:
		json2, err = SetRaw(json, path, value.(string))
	case setDelete:
		json2, err = Delete(json, path)
	}

	if err != nil {
		t.Fatal(err)
	}
	json2 = sortJSON(json2)
	if json2 != expect {
		t.Fatalf("expected '%v', got '%v'", expect, json2)
	}
	var json3 []byte
	switch kind {
	default:
		json3, err = SetBytes([]byte(json), path, value)
	case setRaw:
		json3, err = SetRawBytes([]byte(json), path, []byte(value.(string)))
	case setDelete:
		json3, err = DeleteBytes([]byte(json), path)
	}
	json3 = []byte(sortJSON(string(json3)))
	if err != nil {
		t.Fatal(err)
	} else if string(json3) != expect {
		t.Fatalf("expected '%v', got '%v'", expect, string(json3))
	}
}
func TestBasic(t *testing.T) {
	testRaw(t, setRaw, `[{"hiw":"planet","hi":"world"}]`, `[{"hi":"world"}]`, "0.hiw", `"planet"`)
	testRaw(t, setRaw, `[true]`, ``, "0", `true`)
	testRaw(t, setRaw, `[null,true]`, ``, "1", `true`)
	testRaw(t, setRaw, `[1,null,true]`, `[1]`, "2", `true`)
	testRaw(t, setRaw, `[1,true,false]`, `[1,null,false]`, "1", `true`)
	testRaw(t, setRaw,
		`[1,{"hello":"when","this":[0,null,2]},false]`,
		`[1,{"hello":"when","this":[0,1,2]},false]`,
		"1.this.1", `null`)
	testRaw(t, setRaw,
		`{"a":1,"b":{"hello":"when","this":[0,null,2]},"c":false}`,
		`{"a":1,"b":{"hello":"when","this":[0,1,2]},"c":false}`,
		"b.this.1", `null`)
	testRaw(t, setRaw,
		`{"a":1,"b":{"hello":"when","this":[0,null,2,null,4]},"c":false}`,
		`{"a":1,"b":{"hello":"when","this":[0,null,2]},"c":false}`,
		"b.this.4", `4`)
	testRaw(t, setRaw,
		`{"b":{"this":[null,null,null,null,4]}}`,
		``,
		"b.this.4", `4`)
	testRaw(t, setRaw,
		`[null,{"this":[null,null,null,null,4]}]`,
		``,
		"1.this.4", `4`)
	testRaw(t, setRaw,
		`{"1":{"this":[null,null,null,null,4]}}`,
		``,
		":1.this.4", `4`)
	testRaw(t, setRaw,
		`{":1":{"this":[null,null,null,null,4]}}`,
		``,
		"\\:1.this.4", `4`)
	testRaw(t, setRaw,
		`{":\\1":{"this":[null,null,null,null,{".HI":4}]}}`,
		``,
		"\\:\\\\1.this.4.\\.HI", `4`)
	testRaw(t, setRaw,
		`{"app.token":"cde"}`,
		`{"app.token":"abc"}`,
		"app\\.token", `"cde"`)
	testRaw(t, setRaw,
		`{"b":{"this":{"😇":""}}}`,
		``,
		"b.this.😇", `""`)
	testRaw(t, setRaw,
		`[ 1,2  ,3]`,
		`  [ 1,2  ] `,
		"-1", `3`)
	testRaw(t, setInt, `[1234]`, ``, `0`, int64(1234))
	testRaw(t, setFloat, `[1234.5]`, ``, `0`, float64(1234.5))
	testRaw(t, setString, `["1234.5"]`, ``, `0`, "1234.5")
	testRaw(t, setBool, `[true]`, ``, `0`, true)
	testRaw(t, setBool, `[null]`, ``, `0`, nil)
	testRaw(t, setString, `{"arr":[1]}`, ``, `arr.-1`, 1)
	testRaw(t, setString, `{"a":"\\"}`, ``, `a`, "\\")
	testRaw(t, setString, `{"a":"C:\\Windows\\System32"}`, ``, `a`, `C:\Windows\System32`)
}

func TestDelete(t *testing.T) {
	testRaw(t, setDelete, `[456]`, `[123,456]`, `0`, nil)
	testRaw(t, setDelete, `[123,789]`, `[123,456,789]`, `1`, nil)
	testRaw(t, setDelete, `[123,456]`, `[123,456,789]`, `-1`, nil)
	testRaw(t, setDelete, `{"a":[123,456]}`, `{"a":[123,456,789]}`, `a.-1`, nil)
	testRaw(t, setDelete, `{"and":"another"}`, `{"this":"that","and":"another"}`, `this`, nil)
	testRaw(t, setDelete, `{"this":"that"}`, `{"this":"that","and":"another"}`, `and`, nil)
	testRaw(t, setDelete, `{}`, `{"and":"another"}`, `and`, nil)
	testRaw(t, setDelete, `{"1":"2"}`, `{"1":"2"}`, `3`, nil)
}

// TestRandomData is a fuzzing test that throws random data at SetRaw
// function looking for panics.
func TestRandomData(t *testing.T) {
	var lstr string
	defer func() {
		if v := recover(); v != nil {
			println("'" + hex.EncodeToString([]byte(lstr)) + "'")
			println("'" + lstr + "'")
			panic(v)
		}
	}()
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 200)
	for i := 0; i < 2000000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		lstr = string(b[:n])
		SetRaw(lstr, "zzzz.zzzz.zzzz", "123")
	}
}

func TestDeleteIssue21(t *testing.T) {
	json := `{"country_code_from":"NZ","country_code_to":"SA","date_created":"2018-09-13T02:56:11.25783Z","date_updated":"2018-09-14T03:15:16.67356Z","disabled":false,"last_edited_by":"Developers","id":"a3e...bc454","merchant_id":"f2b...b91abf","signed_date":"2018-02-01T00:00:00Z","start_date":"2018-03-01T00:00:00Z","url":"https://www.google.com"}`
	res1 := gjson.Get(json, "date_updated")
	var err error
	json, err = Delete(json, "date_updated")
	if err != nil {
		t.Fatal(err)
	}
	res2 := gjson.Get(json, "date_updated")
	res3 := gjson.Get(json, "date_created")
	if !res1.Exists() || res2.Exists() || !res3.Exists() {
		t.Fatal("bad news")
	}

	// We change the number of characters in this to make the section of the string before the section that we want to delete a certain length

	//---------------------------
	lenBeforeToDeleteIs307AsBytes := `{"1":"","0":"012345678901234567890123456789012345678901234567890123456789012345678901234567","to_delete":"0","2":""}`

	expectedForLenBefore307AsBytes := `{"1":"","0":"012345678901234567890123456789012345678901234567890123456789012345678901234567","2":""}`
	//---------------------------

	//---------------------------
	lenBeforeToDeleteIs308AsBytes := `{"1":"","0":"0123456789012345678901234567890123456789012345678901234567890123456789012345678","to_delete":"0","2":""}`

	expectedForLenBefore308AsBytes := `{"1":"","0":"0123456789012345678901234567890123456789012345678901234567890123456789012345678","2":""}`
	//---------------------------

	//---------------------------
	lenBeforeToDeleteIs309AsBytes := `{"1":"","0":"01234567890123456789012345678901234567890123456789012345678901234567890123456","to_delete":"0","2":""}`

	expectedForLenBefore309AsBytes := `{"1":"","0":"01234567890123456789012345678901234567890123456789012345678901234567890123456","2":""}`
	//---------------------------

	var data = []struct {
		desc     string
		input    string
		expected string
	}{
		{
			desc:     "len before \"to_delete\"... = 307",
			input:    lenBeforeToDeleteIs307AsBytes,
			expected: expectedForLenBefore307AsBytes,
		},
		{
			desc:     "len before \"to_delete\"... = 308",
			input:    lenBeforeToDeleteIs308AsBytes,
			expected: expectedForLenBefore308AsBytes,
		},
		{
			desc:     "len before \"to_delete\"... = 309",
			input:    lenBeforeToDeleteIs309AsBytes,
			expected: expectedForLenBefore309AsBytes,
		},
	}

	for i, d := range data {
		result, err := Delete(d.input, "to_delete")

		if err != nil {
			t.Error(fmtErrorf(testError{
				unexpected: "error",
				desc:       d.desc,
				i:          i,
				lenInput:   len(d.input),
				input:      d.input,
				expected:   d.expected,
				result:     result,
			}))
		}
		if result != d.expected {
			t.Error(fmtErrorf(testError{
				unexpected: "result",
				desc:       d.desc,
				i:          i,
				lenInput:   len(d.input),
				input:      d.input,
				expected:   d.expected,
				result:     result,
			}))
		}
	}
}

type testError struct {
	unexpected string
	desc       string
	i          int
	lenInput   int
	input      interface{}
	expected   interface{}
	result     interface{}
}

func fmtErrorf(e testError) string {
	return fmt.Sprintf(
		"Unexpected %s:\n\t"+
			"for=%q\n\t"+
			"i=%d\n\t"+
			"len(input)=%d\n\t"+
			"input=%v\n\t"+
			"expected=%v\n\t"+
			"result=%v",
		e.unexpected, e.desc, e.i, e.lenInput, e.input, e.expected, e.result,
	)
}

func TestSetDotKeyIssue10(t *testing.T) {
	json := `{"app.token":"abc"}`
	json, _ = Set(json, `app\.token`, "cde")
	if json != `{"app.token":"cde"}` {
		t.Fatalf("expected '%v', got '%v'", `{"app.token":"cde"}`, json)
	}
}
func TestDeleteDotKeyIssue19(t *testing.T) {
	json := []byte(`{"data":{"key1":"value1","key2.something":"value2"}}`)
	json, _ = DeleteBytes(json, `data.key2\.something`)
	if string(json) != `{"data":{"key1":"value1"}}` {
		t.Fatalf("expected '%v', got '%v'", `{"data":{"key1":"value1"}}`, json)
	}
}

func TestIssue36(t *testing.T) {
	var json = `
	{
	    "size": 1000
    }
`
	var raw = `
	{
	    "sample": "hello"
	}
`
	_ = raw
	if true {
		json, _ = SetRaw(json, "aggs", raw)
	}
	if !gjson.Valid(json) {
		t.Fatal("invalid json")
	}
	res := gjson.Get(json, "aggs.sample").String()
	if res != "hello" {
		t.Fatal("unexpected result")
	}
}

var example = `
{
	"name": {"first": "Tom", "last": "Anderson"},
	"age":37,
	"children": ["Sara","Alex","Jack"],
	"fav.movie": "Deer Hunter",
	"friends": [
	  {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
	  {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
	  {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
	]
  }
  `

func TestIndex(t *testing.T) {
	path := `friends.#(last="Murphy").last`
	json, err := Set(example, path, "Johnson")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.Get(json, "friends.#.last").String() != `["Johnson","Craig","Murphy"]` {
		t.Fatal("mismatch")
	}
}

func TestIndexes(t *testing.T) {
	path := `friends.#(last="Murphy")#.last`
	json, err := Set(example, path, "Johnson")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.Get(json, "friends.#.last").String() != `["Johnson","Craig","Johnson"]` {
		t.Fatal("mismatch")
	}
}

func TestIssue61(t *testing.T) {
	json := `{
		"@context": {
		  "rdfs": "http://www.w3.org/2000/01/rdf-schema#",
		  "@vocab": "http://schema.org/",
		  "sh": "http://www.w3.org/ns/shacl#"
		}
	}`
	json1, _ := Set(json, "@context.@vocab", "newval")
	if gjson.Get(json1, "@context.@vocab").String() != "newval" {
		t.Fail()
	}
}

// https://github.com/tidwall/sjson/issues/81
func TestNestedWildcards(t *testing.T) {
	json := `{"object1":{"object2":[{"nested_object1":{"nested_object2":[{"nested2_object1":1},{"nested2_object1":1}]}}]}}`
	expected := `{"object1":{"object2":[{"nested_object1":{"nested_object2":[{"nested2_object1":2},{"nested2_object1":2}]}}]}}`

	result, err := Set(json, "object1.object2.#.nested_object1.nested_object2.#.nested2_object1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test case with triple nested # wildcards
	json2 := `{"object1":{"object2":[{"nested_object1":{"nested_object2":[{"nested_object3":[{"nested2_object1":1},{"nested2_object1":1}]}]}}]}}`
	expected2 := `{"object1":{"object2":[{"nested_object1":{"nested_object2":[{"nested_object3":[{"nested2_object1":3},{"nested2_object1":3}]}]}}]}}`

	result2, err := Set(json2, "object1.object2.#.nested_object1.nested_object2.#.nested_object3.#.nested2_object1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}
}

func TestWildcardBehavior(t *testing.T) {
	// Test single # wildcard
	json := `{"users":[{"name":"John","age":30},{"name":"Jane","age":25}]}`
	expected := `{"users":[{"name":"John","age":35},{"name":"Jane","age":35}]}`

	result, err := Set(json, "users.#.age", 35)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test # wildcard with nested objects
	json2 := `{"teams":[{"members":[{"id":1,"active":true},{"id":2,"active":false}]},{"members":[{"id":3,"active":true}]}]}`
	expected2 := `{"teams":[{"members":[{"id":1,"active":false},{"id":2,"active":false}]},{"members":[{"id":3,"active":false}]}]}`

	result2, err := Set(json2, "teams.#.members.#.active", false)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}
}

func TestWildcardEdgeCases(t *testing.T) {
	// Test empty arrays
	json := `{"data":[]}`
	expected := `{"data":[]}`

	result, err := Set(json, "data.#.value", 42)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test nested empty arrays
	json2 := `{"data":[{"items":[]}]}`
	expected2 := `{"data":[{"items":[]}]}`

	result2, err := Set(json2, "data.#.items.#.value", 42)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}

	// Test single element arrays
	json3 := `{"data":[{"items":[{"id":1}]}]}`
	expected3 := `{"data":[{"items":[{"id":1,"value":42}]}]}`

	result3, err := Set(json3, "data.#.items.#.value", 42)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result3) != sortJSON(expected3) {
		t.Fatalf("expected '%v', got '%v'", expected3, result3)
	}
}

func TestWildcardWithDifferentTypes(t *testing.T) {
	// Test with strings
	json := `{"items":[{"name":"old"},{"name":"old"}]}`
	expected := `{"items":[{"name":"new"},{"name":"new"}]}`

	result, err := Set(json, "items.#.name", "new")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test with booleans
	json2 := `{"flags":[{"enabled":true},{"enabled":true}]}`
	expected2 := `{"flags":[{"enabled":false},{"enabled":false}]}`

	result2, err := Set(json2, "flags.#.enabled", false)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}

	// Test with null
	json3 := `{"items":[{"value":1},{"value":2}]}`
	expected3 := `{"items":[{"value":null},{"value":null}]}`

	result3, err := Set(json3, "items.#.value", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result3) != sortJSON(expected3) {
		t.Fatalf("expected '%v', got '%v'", expected3, result3)
	}
}

func TestWildcardWithComplexStructures(t *testing.T) {
	// Test with nested objects and arrays
	json := `{"departments":[{"employees":[{"details":{"salary":50000,"benefits":{"health":true,"dental":false}}}]}]}`
	expected := `{"departments":[{"employees":[{"details":{"salary":50000,"benefits":{"health":false,"dental":false}}}]}]}`

	result, err := Set(json, "departments.#.employees.#.details.benefits.health", false)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test with multiple arrays at same level
	json2 := `{"data":[{"list1":[{"val":1}],"list2":[{"val":2}]}]}`
	expected2 := `{"data":[{"list1":[{"val":99}],"list2":[{"val":2}]}]}`

	result2, err := Set(json2, "data.#.list1.#.val", 99)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}
}

func TestWildcardMixedWithIndexes(t *testing.T) {
	// Test mixing # wildcard with specific indexes
	json := `{"groups":[{"items":[{"id":1},{"id":2}]},{"items":[{"id":3},{"id":4}]}]}`
	expected := `{"groups":[{"items":[{"id":99},{"id":2}]},{"items":[{"id":99},{"id":4}]}]}`

	result, err := Set(json, "groups.#.items.0.id", 99)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test specific index with # wildcard
	json2 := `{"groups":[{"items":[{"id":1},{"id":2}]},{"items":[{"id":3},{"id":4}]}]}`
	expected2 := `{"groups":[{"items":[{"id":1},{"id":2}]},{"items":[{"id":88},{"id":88}]}]}`

	result2, err := Set(json2, "groups.1.items.#.id", 88)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}
}

func TestWildcardDeepNesting(t *testing.T) {
	// Test with very deep nesting
	json := `{"level1":[{"level2":[{"level3":[{"level4":[{"level5":[{"value":1}]}]}]}]}]}`
	expected := `{"level1":[{"level2":[{"level3":[{"level4":[{"level5":[{"value":999}]}]}]}]}]}`

	result, err := Set(json, "level1.#.level2.#.level3.#.level4.#.level5.#.value", 999)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}
}

func TestWildcardWithRawValues(t *testing.T) {
	// Test with raw JSON values
	json := `{"items":[{"config":{"old":"value"}},{"config":{"old":"value"}}]}`
	expected := `{"items":[{"config":{"new":"object","with":"properties"}},{"config":{"new":"object","with":"properties"}}]}`

	result, err := SetRaw(json, "items.#.config", `{"new":"object","with":"properties"}`)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}
}

func TestUserIssueNestedWildcard(t *testing.T) {
	// User's specific test case
	json := `[{"env": [{"name": "AWS_ACCESS_KEY_ID", "value": "test"}]}]`
	expected := `[{"env": [{"name": "AWS_ACCESS_KEY_ID", "value": "newvalue"}]}]`

	result, err := Set(json, "#.env.#.value", "newvalue")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}
}

func TestRootArrayWildcards(t *testing.T) {
	// Test simple root array wildcard
	json := `[{"name":"John"},{"name":"Jane"}]`
	expected := `[{"name":"Updated"},{"name":"Updated"}]`

	result, err := Set(json, "#.name", "Updated")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test root array with nested arrays
	json2 := `[{"items":[{"id":1},{"id":2}]},{"items":[{"id":3}]}]`
	expected2 := `[{"items":[{"id":99},{"id":99}]},{"items":[{"id":99}]}]`

	result2, err := Set(json2, "#.items.#.id", 99)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}

	// Test adding new properties with root array wildcard
	json3 := `[{"name":"John"},{"name":"Jane"}]`
	expected3 := `[{"name":"John","age":30},{"name":"Jane","age":30}]`

	result3, err := Set(json3, "#.age", 30)
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result3) != sortJSON(expected3) {
		t.Fatalf("expected '%v', got '%v'", expected3, result3)
	}
}

func TestWildcardDeletion(t *testing.T) {
	// Test simple wildcard deletion
	json := `{"users":[{"name":"John","age":30},{"name":"Jane","age":25}]}`
	expected := `{"users":[{"name":"John"},{"name":"Jane"}]}`

	result, err := Delete(json, "users.#.age")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test nested wildcard deletion
	json2 := `{"teams":[{"members":[{"id":1,"active":true},{"id":2,"active":false}]},{"members":[{"id":3,"active":true}]}]}`
	expected2 := `{"teams":[{"members":[{"id":1},{"id":2}]},{"members":[{"id":3}]}]}`

	result2, err := Delete(json2, "teams.#.members.#.active")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}

	// Test root array wildcard deletion
	json3 := `[{"name":"John","age":30},{"name":"Jane","age":25}]`
	expected3 := `[{"name":"John"},{"name":"Jane"}]`

	result3, err := Delete(json3, "#.age")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result3) != sortJSON(expected3) {
		t.Fatalf("expected '%v', got '%v'", expected3, result3)
	}

	// Test user's specific case - deletion with nested wildcards
	json4 := `[{"env": [{"name": "AWS_ACCESS_KEY_ID", "value": "test"}]}]`
	expected4 := `[{"env": [{"name": "AWS_ACCESS_KEY_ID"}]}]`

	result4, err := Delete(json4, "#.env.#.value")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result4) != sortJSON(expected4) {
		t.Fatalf("expected '%v', got '%v'", expected4, result4)
	}
}

func TestWildcardDeletionEdgeCases(t *testing.T) {
	// Test deletion from empty arrays
	json := `{"data":[]}`
	expected := `{"data":[]}`

	result, err := Delete(json, "data.#.value")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test deletion of non-existent fields
	json2 := `{"items":[{"name":"test"}]}`
	expected2 := `{"items":[{"name":"test"}]}`

	result2, err := Delete(json2, "items.#.nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}

	// Test deletion with mixed existing and non-existent fields
	json3 := `{"items":[{"name":"test","id":1},{"name":"test2"}]}`
	expected3 := `{"items":[{"name":"test"},{"name":"test2"}]}`

	result3, err := Delete(json3, "items.#.id")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result3) != sortJSON(expected3) {
		t.Fatalf("expected '%v', got '%v'", expected3, result3)
	}

	// Test deep nested deletion
	json4 := `{"level1":[{"level2":[{"level3":[{"level4":[{"level5":[{"value":1,"keep":"this"}]}]}]}]}]}`
	expected4 := `{"level1":[{"level2":[{"level3":[{"level4":[{"level5":[{"keep":"this"}]}]}]}]}]}`

	result4, err := Delete(json4, "level1.#.level2.#.level3.#.level4.#.level5.#.value")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result4) != sortJSON(expected4) {
		t.Fatalf("expected '%v', got '%v'", expected4, result4)
	}
}

func TestWildcardDeletionComplexStructures(t *testing.T) {
	// Test deletion in complex nested structures
	json := `{"departments":[{"employees":[{"details":{"salary":50000,"benefits":{"health":true,"dental":false},"temp":"remove"}}]}]}`
	expected := `{"departments":[{"employees":[{"details":{"salary":50000,"benefits":{"health":true,"dental":false}}}]}]}`

	result, err := Delete(json, "departments.#.employees.#.details.temp")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("expected '%v', got '%v'", expected, result)
	}

	// Test deletion of entire nested objects
	json2 := `{"data":[{"config":{"old":"value","settings":{"a":1,"b":2}},"keep":"this"}]}`
	expected2 := `{"data":[{"config":{"old":"value"},"keep":"this"}]}`

	result2, err := Delete(json2, "data.#.config.settings")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("expected '%v', got '%v'", expected2, result2)
	}
}

func TestUserRequestedWildcardDeletion(t *testing.T) {
	// User's exact scenario: Delete field using #.env.#.value
	json := `[{"env": [{"name": "AWS_ACCESS_KEY_ID", "value": "test"}]}]`
	expected := `[{"env": [{"name": "AWS_ACCESS_KEY_ID"}]}]`

	result, err := Delete(json, "#.env.#.value")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result) != sortJSON(expected) {
		t.Fatalf("User's scenario failed. Expected '%v', got '%v'", expected, result)
	}

	// More complex scenario with multiple nested elements
	json2 := `[{"env": [{"name": "AWS_ACCESS_KEY_ID", "value": "test"}, {"name": "SECRET", "value": "secret"}]}, {"env": [{"name": "API_KEY", "value": "key"}]}]`
	expected2 := `[{"env": [{"name": "AWS_ACCESS_KEY_ID"}, {"name": "SECRET"}]}, {"env": [{"name": "API_KEY"}]}]`

	result2, err := Delete(json2, "#.env.#.value")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result2) != sortJSON(expected2) {
		t.Fatalf("Complex scenario failed. Expected '%v', got '%v'", expected2, result2)
	}

	// Verify deletion with arrays and nested # works for different patterns
	json3 := `{"configs": [{"settings": [{"key": "timeout", "value": 30}, {"key": "retry", "value": 3}]}]}`
	expected3 := `{"configs": [{"settings": [{"key": "timeout"}, {"key": "retry"}]}]}`

	result3, err := Delete(json3, "configs.#.settings.#.value")
	if err != nil {
		t.Fatal(err)
	}
	if sortJSON(result3) != sortJSON(expected3) {
		t.Fatalf("Nested settings scenario failed. Expected '%v', got '%v'", expected3, result3)
	}
}
