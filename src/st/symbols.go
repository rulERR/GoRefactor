package st
//Package contains definitions of various symbol types, descripting GO entities

import (
	"go/ast"
	"go/token"
	"container/vector"
)
import "strconv"
//import "fmt"

func init() {

	PredeclaredTypes = make(map[string]*BasicTypeSymbol)
	for _, s := range ast.BasicTypes {
		PredeclaredTypes[s] = MakeBasicType(s, nil)
	}

	PredeclaredConsts = make(map[string]*VariableSymbol)

	b := PredeclaredTypes["bool"]
	n := MakeInterfaceType(NO_NAME, nil)

	PredeclaredConsts["true"] = MakeVariable("true", nil, b)
	PredeclaredConsts["false"] = MakeVariable("false", nil, b)
	PredeclaredConsts["nil"] = MakeVariable("nil", nil, n)
	PredeclaredConsts["iota"] = MakeVariable("iota", nil, PredeclaredTypes["int"])

	//make,new,cmoplex,imag,real,append - in concrete occasion
	//print, println - nothing interesting

	capFts := MakeFunctionType(NO_NAME, nil)
	capFts.Parameters.addSymbol(MakeVariable("_", nil, MakeInterfaceType(NO_NAME, nil)))
	capFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["int"]))

	closeFts := MakeFunctionType(NO_NAME, nil)
	closeFts.Parameters.addSymbol(MakeVariable("_", nil, MakeChannelType(NO_NAME, nil, nil, ast.SEND|ast.RECV)))

	closedFts := MakeFunctionType(NO_NAME, nil)
	closedFts.Parameters.addSymbol(MakeVariable("_", nil, MakeChannelType(NO_NAME, nil, nil, ast.SEND|ast.RECV)))
	closedFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["bool"]))

	copyFts := MakeFunctionType(NO_NAME, nil)
	copyFts.Parameters.addSymbol(MakeVariable("p1", nil, MakeArrayType(NO_NAME, nil, nil, 0)))
	copyFts.Parameters.addSymbol(MakeVariable("p2", nil, MakeArrayType(NO_NAME, nil, nil, 0)))
	copyFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["int"]))

	panicFts := MakeFunctionType(NO_NAME, nil)
	panicFts.Parameters.addSymbol(MakeVariable("_", nil, MakeInterfaceType(NO_NAME, nil)))

	recoverFts := MakeFunctionType(NO_NAME, nil)
	recoverFts.Results.addSymbol(MakeVariable("_", nil, MakeInterfaceType(NO_NAME, nil)))

	lenFts := MakeFunctionType(NO_NAME, nil)
	lenFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["int"]))

	noResultsFts := MakeFunctionType(NO_NAME, nil)

	predeclaredFunctionTypes = make(map[string]*FunctionTypeSymbol)
	predeclaredFunctionTypes["cap"] = capFts
	predeclaredFunctionTypes["close"] = closeFts
	predeclaredFunctionTypes["closed"] = closedFts
	predeclaredFunctionTypes["copy"] = copyFts
	predeclaredFunctionTypes["panic"] = panicFts
	predeclaredFunctionTypes["recover"] = recoverFts
	predeclaredFunctionTypes["print"] = noResultsFts
	predeclaredFunctionTypes["println"] = noResultsFts
	predeclaredFunctionTypes["complex"] = noResultsFts
	predeclaredFunctionTypes["imag"] = noResultsFts
	predeclaredFunctionTypes["len"] = lenFts
	predeclaredFunctionTypes["make"] = noResultsFts
	predeclaredFunctionTypes["new"] = noResultsFts
	predeclaredFunctionTypes["real"] = noResultsFts
	predeclaredFunctionTypes["append"] = noResultsFts

	PredeclaredFunctions = make(map[string]*FunctionSymbol)
	for _, s := range builtIn {
		PredeclaredFunctions[s] = MakeFunction(s, nil, predeclaredFunctionTypes[s])
	}

}

var builtIn []string = []string{"cap", "close", "closed", "complex", "copy", "imag", "len", "make", "new", "panic",
	"print", "println", "real", "recover", "append"}
