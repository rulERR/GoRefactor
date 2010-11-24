package program

import (
	"container/vector"
	"st"
	"os"
	"utils/utils"
	"path"
	"packageParser"
	"go/parser"
	"bufio"
	//"go/ast"
	"strings"
)
import "fmt"


var program *Program
var externPackageTrees *vector.StringVector // [dir][packagename]package
var goSrcDir string
var specificFiles map[string]*vector.StringVector
var specificFilesPackages []string = []string{"syscall", "os"}

func initialize() {

	for _, s := range st.PredeclaredTypes {
		program.BaseSymbolTable.AddSymbol(s)
	}
	for _, s := range st.PredeclaredFunctions {
		program.BaseSymbolTable.AddSymbol(s)
	}
	for _, s := range st.PredeclaredConsts {
		program.BaseSymbolTable.AddSymbol(s)
	}

	goRoot := os.Getenv("GOROOT")
	goSrcDir = path.Join(goRoot, "src", "pkg")

	externPackageTrees = new(vector.StringVector)
	externPackageTrees.Push(goSrcDir)
	externPackageTrees.Push("/home/rulerr/GoRefactor/src") // for tests on self

	specificFiles = make(map[string]*vector.StringVector)

}

type Program struct {
	BaseSymbolTable *st.SymbolTable        //Base sT for parsing any package. Contains basic language symbols
	Packages        map[string]*st.Package //map[qualifiedPath] package
}

func loadConfig(packageName string) *vector.StringVector {
	fd, err := os.Open(packageName+".cfg", os.O_RDONLY, 0)
	if err != nil {
		println(err.String())
		panic("Couldn't open " + packageName + " config")
	}
	defer fd.Close()

	res := new(vector.StringVector)

	reader := bufio.NewReader(fd)
	for {

		str, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		res.Push(str[:len(str)-1])

	}
	fmt.Printf("%s:\n%v\n", packageName, res)

	return res
}

func isPackageDir(fileInIt *os.FileInfo) bool {
	return !fileInIt.IsDirectory() && utils.IsGoFile(fileInIt.Name)
}

func makeFilter(srcDir string) func(f *os.FileInfo) bool {
	_, d := path.Split(srcDir)
	println("^&*^&* Specific files for " + d)
	if files, ok := specificFiles[d]; ok {
		println("^&*^&* found " + d)
		return func(f *os.FileInfo) bool {
			print("\n" + f.Name)
			for _, fName := range *files {
				print(" " + fName)
				if fName == f.Name {
					return true
				}
			}
			return false
		}
	}
	return utils.GoFilter

}
func parsePack(srcDir string) {

	packs, _ := parser.ParseDir(srcDir, makeFilter(srcDir), parser.ParseComments)

	_, d := path.Split(srcDir)
	if packTree, ok := packs[d]; !ok {
		fmt.Printf("Couldn't find a package " + d + " in " + d + " directory\n")
		return
	} else {
		pack := st.NewPackage(srcDir, packTree)
		program.Packages[srcDir] = pack
	}
}

func locatePackages(srcDir string) {

	fd, err := os.Open(srcDir, os.O_RDONLY, 0)
	if err != nil {
		panic("Couldn't open src directory")
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		panic("Couldn't read src directory")
	}

	for i := 0; i < len(list); i++ {
		d := &list[i]
		if isPackageDir(d) { //current dir describes a package
			parsePack(srcDir)
			return
		}
	}

	//no package in this dir, look inside dirs' dirs
	for i := 0; i < len(list); i++ {
		d := &list[i]
		if d.IsDirectory() { //can still contain packages inside
			locatePackages(path.Join(srcDir, d.Name))
		}
	}

}

func ParseProgram(srcDir string) *Program {

	program = &Program{st.NewSymbolTable(nil), make(map[string]*st.Package)}

	initialize()

	for _, pName := range specificFilesPackages {
		specificFiles[pName] = loadConfig(pName)
	}

	locatePackages(srcDir)

	packs := new(vector.Vector)
	for _, pack := range program.Packages {
		packs.Push(pack)
	}

	// Recursively fills program.Packages map.
	for _, ppack := range *packs {
		pack := ppack.(*st.Package)
		parseImports(pack)
	}

	for _, pack := range program.Packages {
		if IsGoSrcPackage(pack) {
			pack.IsGoPackage = true
			//ast.PackageExports(pack.AstPackage)
		}
	}

	for _, pack := range program.Packages {

		pack.Symbols.AddOpenedScope(program.BaseSymbolTable)
		go packageParser.ParsePackage(pack)
	}
	for _, pack := range program.Packages {
		<-pack.Communication
	}
	// type resolving
	for _, pack := range program.Packages {
		<-pack.Communication
	}
	for _, pack := range program.Packages {
		pack.Communication <- 0
	}
	for _, pack := range program.Packages {
		<-pack.Communication
	}
	fmt.Printf("===================All packages stopped fixing \n")

	for _, pack := range program.Packages {
		pack.Communication <- 0
	}

	for _, pack := range program.Packages {
		<-pack.Communication
	}
	fmt.Printf("===================All packages stopped opening \n")

	for _, pack := range program.Packages {
		pack.Communication <- 0
	}

	for _, pack := range program.Packages {
		<-pack.Communication
	}
	fmt.Printf("===================All packages stopped parsing globals \n")
	for _, pack := range program.Packages {
		pack.Communication <- 0
	}

	for _, pack := range program.Packages {
		<-pack.Communication
	}
	fmt.Printf("===================All packages stopped fixing globals \n")
	return program
}

func IsGoSrcPackage(p *st.Package) bool {
	//fmt.Printf("IS GO? %s %s\n", p.QualifiedPath,goSrcDir)
	return strings.HasPrefix(p.QualifiedPath, goSrcDir)
}

func (p *Program) renameSymbol(sym st.Symbol, newName string) int {
	sym.Object().Name = newName

	/*for _,pos := range *sym.Positions() {
		obj := pos.(symbolTable.Occurence).Obj
		obj.Name = newName
	}*/

	return sym.Positions().Len()
}
