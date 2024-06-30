package main

import (
	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/kisielk/errcheck/errcheck"
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/ast/inspector"
	"honnef.co/go/tools/staticcheck"
	"strings"
)

// ExitAnalizer - анализирует использование функции os.Exit в функции main пакета main.
var ExitAnalizer = &analysis.Analyzer{
	Name: "callExit",
	Doc:  "Анализирует использование os.Exit в функции main пакета main",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

// Это пакет включает в себя следующие анализаторы:
// * inspect.Analyzer - анализирует абстрактное синтаксическое дерево (AST) кода
// * printf.Analyzer - анализирует использование функций форматирования printf
// * shadow.Analyzer - анализирует переопределение переменных внутри блоков
// * shift.Analyzer - анализирует использование сдвига влево/вправо на недопустимое количество битов
// * structtag.Analyzer - анализирует использование некорректных тегов структур
// * errcheck.Analyzer - анализирует необработанные ошибки
func main() {
	var checks []*analysis.Analyzer
	for _, v := range staticcheck.Analyzers {
		if strings.Contains(v.Analyzer.Name, "SA") {
			checks = append(checks, v.Analyzer)
		}
		if v.Analyzer.Name == "ST1001" || v.Analyzer.Name == "QF1007" {
			checks = append(checks, v.Analyzer)
		}
	}

	checks = append(
		checks,
		ExitAnalizer,
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		shift.Analyzer,
		errcheck.Analyzer,
		ineffassign.Analyzer,
	)

	multichecker.Main(
		checks...,
	)
}

func run(pass *analysis.Pass) (interface{}, error) {
	iLocalInspector, ok := pass.ResultOf[inspect.Analyzer]
	if !ok {
		return nil, nil
	}
	localInspector, ok := iLocalInspector.(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	localInspector.Preorder(nodeFilter, func(node ast.Node) {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return
		}
		fun, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		ident, ok := fun.X.(*ast.Ident)
		if !ok || ident.Name != "os" || fun.Sel.Name != "Exit" {
			return
		}
		pass.Reportf(callExpr.Pos(), "прямой вызов os.Exit в функции main пакета main запрещен")
	})

	return nil, nil
}