var integerTypes map[string]bool = map[string]bool{"uintptr": true, "byte": true, "int8": true, "int16": true, "int32": true, "int64": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "int": true, "uint": true}
var floatTypes map[string]bool = map[string]bool{"float32": true, "float64": true}
var complexTypes map[string]bool = map[string]bool{"complex64": true, "complex128": true}

var PredeclaredTypes map[string]*BasicTypeSymbol
var PredeclaredFunctions map[string]*FunctionSymbol
var PredeclaredConsts map[string]*VariableSymbol
var predeclaredFunctionTypes map[string]*FunctionTypeSymbol


/* Interfaces */
//Main interface, implemented by every symbol
type Symbol interface {
	Positions() PositionSet
	Identifiers() IdentSet //Set of Idents, representing symbol
	AddIdent(*ast.Ident)
	SetName(name string)
	Name() string
	AddPosition(token.Position)
	HasPosition(token.Position) bool
	PackageFrom() *Package
	Scope() *SymbolTable
}

type IdentifierMap map[*ast.Ident]Symbol

func (im IdentifierMap) AddIdent(ident *ast.Ident, sym Symbol) {
	im[ident] = sym
}
func (im IdentifierMap) GetSymbol(ident *ast.Ident) Symbol {
	s, ok := im[ident]
	if !ok {
		panic("untracked ident " + ident.Name)
	}
	return s
}

type IdentSet map[*ast.Ident]bool

func NewIdentSet() IdentSet {
	return make(map[*ast.Ident]bool)
}
func (set IdentSet) AddIdent(ident *ast.Ident) {
	set[ident] = true
}

type PositionSet map[string]token.Position

func (set PositionSet) AddPosition(p token.Position) {
	set[makePositionKey(p)] = p
}
func NewPositionSet() PositionSet {
	return make(map[string]token.Position)
}
//Interface for symbols that describe types
type ITypeSymbol interface {
	Symbol //Every type symbol is a symbol
	SetMethods(*SymbolTable)
	Methods() *SymbolTable //Returns type's methods
	AddMethod(meth Symbol) //Adds a method to the type symbol
	ToAstExpr(pack *Package, filename string) ast.Expr
}

/***TypeSymbols. Implement ITypeSymbol***/

//Common part (base) embedded in any type symbol
type TypeSymbol struct {
	name   string
	Idents IdentSet //Corresponding program entity such as a package, constant, type, variable, or function/method
	//List of type's methods
	Meths  *SymbolTable
	Posits PositionSet
	Scope_ *SymbolTable
}
//Dummy type symbol, used in forward declarations during first pass
type UnresolvedTypeSymbol struct {
	*TypeSymbol
	Declaration ast.Expr
}

//Basic type symbol
type BasicTypeSymbol struct {
	*TypeSymbol
}

//Pseudonim type symbol
type AliasTypeSymbol struct {
	*TypeSymbol
	BaseType ITypeSymbol
}

const (
	SLICE    = -1
	ELLIPSIS = -2
)

//Array type symbol
type ArrayTypeSymbol struct {
	*TypeSymbol
	ElemType ITypeSymbol
	Len      int
}
//Channel type symbol
type ChanTypeSymbol struct {
	*TypeSymbol
	Dir       ast.ChanDir
	ValueType ITypeSymbol
}
//Function type symbol
type FunctionTypeSymbol struct {
	*TypeSymbol
	Parameters *SymbolTable //(incoming) parameters
	Results    *SymbolTable //(outgoing) results
	Reciever   *SymbolTable //Reciever (if empty - function)
}
//Interface type symbol
type InterfaceTypeSymbol struct {
	*TypeSymbol //Interface methods stored in base.Methods
}
//Map type symbol
type MapTypeSymbol struct {
	*TypeSymbol
	KeyType   ITypeSymbol
	ValueType ITypeSymbol
}
//Pointer type symbol
type PointerTypeSymbol struct {
	*TypeSymbol
	BaseType ITypeSymbol
	Fields   *SymbolTable
}
//Struct type symbol
type StructTypeSymbol struct {
	*TypeSymbol
	Fields *SymbolTable
}


/***Other Symbols. Implement Symbol***/

type PackageSymbol struct {
	name      string
	Idents    IdentSet
	ShortPath string       // "go/ast", "fmt" etc.
	Posits    PositionSet  //local file name occurances
	Package   *Package     //package entity that's described by symbol
	Scope_    *SymbolTable //scope where symbol is declared
}

