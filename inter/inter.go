package inter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
)

/*
FindInterfaces 根据文件查找所有接口
*/
func FindInterfaces(path string) {
	// 创建一个新的标记集合
	fset := token.NewFileSet()

	// 通过ParseDir函数解析目录下的Go源文件
	files, err := parser.ParseDir(fset, path, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		log.Println("Error parsing directory:", err)
		return
	}

	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.InterfaceType:
				// 打印接口名称及位置信息
				//fmt.Printf("%s\n", x.Name.tString())
				log.Println(x)
				pos := fset.PositionFor(x.Pos(), false)
				log.Printf("\t%s:%d\n", pos.Filename, pos.Line)
			default:
			}
			return true
		})
	}
}
