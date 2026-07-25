package etc_test

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/BurntSushi/toml"
)

func TestConfigExampleCoversEtcKeys(t *testing.T) {
	root := repositoryRoot(t)
	examplePath := filepath.Join(root, "testdata", "etc", "etc.toml")

	var example map[string]any
	if _, err := toml.DecodeFile(examplePath, &example); err != nil {
		t.Fatalf("decode config example: %v", err)
	}

	covered := make(map[string]struct{})
	flattenConfigKeys("", example, covered)

	keys := sourceEtcKeys(t, root)
	missing := make([]string, 0)
	for key := range keys {
		if strings.HasPrefix(key, "clusterDemo.") {
			continue
		}
		if _, ok := covered[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("testdata/etc/etc.toml is missing configuration examples: %s", strings.Join(missing, ", "))
	}
}

func TestConfigExampleHasChineseComments(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "testdata", "etc", "etc.toml")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open config example: %v", err)
	}
	defer file.Close()

	var previous string
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		current := strings.TrimSpace(scanner.Text())
		if current == "" {
			continue
		}
		if isConfigEntry(current) && !isChineseComment(previous) {
			t.Errorf("testdata/etc/etc.toml:%d configuration entry %q is missing a Chinese comment", line, current)
		}
		previous = current
	}
	if err = scanner.Err(); err != nil {
		t.Fatalf("scan config example: %v", err)
	}
}

func isConfigEntry(line string) bool {
	return strings.HasPrefix(line, "[") || (!strings.HasPrefix(line, "#") && strings.Contains(line, "="))
}

func isChineseComment(line string) bool {
	if !strings.HasPrefix(line, "#") {
		return false
	}
	for _, r := range line {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
}

func sourceEtcKeys(t *testing.T, root string) map[string]struct{} {
	t.Helper()

	keys := make(map[string]struct{})
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, unquoteErr := strconv.Unquote(literal.Value)
			if unquoteErr == nil && strings.HasPrefix(value, "etc.") {
				keys[strings.TrimPrefix(value, "etc.")] = struct{}{}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("scan source configuration keys: %v", err)
	}
	return keys
}

func flattenConfigKeys(prefix string, values map[string]any, keys map[string]struct{}) {
	for key, value := range values {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		keys[path] = struct{}{}
		if children, ok := value.(map[string]any); ok {
			flattenConfigKeys(path, children, keys)
		}
	}
}
