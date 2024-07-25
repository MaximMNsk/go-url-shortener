package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExitAnalizer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ExitAnalizer, "./...")
}

func Test_Main(t *testing.T) {
	// go run -ldflags "-X main. "
	//main()
}
