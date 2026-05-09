package codeanalysis

import (
	"ChronoDraftAEx/pkg/models"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// exprString 将 ast.Expr 转换为类型字符串
func exprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len != nil {
			return "[" + exprString(t.Len) + "]" + exprString(t.Elt)
		}
		return "[]" + exprString(t.Elt)
	case *ast.Ellipsis:
		return "..." + exprString(t.Elt)
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.InterfaceType:
		return "interface{...}"
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + exprString(t.Value)
	case *ast.BasicLit:
		return t.Value
	case *ast.ParenExpr:
		return "(" + exprString(t.X) + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// fieldListString 将 *ast.FieldList 转换为参数字符串
// sep: 分隔符（如 ", "）
func fieldListString(fl *ast.FieldList, sep string) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fl.List {
		typeStr := exprString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typeStr)
		} else {
			for _, n := range f.Names {
				parts = append(parts, n.Name+" "+typeStr)
			}
		}
	}
	return strings.Join(parts, sep)
}

// analyzeGoFile 分析单个 Go 源文件，提取代码实体
// 提取：导出函数/方法、导出结构体/接口、导入语句
// 跳过：非导出函数/类型、变量、常量
func analyzeGoFile(filePath string) ([]models.CodeEntity, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析 Go 文件失败 %s: %w", filePath, err)
	}

	pkgName := ""
	if f.Name != nil {
		pkgName = f.Name.Name
	}

	var entities []models.CodeEntity

	// 遍历 AST
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if !ast.IsExported(node.Name.Name) {
				return true
			}

			// 构建签名: func Name(params) (returns) 或 func (r Receiver) Name(params) (returns)
			sig := "func "

			// 接收者（方法）
			recvStr := ""
			if node.Recv != nil && len(node.Recv.List) > 0 {
				r := node.Recv.List[0]
				recvType := exprString(r.Type)
				if len(r.Names) > 0 {
					recvStr = "(" + r.Names[0].Name + " " + recvType + ") "
				} else {
					recvStr = "(" + recvType + ") "
				}
			}

			sig += recvStr + node.Name.Name

			// 参数
			sig += "(" + fieldListString(node.Type.Params, ", ") + ")"

			// 返回值
			if node.Type.Results != nil && len(node.Type.Results.List) > 0 {
				returnStr := fieldListString(node.Type.Results, ", ")
				if len(node.Type.Results.List) == 1 && len(node.Type.Results.List[0].Names) == 0 {
					sig += " " + returnStr
				} else {
					sig += " (" + returnStr + ")"
				}
			}

			meta := fmt.Sprintf(`{"package":"%s"}`, pkgName)
			if recvStr != "" {
				meta = fmt.Sprintf(`{"package":"%s","receiver":"%s"}`, pkgName, strings.TrimSpace(recvStr))
			}

			entities = append(entities, models.CodeEntity{
				FilePath:   filePath,
				EntityType: "function",
				Name:       node.Name.Name,
				Signature:  sig,
				Metadata:   meta,
			})

		case *ast.GenDecl:
			if node.Tok != token.TYPE {
				return true
			}
			for _, spec := range node.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ast.IsExported(ts.Name.Name) {
					continue
				}

				var typeKind string
				switch ts.Type.(type) {
				case *ast.StructType:
					typeKind = "struct"
				case *ast.InterfaceType:
					typeKind = "interface"
				default:
					continue
				}

				sig := fmt.Sprintf("type %s %s{...}", ts.Name.Name, typeKind)

				entities = append(entities, models.CodeEntity{
					FilePath:   filePath,
					EntityType: typeKind,
					Name:       ts.Name.Name,
					Signature:  sig,
					Metadata:   fmt.Sprintf(`{"package":"%s","kind":"%s"}`, pkgName, typeKind),
				})
			}
		}
		return true
	})

	// 提取导入语句
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		sig := "import"
		if imp.Name != nil {
			sig += " " + imp.Name.Name
		}
		sig += " " + imp.Path.Value

		entities = append(entities, models.CodeEntity{
			FilePath:   filePath,
			EntityType: "import",
			Name:       path,
			Signature:  sig,
			Metadata:   fmt.Sprintf(`{"package":"%s"}`, pkgName),
		})
	}

	return entities, nil
}
