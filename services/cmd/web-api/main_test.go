package main

import (
	"reflect"
	"testing"
)

func TestSplitOriginsTrimsAndDropsEmptyEntries(t *testing.T) {
	want := []string{"https://one.example", "https://two.example"}
	if got := splitOrigins(" https://one.example, ,https://two.example  ,"); !reflect.DeepEqual(got, want) {
		t.Fatalf("splitOrigins = %#v", got)
	}
}
