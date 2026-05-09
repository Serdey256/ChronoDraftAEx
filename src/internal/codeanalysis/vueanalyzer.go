package codeanalysis

import (
	"ChronoDraftAEx/pkg/models"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var vueScriptPatterns = []struct {
	pattern    *regexp.Regexp
	entityType string
	nameIdx    int
}{
	{regexp.MustCompile(`export\s+(async\s+)?function\s+(\w+)`), "function", 2},
	{regexp.MustCompile(`export\s+class\s+(\w+)`), "class", 1},
	{regexp.MustCompile(`import\s+.+\s+from\s+['"](.+)['"]`), "import", 1},
	{regexp.MustCompile(`import\s+['"](.+)['"]`), "import", 1},
	{regexp.MustCompile(`const\s+(\w+)\s*=\s*defineComponent`), "component", 1},
	{regexp.MustCompile(`const\s+(\w+)\s*=\s*\(`), "function", 1},
}

func analyzeVueFile(filePath string) ([]models.CodeEntity, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取 Vue 文件失败 %s: %w", filePath, err)
	}

	// Extract <script> section
	scriptRe := regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`)
	matches := scriptRe.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return []models.CodeEntity{}, nil // no script section
	}
	scriptContent := matches[1]

	scanner := bufio.NewScanner(strings.NewReader(scriptContent))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var entities []models.CodeEntity
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		for _, tp := range vueScriptPatterns {
			m := tp.pattern.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			entityName := ""
			if tp.nameIdx > 0 && tp.nameIdx < len(m) {
				entityName = m[tp.nameIdx]
			}
			if tp.entityType == "import" {
				entityName = strings.Trim(m[1], "\"'")
			}
			entities = append(entities, models.CodeEntity{
				FilePath: filePath, EntityType: tp.entityType,
				Name: entityName, Signature: strings.TrimSpace(line),
				Metadata: fmt.Sprintf(`{"line":%d}`, lineNum),
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 Vue script 失败: %w", err)
	}
	if entities == nil {
		entities = []models.CodeEntity{}
	}
	return entities, nil
}