type Package struct { //NOT A SYMBOL
	QualifiedPath string //full filesystem path to package src folder

	Symbols         *SymbolTable   //top level declarations
	SymbolTablePool *vector.Vector //links to all symbol tables including nested

	FileSet     *token.FileSet
	AstPackage  *ast.Package              //ast tree
	Imports     map[string]*vector.Vector //map[file] *[]packageSymbol
	IsGoPackage bool                      //true if package source is in $GOROOT/src/pkg/

	Communication chan int
}

func NewPackage(qualifiedPath string, fileSet *token.FileSet, astPackage *ast.Package) *Package {
	p := &Package{QualifiedPath: qualifiedPath, FileSet: fileSet, AstPackage: astPackage}

	p.Symbols = NewSymbolTable(p)
	p.SymbolTablePool = new(vector.Vector)
	p.SymbolTablePool.Push(p.Symbols)
	p.Imports = make(map[string]*vector.Vector)
	p.Communication = make(chan int)
	return p
}
func (pack *Package) GetImport(filename string, imported *Package) *PackageSymbol {
	imps := pack.Imports[filename]
	for _, el := range *imps {
		imp := el.(*PackageSymbol)
		if imp.Package == imported {
			return imp
		}
	}
	return nil
}
//Symbol represents a function or a method
type FunctionSymbol struct {
	name              string
	Idents            IdentSet
	FunctionType      ITypeSymbol  //FunctionTypeSymbol or Alias
	Locals            *SymbolTable //Local variables
	Posits            PositionSet
	Scope_            *SymbolTable
	IsInterfaceMethod bool
}

//Symbol Represents a variable
type VariableSymbol struct {
	name            string
	Idents          IdentSet
	VariableType    ITypeSymbol
	IsTypeSwitchVar bool
	Posits          PositionSet
	Scope_          *SymbolTable
}


/*^^Other Symbol Methods^^*/
const NO_NAME string = ""
const UNNAMED_PREFIX string = "$"

func (s *TypeSymbol) Name() string { return s.name }

func (s *BasicTypeSymbol) Name() string { return s.name }

func (s *PointerTypeSymbol) Name() string {

	ss := "*"
	return ss + s.BaseType.Name()
}
func (s *PointerTypeSymbol) BaseName() string {

	ss := s.BaseType.Name()
	for ss[0] == '*' {
		ss = ss[1:]
	}
	return ss
}

func (s *PackageSymbol) Name() string { return s.name }

func (s *VariableSymbol) Name() string { return s.name }

func (s *FunctionSymbol) Name() string { return s.name }

func (s *ArrayTypeSymbol) Name() string { return s.name }

func (s *ChanTypeSymbol) Name() string { return s.name }

func (s *FunctionTypeSymbol) Name() string { return s.name }

func (s *InterfaceTypeSymbol) Name() string { return s.name }

func (s *MapTypeSymbol) Name() string { return s.name }

func (s *StructTypeSymbol) Name() string { return s.name }


func makePositionKey(pos token.Position) string {
	return pos.Filename + ": " + strconv.Itoa(pos.Line) + "," + strconv.Itoa(pos.Column)
}

func hasPosition(sym Symbol, pos token.Position) bool {
	if _, ok := sym.Positions()[makePositionKey(pos)]; ok {
		return true
	}
	return false
}

func (ts *TypeSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}
func (ts *PackageSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}
func (ts *FunctionSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}
func (ts *VariableSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}

func (ts *TypeSymbol) PackageFrom() *Package {
	if ts.Scope_ != nil {
		return ts.Scope_.Package
	}
	return nil
}
func (ps *PackageSymbol) PackageFrom() *Package {
	if ps.Scope_ != nil {
		return ps.Scope_.Package
	}
	return nil
}
func (fs *FunctionSymbol) PackageFrom() *Package {
	if fs.Scope_ != nil {
		return fs.Scope_.Package
	}
	return nil
}
func (vs *VariableSymbol) PackageFrom() *Package {
	if vs.Scope_ != nil {
		return vs.Scope_.Package
	}
	return nil
}

func (ts *TypeSymbol) Scope() *SymbolTable     { return ts.Scope_ }
func (ps *PackageSymbol) Scope() *SymbolTable  { return ps.Scope_ }
func (fs *FunctionSymbol) Scope() *SymbolTable { return fs.Scope_ }
func (vs *VariableSymbol) Scope() *SymbolTable { return vs.Scope_ }

