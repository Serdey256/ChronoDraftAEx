// Package depsdetect 提供项目依赖自动检测功能
// 通过解析各语言的清单文件，提取项目依赖列表
package depsdetect

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// manifestParser 定义清单文件解析函数类型
type manifestParser func(filePath string) ([]string, error)

// manifestFiles 注册支持的清单文件及其解析器
var manifestFiles = map[string]manifestParser{
	"go.mod":          parseGoMod,
	"package.json":    parsePackageJSON,
	"requirements.txt": parseRequirementsTxt,
	"Cargo.toml":      parseCargoToml,
}

// DetectDependencies 自动检测项目目录中的依赖清单文件并解析依赖列表
func DetectDependencies(projectRoot string) ([]string, error) {
	seen := make(map[string]struct{})

	for filename, parser := range manifestFiles {
		filePath := filepath.Join(projectRoot, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}
		deps, err := parser(filePath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", filename, err)
		}
		for _, dep := range deps {
			seen[dep] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return []string{}, nil
	}

	result := make([]string, 0, len(seen))
	for dep := range seen {
		result = append(result, dep)
	}
	sort.Strings(result)
	return result, nil
}

// parseGoMod 解析 go.mod 文件中的 require 块，跳过 // indirect 依赖
func parseGoMod(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var deps []string
	inRequire := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire && line != "" && !strings.HasSuffix(line, "// indirect") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				deps = append(deps, parts[0]+"@"+parts[1])
			}
		}
	}
	return deps, scanner.Err()
}

// packageJSONDeps 用于解析 package.json 中的依赖字段
type packageJSONDeps struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// parsePackageJSON 解析 package.json 文件中的 dependencies 和 devDependencies
func parsePackageJSON(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var pkg packageJSONDeps
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	var deps []string
	for name, version := range pkg.Dependencies {
		deps = append(deps, name+"@"+version)
	}
	for name, version := range pkg.DevDependencies {
		deps = append(deps, name+"@"+version)
	}
	return deps, nil
}

// parseRequirementsTxt 解析 requirements.txt 文件，跳过注释和选项行
func parseRequirementsTxt(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var deps []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "--") {
			continue
		}

		if idx := strings.Index(line, "=="); idx != -1 {
			deps = append(deps, line[:idx]+"@"+line[idx+2:])
		} else if idx := strings.Index(line, ">="); idx != -1 {
			deps = append(deps, line[:idx]+"@"+line[idx+2:])
		} else {
			deps = append(deps, line)
		}
	}
	return deps, scanner.Err()
}

// parseCargoToml 解析 Cargo.toml 文件中的 [dependencies] 段
func parseCargoToml(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var deps []string
	inDeps := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "[dependencies]" {
			inDeps = true
			continue
		}
		if inDeps && strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}

		if inDeps && line != "" && !strings.HasPrefix(line, "#") {
			eqIdx := strings.Index(line, "=")
			if eqIdx == -1 {
				continue
			}
			name := strings.TrimSpace(line[:eqIdx])
			value := strings.TrimSpace(line[eqIdx+1:])
			value = strings.Trim(value, "\"")
			deps = append(deps, name+"@"+value)
		}
	}
	return deps, scanner.Err()
}
