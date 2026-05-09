package codeanalysis

import (
	"ChronoDraftAEx/pkg/models"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var csharpPatterns = []struct {
	pattern    *regexp.Regexp
	entityType string
	nameIdx    int
}{
	{regexp.MustCompile(`public\s+(static\s+)?(async\s+)?\w+[<>\w]*\s+(\w+)\s*\(`), "function", 3},
	{regexp.MustCompile(`public\s+(static\s+)?class\s+(\w+)`), "class", 2},
	{regexp.MustCompile(`public\s+interface\s+(\w+)`), "interface", 1},
	{regexp.MustCompile(`using\s+([\w.]+)`), "import", 1},
}

func analyzeCSharpFile(filePath string) ([]models.CodeEntity, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败 %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var entities []models.CodeEntity
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		for _, tp := range csharpPatterns {
			matches := tp.pattern.FindStringSubmatch(line)
			if matches == nil {
				continue
			}
			entityName := ""
			if tp.nameIdx > 0 && tp.nameIdx < len(matches) {
				entityName = matches[tp.nameIdx]
			}
			if tp.entityType == "import" {
				entityName = strings.TrimSpace(entityName)
			}
			entities = append(entities, models.CodeEntity{
				FilePath: filePath, EntityType: tp.entityType,
				Name: entityName, Signature: strings.TrimSpace(line),
				Metadata: fmt.Sprintf(`{"line":%d}`, lineNum),
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件失败 %s: %w", filePath, err)
	}
	if entities == nil {
		entities = []models.CodeEntity{}
	}
	return entities, nil
}