func (ts *TypeSymbol) Positions() PositionSet        { return ts.Posits }
func (ps *PackageSymbol) Positions() PositionSet     { return ps.Posits }
func (fs *FunctionSymbol) Positions() PositionSet    { return fs.Posits }
func (vs *VariableSymbol) Positions() PositionSet    { return vs.Posits }
func (ts *PointerTypeSymbol) Positions() PositionSet { return ts.BaseType.Positions() }


func (ts *TypeSymbol) Identifiers() IdentSet     { return ts.Idents }
func (ps *PackageSymbol) Identifiers() IdentSet  { return ps.Idents }
func (fs *FunctionSymbol) Identifiers() IdentSet { return fs.Idents }
func (vs *VariableSymbol) Identifiers() IdentSet { return vs.Idents }


func (ts *TypeSymbol) SetName(name string)     { ts.name = name }
func (ps *PackageSymbol) SetName(name string)  { ps.name = name }
func (vs *VariableSymbol) SetName(name string) { vs.name = name }
func (fs *FunctionSymbol) SetName(name string) { fs.name = name }

func addPosition(sym Symbol, p token.Position) {
	if sym.Positions() != nil {
		sym.Positions().AddPosition(p)
	}
}

func (ts *TypeSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}
func (ts *PackageSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}
func (ts *VariableSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}
func (ts *FunctionSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}


func addIdent(sym Symbol, ident *ast.Ident) {
	sym.Identifiers().AddIdent(ident)
}

func (s *TypeSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}
func (s *PackageSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}
func (s *VariableSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}
func (s *FunctionSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}

//ITypeSymbol.Methods()
func (ts *TypeSymbol) Methods() *SymbolTable { return ts.Meths }

//ITypeSymbol.AddMethod()
func (ts *TypeSymbol) AddMethod(meth Symbol) {
	ts.Meths.AddSymbol(meth)
}
//ITypeSymbol.SetMethods()
func (ts *TypeSymbol) SetMethods(table *SymbolTable) {
	ts.Meths = table
}


func GetBaseType(sym ITypeSymbol) (ITypeSymbol, bool) {
	switch sym.(type) {
	case *PointerTypeSymbol, *AliasTypeSymbol:
		visitedTypes := make(map[string]ITypeSymbol)
		return getBaseType(sym, visitedTypes)
	}
	return sym, false

}
func GetBaseTypeOnlyPointer(sym ITypeSymbol) (ITypeSymbol, bool) {
	switch sym.(type) {
	case *PointerTypeSymbol:
		visitedTypes := make(map[string]ITypeSymbol)
		return getBaseTypeOnlyPointer(sym, visitedTypes)
	}
	return sym, false
}

func getBaseTypeOnlyPointer(sym ITypeSymbol, visited map[string]ITypeSymbol) (ITypeSymbol, bool) {
	if _, ok := visited[sym.Name()]; ok {
		return nil, true
	}
	if sym.Name() != "" {
		visited[sym.Name()] = sym
	}
	switch t := sym.(type) {
	case *PointerTypeSymbol:
		return getBaseType(t.BaseType, visited)
	}
	return sym, false
}


func getBaseType(sym ITypeSymbol, visited map[string]ITypeSymbol) (ITypeSymbol, bool) {
	if _, ok := visited[sym.Name()]; ok {
		return nil, true
	}
	if sym.Name() != "" {
		visited[sym.Name()] = sym
	}
	switch t := sym.(type) {
	case *PointerTypeSymbol:
		return getBaseType(t.BaseType, visited)
	case *AliasTypeSymbol:
		return getBaseType(t.BaseType, visited)
	}
	return sym, false
}

func (pt *PointerTypeSymbol) GetBaseStruct() (*StructTypeSymbol, bool) {
	t, _ := GetBaseType(pt)
	// 	if pt.Name() == "*Package" {
	// 		fmt.Printf("____%s__%s__\n", t.Name(), t.PackageFrom().AstPackage.Name)
	// 	}
	s, ok := t.(*StructTypeSymbol)
	return s, ok
}

