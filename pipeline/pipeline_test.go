package pipeline

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args []string
		exp  []Process
	}{
		{[]string{"echo", "hi"}, []Process{{"echo", []string{"hi"}}}},
		{[]string{"a", ";;", "b", "x"}, []Process{{"a", []string{}}, {"b", []string{"x"}}}},
		{[]string{";;", "b"}, []Process{{"b", []string{}}}},
		{[]string{"a", ";;", ";;", "b"}, []Process{{"a", []string{}}, {"b", []string{}}}},
		{[]string{"a", ";;"}, []Process{{"a", []string{}}}},
	}
	for _, test := range tests {
		got := ParseArgs(test.args)
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("ParseArgs(%q) = %v, expected %v", test.args, got, test.exp)
		}
	}
}
