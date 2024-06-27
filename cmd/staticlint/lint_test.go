package main

import (
	"golang.org/x/tools/go/analysis/analysistest"
	"testing"
)

func TestExitAnalizer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ExitAnalizer, "./...")
}
