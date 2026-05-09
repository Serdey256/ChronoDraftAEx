// Package codeanalysis 提供通用的 AST 代码分析接口
// 支持 Go/TS/Java/Python/C/C++/Rust/C#/Kotlin/JS/Vue 分析
package codeanalysis

import (
	"ChronoDraftAEx/pkg/models"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Analyzer 定义代码分析器接口
// 实现: goanalyzer, tsanalyzer
type Analyzer interface {
	AnalyzeFile(filePath string) ([]models.CodeEntity, error)
}

// 忽略目录和文件列表（与 internal/changedetect 保持一致）
var ignoreList = []string{
	".git", ".chronodraft", "node_modules", "vendor",
	"build", "dist", "target", "out", "bin", "obj",
	"__pycache__", ".next", ".nuxt", ".output",
	".gradle", ".idea", ".vscode", ".vs",
	".DS_Store", "Thumbs.db", "*.log",
}

var ignoreExtensions = []string{".class", ".pyc", ".pyo", ".o", ".so", ".dll", ".exe", ".bin", ".flat", ".dex"}

// shouldIgnore 判断路径是否匹配忽略规则
func shouldIgnore(path string) bool {
	for _, ig := range ignoreList {
		if strings.Contains(path, ig) {
			return true
		}
	}
	return false
}

// shouldIgnoreByExtension 判断文件扩展名是否匹配忽略规则
func shouldIgnoreByExtension(path string) bool {
	ext := filepath.Ext(path)
	for _, ig := range ignoreExtensions {
		if ext == ig {
			return true
		}
	}
	return false
}

// AnalyzeFile 根据文件扩展名分发到对应的分析器
// 支持: .go → goanalyzer, .ts/.tsx → tsanalyzer
func AnalyzeFile(filePath string) ([]models.CodeEntity, error) {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return analyzeGoFile(filePath)
	case ".ts", ".tsx":
		return analyzeTSFile(filePath)
	case ".java":
		return analyzeJavaFile(filePath)
	case ".py":
		return analyzePythonFile(filePath)
	case ".c", ".cpp", ".cxx", ".cc", ".h", ".hpp", ".hxx":
		return analyzeCFile(filePath)
	case ".rs":
		return analyzeRustFile(filePath)
	case ".cs":
		return analyzeCSharpFile(filePath)
	case ".kt", ".kts":
		return analyzeKotlinFile(filePath)
	case ".js", ".jsx":
		return analyzeJSFile(filePath)
	case ".vue":
		return analyzeVueFile(filePath)
	default:
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}
}

// AnalyzeProject 遍历项目目录，分析所有支持的代码文件
// projectRoot: 项目根目录
// saveFn: 保存回调，参数为 (filePath, entities)
func AnalyzeProject(projectRoot string, saveFn func(string, []models.CodeEntity) error) error {
	return filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if shouldIgnore(relPath) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldIgnore(relPath) || shouldIgnoreByExtension(path) {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".java" && ext != ".py" && ext != ".c" && ext != ".cpp" && ext != ".cxx" && ext != ".cc" && ext != ".h" && ext != ".hpp" && ext != ".hxx" && ext != ".rs" && ext != ".cs" && ext != ".kt" && ext != ".kts" && ext != ".js" && ext != ".jsx" && ext != ".vue" {
			return nil
		}

		entities, err := AnalyzeFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "分析文件失败 %s: %v\n", path, err)
			return nil
		}

		if len(entities) > 0 && saveFn != nil {
			// Normalize to relative path for consistent DB queries
			for i := range entities {
				entities[i].FilePath = relPath
			}
			if err := saveFn(relPath, entities); err != nil {
				fmt.Fprintf(os.Stderr, "保存代码实体失败 %s: %v\n", path, err)
			}
		}

		return nil
	})
}