func (pt *PointerTypeSymbol) Depth() int {
	switch t := pt.BaseType.(type) {
	case *PointerTypeSymbol:
		return t.Depth() + 1
	}
	return 1
}

func (ps *PackageSymbol) SetMethods(*SymbolTable) {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
}
func (ps *PackageSymbol) Methods() *SymbolTable {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
	return nil
}
func (ps *PackageSymbol) AddMethod(meth Symbol) {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
}


func SetIdentObject(ident *ast.Ident) *ast.Object {
	ident.Obj = &ast.Object{Name: ident.Name}
	return ident.Obj
}

func IsPredeclaredIdentifier(name string) bool {
	if _, ok := PredeclaredTypes[name]; ok {
		return true
	}
	if _, ok := PredeclaredFunctions[name]; ok {
		return true
	}
	if _, ok := PredeclaredConsts[name]; ok {
		return true
	}
	return false
}

func IsIntegerType(name string) (r bool) {
	_, r = integerTypes[name]
	return
}
func IsFloatType(name string) (r bool) {
	_, r = floatTypes[name]
	return
}
func IsComplexType(name string) (r bool) {
	_, r = complexTypes[name]
	return
}

func EqualsMethods(sym1 *FunctionSymbol, sym2 *FunctionSymbol) bool {
	return sym1.Name() == sym2.Name() && Equals(sym1.FunctionType, sym2.FunctionType)
}
func EqualsVariables(sym1 *VariableSymbol, sym2 *VariableSymbol) bool {
	return sym1.Name() == sym2.Name() && Equals(sym1.VariableType, sym2.VariableType)
}

func Equals(sym1 ITypeSymbol, sym2 ITypeSymbol) bool {
	if sym1.Name() != NO_NAME {
		return sym1 == sym2
	} else if sym2.Name() != NO_NAME {
		return false
	}
	switch t1 := sym1.(type) {
	case *AliasTypeSymbol:
		panic("basic or alias with no name")
	case *BasicTypeSymbol:
		panic("basic or alias with no name")
	case *StructTypeSymbol:
		t2, ok := sym2.(*StructTypeSymbol)
		if !ok {
			return false
		}
		if t1.Fields.Count() != t2.Fields.Count() {
			return false
		}
		for i, v := range *t1.Fields.Table {
			if !EqualsVariables(v.(*VariableSymbol), t2.Fields.Table.At(i).(*VariableSymbol)) {
				return false
			}
		}
		return true
	case *MapTypeSymbol:
		t2, ok := sym2.(*MapTypeSymbol)
		if !ok {
			return false
		}
		return Equals(t1.KeyType, t2.KeyType) && Equals(t1.ValueType, t2.ValueType)
	case *ChanTypeSymbol:
		t2, ok := sym2.(*ChanTypeSymbol)
		if !ok {
			return false
		}
		return Equals(t1.ValueType, t2.ValueType) && t1.Dir == t2.Dir
	case *InterfaceTypeSymbol:
		t2, ok := sym2.(*InterfaceTypeSymbol)
		if !ok {
			return false
		}
		if t1.Methods().Count() != t2.Methods().Count() {
			return false
		}
		for i, v := range *t1.Methods().Table {
			if !EqualsMethods(v.(*FunctionSymbol), t2.Methods().Table.At(i).(*FunctionSymbol)) {
				return false
			}
		}
		return true
	case *PointerTypeSymbol:
		t2, ok := sym2.(*PointerTypeSymbol)
		if !ok {
			return false
		}
		return Equals(t1.BaseType, t2.BaseType)
	case *FunctionTypeSymbol:
		t2, ok := sym2.(*FunctionTypeSymbol)
		if !ok {
			return false
		}
		if t1.Parameters.Count() != t2.Parameters.Count() {
			return false
		}
		if t1.Results.Count() != t2.Results.Count() {
			return false
		}
		for i, v := range *t1.Parameters.Table {
			if !Equals(v.(*VariableSymbol).VariableType, t2.Parameters.Table.At(i).(*VariableSymbol).VariableType) {
				return false
			}
		}
		for i, v := range *t1.Results.Table {
			if !Equals(v.(*VariableSymbol).VariableType, t2.Results.Table.At(i).(*VariableSymbol).VariableType) {
				return false
			}
		}
		return true

	}
	panic("unknown symbol type")
}
