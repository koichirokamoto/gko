package main

import (
	"path/filepath"
	"testing"
)

func TestGenerateModels(t *testing.T) {
	path, err := filepath.Abs("testdata/test.yaml")
	if err != nil {
		t.Fatal(err)
	}

	s, err := loadSwaggerSpec(path)
	if err != nil {
		t.Fatal(err)
	}

	var g generator
	err = g.generateModels(s)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(g.buf.String())
}
