package report

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestMessageTemplatesExcludeProhibitedLanguage(t *testing.T) {
	t.Parallel()

	templates := detectorMessageTemplates(t)
	templates = append(templates, model.ReportDisclaimer)
	prohibited := []string{
		"exploit",
		"malicious",
		"attack",
		"hack",
		"rug",
		"scam",
		"illegal",
		"non-compliant",
		"safe",
		"unsafe",
		"secure",
		"insecure",
	}
	for _, template := range templates {
		lower := strings.ToLower(template)
		for _, word := range prohibited {
			if strings.Contains(lower, word) {
				t.Errorf("message template contains prohibited word %q: %q", word, template)
			}
		}
	}
}

func detectorMessageTemplates(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("../diff")
	if err != nil {
		t.Fatalf("read detector package: %v", err)
	}
	templates := []string{}
	files := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(files, filepath.Join("../diff", entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("parse detector file %s: %v", entry.Name(), err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			pair, ok := node.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := pair.Key.(*ast.Ident)
			if !ok || (key.Name != "Explanation" && key.Name != "NextStep") {
				return true
			}
			literal, ok := pair.Value.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Fatalf("%s message is not a string literal", key.Name)
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatalf("decode %s message: %v", key.Name, err)
			}
			templates = append(templates, value)
			return true
		})
	}
	if len(templates) < 24 {
		t.Fatalf("found %d detector messages, want at least 24", len(templates))
	}
	return templates
}
