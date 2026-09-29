package main

import (
	"reflect"
	"testing"
)

func TestBuildArgsKeepsTextAsOneLiteralArgument(t *testing.T) {
	text := `WBS{d08}- $() "literal"`
	got := buildArgs("./stamp.toml", "/input drawings", "/output", text, true)
	want := []string{
		"--config", "./stamp.toml",
		"--input", "/input drawings",
		"--output", "/output",
		"--shrink",
		"--text", text,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildArgsOmitsEmptyTextOverride(t *testing.T) {
	got := buildArgs("./stamp.toml", "/input", "/output", "", false)
	want := []string{"--config", "./stamp.toml", "--input", "/input", "--output", "/output"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildArgs() = %#v, want %#v", got, want)
	}
}
