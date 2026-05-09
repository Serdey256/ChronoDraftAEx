package codeanalysis

import (
	"ChronoDraftAEx/pkg/models"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TS 分析正则表达式模式
var tsPatterns = []struct {
	pattern    *regexp.Regexp
	entityType string
	nameIdx    int // 捕获组索引，-1 表示需要特殊处理
}{
	{regexp.MustCompile(`export\s+(async\s+)?function\s+(\w+)`), "function", 2},
	{regexp.MustCompile(`export\s+(abstract\s+)?class\s+(\w+)`), "class", 2},
	{regexp.MustCompile(`export\s+interface\s+(\w+)`), "interface", 1},
	{regexp.MustCompile(`export\s+const\s+(\w+)`), "const", 1},
	{regexp.MustCompile(`import\s+.+\s+from\s+['"](.+)['"]`), "import", 1},
	{regexp.MustCompile(`import\s+['"](.+)['"]`), "import", 1}, // 纯路径导入: import "./styles.css"
}

// analyzeTSFile 分析单个 TypeScript 源文件，提取代码实体
// 使用正则表达式提取 export 声明和 import 语句
func analyzeTSFile(filePath string) ([]models.CodeEntity, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 TS 文件失败 %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// 增大扫描缓冲区以处理长行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var entities []models.CodeEntity
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		for _, tp := range tsPatterns {
			matches := tp.pattern.FindStringSubmatch(line)
			if matches == nil {
				continue
			}

			entityName := ""
			if tp.nameIdx > 0 && tp.nameIdx < len(matches) {
				entityName = matches[tp.nameIdx]
			}

			// 对于 import，name 是导入路径
			if tp.entityType == "import" {
				entityName = strings.Trim(matches[1], "\"'")
			}

			sig := strings.TrimSpace(line)

			entities = append(entities, models.CodeEntity{
				FilePath:   filePath,
				EntityType: tp.entityType,
				Name:       entityName,
				Signature:  sig,
				Metadata:   fmt.Sprintf(`{"line":%d}`, lineNum),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 TS 文件失败 %s: %w", filePath, err)
	}

	if entities == nil {
		return []models.CodeEntity{}, nil
	}

	return entities, nil
}
