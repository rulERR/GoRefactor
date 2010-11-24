package packageParser

import (
	"go/token"
	"go/ast"
	"container/vector"
	"fmt"
	"st"
)

type exprParser struct {
	IsTypeNameUsed    bool
	FieldsSymbolTable *st.SymbolTable
	SearchInFields    bool

	CompositeLiteralElementType st.ITypeSymbol
}

func (pp *packageParser) parseExpr(exp ast.Expr) (res *vector.Vector) {

	if exp == nil {
		fmt.Printf("((nil expr\n")
		return nil
	}
	res = new(vector.Vector)

	switch e := exp.(type) {
	case *ast.BasicLit:
		return pp.eParseBasicLit(e)
	case *ast.BinaryExpr:
		return pp.eParseBinaryExpr(e)
	case *ast.CallExpr:
		return pp.eParseCallExpr(e)
	case *ast.CompositeLit:
		return pp.eParseCompositeLit(e)
	case *ast.FuncLit: //later in locals visitor
		if pp.RegisterPositions {
			pp.RegisterPositions = false
			res.Push(pp.parseExpr(e.Type).At(0))
			pp.RegisterPositions = true
		} else {
			res.Push(pp.parseExpr(e.Type).At(0))
		}

	case *ast.Ident:
		return pp.eParseIdent(e)
	case *ast.IndexExpr:
		return pp.eParseIndexExpr(e)
	case *ast.KeyValueExpr:
		return pp.eParseKeyValueExpr(e)
	case *ast.ParenExpr:
		res.AppendVector(pp.parseExpr(e.X))
	case *ast.SelectorExpr:
		return pp.eParseSelectorExpr(e)
	case *ast.SliceExpr:
		return pp.eParseSliceExpr(e)
	case *ast.StarExpr:
		return pp.eParseStarExpr(e)
	case *ast.TypeAssertExpr:
		return pp.eParseTypeAssertExpr(e)
	case *ast.UnaryExpr:
		return pp.eParseUnaryExpr(e)
	case *ast.ArrayType, *ast.StructType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.ChanType:
		//type conversion, composite lits
		t := pp.parseTypeSymbol(exp)
		res.Push(t)
	}
	return
}

func (pp *packageParser) eParseBasicLit(e *ast.BasicLit) (res *vector.Vector) {
	res = new(vector.Vector)
	switch e.Kind {
	case token.INT:
		res.Push(st.PredeclaredTypes["int"])
	case token.FLOAT:
		res.Push(st.PredeclaredTypes["float"])
	case token.CHAR:
		res.Push(st.PredeclaredTypes["byte"])
	case token.STRING:
		res.Push(st.PredeclaredTypes["string"])
	case token.IMAG:
		res.Push(st.PredeclaredTypes["complex"])
	}
	return
}

func (pp *packageParser) eParseBinaryExpr(e *ast.BinaryExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	xxType := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	yyType := pp.parseExpr(e.Y).At(0).(st.ITypeSymbol)

	xType, cyc := st.GetBaseType(xxType)
	if cyc {
		panic("aaaa")
	}
	yType, cyc := st.GetBaseType(yyType)
	if cyc {
		panic("aaaa")
	}

	if e.Op == token.OR {
		fmt.Printf("%T,%T,%v,%v\n", xType, yType, xType.Name(), yType.Name())
	}
	switch e.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.AND, token.OR, token.XOR, token.SHL, token.SHR, token.AND_NOT, token.LAND, token.LOR:

		x, okx := st.PredeclaredTypes[xType.Name()]
		y, oky := st.PredeclaredTypes[yType.Name()]
		switch {
		case okx && oky:
			res.Push(x)
		case okx:
			res.Push(y)
		case oky:
			res.Push(x)
		default:
			res.Push(x)
		}
	case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		res.Push(st.PredeclaredTypes["bool"])
	case token.ARROW: //ch <- val
		res.Push(st.PredeclaredTypes["bool"])
	}

	return
}

func (pp *packageParser) eParseCallExpr(e *ast.CallExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	for _, par := range e.Args {
		pp.parseExpr(par)
	}

	//check if builtIn special case
	if f, ok := e.Fun.(*ast.Ident); ok {
		name := f.Name
		if _, ok1 := st.PredeclaredFunctions[name]; ok1 {
			if vect, ok := pp.eParseBuiltInFunctionCall(name, e); ok {
				return vect
			}
		}
	}

	tt := pp.parseExpr(e.Fun).At(0).(st.ITypeSymbol)

	if vect, ok := pp.eParseRegularFunctionCall(tt); ok {
		return vect
	}

	x, cyc := st.GetBaseType(tt)
	if cyc {
		panic("ERROR: cycle wasn't expected. eParseCallExpr, parseExpr.go")
	}
	switch x.(type) {
	case *st.UnresolvedTypeSymbol: // sould be resolved later
		res.Push(x)
	default: // Type conversion,push original type
		res.Push(tt)
	}

	return
}
func (pp *packageParser) eParseCompositeLit(e *ast.CompositeLit) (res *vector.Vector) {
	res = new(vector.Vector)

	clType := pp.processCLType(e)

	realClType, cyc := st.GetBaseType(clType)
	if cyc {
		panic("ERROR: cycle wasn't expected. eParseCompositeLit, parseExpr.go")
	}

	pp.detectWhereToLookUpKeysIfAny(realClType, e)
	defer func() { pp.ExprParser.SearchInFields = false; pp.ExprParser.FieldsSymbolTable = nil }()

	max := pp.eParseCLElementsAndGetMaxIndex(e)

	// get length for [...]array type
	if arr, ok := clType.(*st.ArrayTypeSymbol); ok {
		if arr.Len == st.ELLIPSIS {
			arr.Len = max + 1
		}
	}

	res.Push(clType)
	return
}

func (pp *packageParser) eParseIdent(e *ast.Ident) (res *vector.Vector) {
	res = new(vector.Vector)
	//fmt.Printf("%s:	Parse Ident %v\n", pp.Package.AstPackage.Name, e.Name)

	lookupST := pp.detectWhereToLookUpIdent()

	if t, found := lookupST.LookUp(e.Name, pp.CurrentFileName); found {
		e.Obj = t.Object()
		if pp.RegisterPositions {
			t.AddPosition(st.NewOccurence(e.Pos()))
		}

		switch v := t.(type) {
		case *st.VariableSymbol:
			res.Push(v.VariableType)
		case *st.FunctionSymbol:
			res.Push(v.FunctionType)
		default: //PackageSymbol or type
			if _, ok := t.(*st.PackageSymbol); !ok {
				fmt.Printf("%s:	<><><><>  %v - %T \n", pp.Package.AstPackage.Name, t.Name(), t)
				pp.ExprParser.IsTypeNameUsed = true
			}
			res.Push(v)
		}
	} else {
		//sould be resolved later
		fmt.Printf("%s:	WARNING! Ident %v wasn't found\n", pp.Package.AstPackage.Name, e.Name)
		res.Push(&st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: e.Obj, Posits: new(vector.Vector), PackFrom: nil}, e})
	}

	return
}
func (pp *packageParser) eParseIndexExpr(e *ast.IndexExpr) (res *vector.Vector) {

	res = new(vector.Vector)
	x, cyc := st.GetBaseType(pp.parseExpr(e.X).At(0).(st.ITypeSymbol))
	if cyc {
		fmt.Println("ERROR: cycle wasn't expected. eParseIndexExpr, parseExpr.go")
	}

	switch s := x.(type) {
	case *st.ArrayTypeSymbol:
		res.Push(s.ElemType)
	case *st.MapTypeSymbol:
		res.Push(s.ValueType)
		//if RightHand
		res.Push(st.PredeclaredTypes["bool"])
	case *st.BasicTypeSymbol: //string
		res.Push(st.PredeclaredTypes["byte"])
	case *st.UnresolvedTypeSymbol: //string
		res.Push(s)
	}
	return
}
func (pp *packageParser) eParseKeyValueExpr(e *ast.KeyValueExpr) (res *vector.Vector) {

	pp.parseExpr(e.Key) // struct fields or array indexes
	pp.parseExpr(e.Value)

	res = new(vector.Vector)
	//return array index to count it's length
	if l, ok := e.Key.(*ast.BasicLit); ok {
		if l.Kind == token.INT || l.Kind == token.CHAR {
			res.Push(getIntValue(l))
		}
	}
	return
}
func (pp *packageParser) eParseSelectorExpr(e *ast.SelectorExpr) (res *vector.Vector) {

	pp.ExprParser.IsTypeNameUsed = false

	t := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)

	// find method
	if vect, ok := pp.eParseMethodSelector(t, e); ok {
		return vect
	}

	// field
	if vect, ok := pp.eParseFieldSelector(t, e); ok {
		return vect
	}

	//type from other package
	if vect, ok := pp.eParsePackageEntitySelector(t, e); ok {
		return vect
	}

	res = new(vector.Vector)
	//Sould be resolved
	res.Push(&st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: e.Sel.Obj, Posits: new(vector.Vector), PackFrom: nil}, e})
	return
}
func (pp *packageParser) eParseSliceExpr(e *ast.SliceExpr) (res *vector.Vector) {
	res = new(vector.Vector)

	x := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	pp.parseExpr(e.Index)
	pp.parseExpr(e.End)
	r := x
	// slicing an array results a slice
	if arr, ok := x.(*st.ArrayTypeSymbol); ok {
		if arr.Len != st.SLICE {
			sl := arr.Copy().(*st.ArrayTypeSymbol)
			sl.Len = st.SLICE
			r = sl
		}
	}
	res.Push(r)
	return
}

func (pp *packageParser) eParseStarExpr(e *ast.StarExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	base := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	res.Push(pp.getOrAddPointer(base))
	return
}
func (pp *packageParser) eParseTypeAssertExpr(e *ast.TypeAssertExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	pp.parseExpr(e.X) //pos
	t := pp.parseExpr(e.Type)
	res.Push(t.At(0))
	res.Push(st.PredeclaredTypes["bool"])
	return
}
func (pp *packageParser) eParseUnaryExpr(e *ast.UnaryExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	t := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	switch e.Op {
	case token.ADD, token.SUB, token.XOR, token.NOT:
		res.Push(t)
	case token.ARROW:
		chh, cyc := st.GetBaseType(t)
		if cyc {
			fmt.Println("ERROR: cycle wasn't expected. eParseUnaryExpr, parseExpr.go")
		}
		ch := chh.(*st.ChanTypeSymbol)
		res.Push(ch.ValueType)
		res.Push(st.PredeclaredTypes["bool"])
	case token.AND:
		base := t
		res.Push(pp.getOrAddPointer(base))
	}
	return
}

// lower levels parses

func (pp *packageParser) eParseBuiltInFunctionCall(name string, e *ast.CallExpr) (res *vector.Vector, success bool) {
	res = new(vector.Vector)
	success = true
	var tt st.ITypeSymbol
	switch name {
	case "new":
		if pp.RegisterPositions {
			pp.RegisterPositions = false
			tt = pp.parseTypeSymbol(e.Args[0])
			pp.RegisterPositions = true
		} else {
			tt = pp.parseTypeSymbol(e.Args[0])
		}
		var resT st.ITypeSymbol
		// LookUp or create a pointer type
		// check if it's possible to call new(*T)
		pd, found := 0, false
		if p, ok := tt.(*st.PointerTypeSymbol); ok {
			pd = p.Depth()
		}

		if resT, found = pp.RootSymbolTable.LookUpPointerType(tt.Name(), pd+1); !found {

			resT = &st.PointerTypeSymbol{&st.TypeSymbol{Obj: tt.Object(), Meths: nil, Posits: new(vector.Vector), PackFrom: pp.Package}, tt, nil}
			pp.RootSymbolTable.AddSymbol(resT)
		}

		res.Push(resT)
		return
	case "make":
		if pp.RegisterPositions {
			pp.RegisterPositions = false
			tt = pp.parseTypeSymbol(e.Args[0])
			pp.RegisterPositions = true
		} else {
			tt = pp.parseTypeSymbol(e.Args[0])
		}
		res.Push(tt)
		return
	case "real", "imag":
		if pp.RegisterPositions {
			pp.RegisterPositions = false
			tt = pp.parseTypeSymbol(e.Args[0])
			pp.RegisterPositions = true
		} else {
			tt = pp.parseTypeSymbol(e.Args[0])
		}
		switch tt.Name() {
		case "cmplx":
			res.Push(st.PredeclaredTypes["float"])
		case "cmplx64":
			res.Push(st.PredeclaredTypes["float32"])
		case "cmplx128":
			res.Push(st.PredeclaredTypes["float64"])
		}
		return
	case "cmplx":
		var t1, t2 st.ITypeSymbol
		if pp.RegisterPositions {
			pp.RegisterPositions = false
			t1 = pp.parseTypeSymbol(e.Args[0])
			t2 = pp.parseTypeSymbol(e.Args[1])
			pp.RegisterPositions = true
		} else {
			t1 = pp.parseTypeSymbol(e.Args[0])
			t2 = pp.parseTypeSymbol(e.Args[1])
		}

		switch {
		case t1.Name() == "float64" || t2.Name() == "float64":
			res.Push(st.PredeclaredTypes["cmplx128"])
		case t1.Name() == "float32" || t2.Name() == "float32":
			res.Push(st.PredeclaredTypes["cmplx64"])
		default:
			res.Push(st.PredeclaredTypes["float"])
		}
		return
	case "append":
		if pp.RegisterPositions {

			pp.RegisterPositions = false
			tt = pp.parseTypeSymbol(e.Args[0])
			pp.RegisterPositions = true
		} else {
			tt = pp.parseTypeSymbol(e.Args[0])
		}
		res.Push(tt)

		return
	}

	return nil, false
}

func (pp *packageParser) eParseRegularFunctionCall(fType st.ITypeSymbol) (res *vector.Vector, success bool) {
	res = new(vector.Vector)

	x, cyc := st.GetBaseType(fType)
	if cyc {
		panic("ERROR: cycle wasn't expected. eParseCallExpr, parseExpr.go")
	}

	if ft, ok := x.(*st.FunctionTypeSymbol); ok { //regular case - function
		if ft.Results != nil {
			for v := range ft.Results.Iter() {
				res.Push(v.(*st.VariableSymbol).VariableType)
			}
		}
		return res, true
	}
	return nil, false
}

func (pp *packageParser) eParseCLElementsAndGetMaxIndex(e *ast.CompositeLit) (max int) {
	for _, elt := range e.Elts {
		v := pp.parseExpr(elt)
		if v.Len() > 0 {
			if i, ok := v.At(0).(int); ok && max < i {
				max = i
			}
		}
	}
	return
}
func (pp *packageParser) eParseMethodSelector(t st.ITypeSymbol, e *ast.SelectorExpr) (res *vector.Vector, success bool) {

	lookupST := pp.detectWhereToLookUpMethodSelector(t)

	if lookupST != nil {
		if ff, ok := lookupST.LookUp(e.Sel.Name, ""); ok {
			if f, ok := ff.(*st.FunctionSymbol); ok {

				res = new(vector.Vector)

				e.Sel.Obj = f.Object()
				if pp.RegisterPositions {
					f.AddPosition(st.NewOccurence(e.Sel.Pos()))
				}

				if pp.ExprParser.IsTypeNameUsed {
					pp.ExprParser.IsTypeNameUsed = false
					f = pp.makeMethodExpression(f)
				}
				res.Push(f.FunctionType)
				return res, true
			}
		}
	}
	return nil, false
}

func (pp *packageParser) eParseFieldSelector(t st.ITypeSymbol, e *ast.SelectorExpr) (res *vector.Vector, success bool) {
	x, cyc := st.GetBaseType(t)
	if cyc {
		panic("ERROR: cycle wasn't expected. eParseSelectorExpr, parseExpr.go")
	}

	var lookupST = pp.detectWhereToLookUpFieldSelector(x)

	if vv, ok := lookupST.LookUp(e.Sel.Name, ""); ok {
		if va, ok := vv.(*st.VariableSymbol); ok {
			e.Sel.Obj = va.Obj
			if pp.RegisterPositions {
				va.AddPosition(st.NewOccurence(e.Sel.Pos()))
			}

			res = new(vector.Vector)
			res.Push(va.VariableType)
			return res, true
		}
	}
	return nil, false
}
func (pp *packageParser) eParsePackageEntitySelector(t st.ITypeSymbol, e *ast.SelectorExpr) (res *vector.Vector, success bool) {
	if s, ok := t.(*st.PackageSymbol); ok {
		if tt, ok := s.Package.Symbols.LookUp(e.Sel.Name, ""); ok {
			e.Sel.Obj = tt.Object()
			if pp.RegisterPositions {
				tt.AddPosition(st.NewOccurence(e.Sel.Pos()))
			}
			res = new(vector.Vector)
			res.Push(tt)
			pp.ExprParser.IsTypeNameUsed = true
			return res, true
		}
	}
	return nil, false
}

func (pp *packageParser) processCLType(e *ast.CompositeLit) (clType st.ITypeSymbol) {
	if e.Type != nil {
		clType = pp.parseExpr(e.Type).At(0).(st.ITypeSymbol)
		if arr, ok := clType.(*st.ArrayTypeSymbol); ok {
			pp.ExprParser.CompositeLiteralElementType = arr.ElemType
		}
	} else {
		clType = pp.ExprParser.CompositeLiteralElementType
	}
	return
}

func (pp *packageParser) detectWhereToLookUpKeysIfAny(clType st.ITypeSymbol, e *ast.CompositeLit) {
	if str, ok := clType.(*st.StructTypeSymbol); ok {
		if len(e.Elts) > 0 {
			if _, ok := e.Elts[0].(*ast.KeyValueExpr); ok {
				pp.ExprParser.SearchInFields = true
				pp.ExprParser.FieldsSymbolTable = str.Fields
			}
		}
	}
}
func (pp *packageParser) detectWhereToLookUpIdent() (lookupST *st.SymbolTable) {
	if pp.ExprParser.SearchInFields {
		lookupST = pp.ExprParser.FieldsSymbolTable
	} else {
		lookupST = pp.CurrentSymbolTable
	}
	return
}
func (pp *packageParser) detectWhereToLookUpMethodSelector(source st.ITypeSymbol) (lookupST *st.SymbolTable) {
	if s, ok := source.(*st.PackageSymbol); ok {
		lookupST = s.Package.Symbols
	} else if source.Methods() != nil {
		lookupST = source.Methods()
	}
	return
}

func (pp *packageParser) detectWhereToLookUpFieldSelector(source st.ITypeSymbol) (lookupST *st.SymbolTable) {
	switch s := source.(type) {
	case *st.StructTypeSymbol:
		lookupST = s.Fields
	case *st.PointerTypeSymbol:
		lookupST = s.Fields
	case *st.PackageSymbol:
		lookupST = s.Package.Symbols
	}
	return
}


func (pp *packageParser) makeMethodExpression(fs *st.FunctionSymbol) (res *st.FunctionSymbol) {
	res = fs.Copy()
	fft, cyc := st.GetBaseType(fs.FunctionType)
	if cyc {
		fmt.Println("ERROR: cycle wasn't expected. eParseUnaryExpr, parseExpr.go")
	}
	ft := fft.(*st.FunctionTypeSymbol)

	newFt := &st.FunctionTypeSymbol{Parameters: st.NewSymbolTable(pp.Package)}

	if ft.Reciever == nil {
		fmt.Println("ERROR : f.Reciever == nil. makeMethodExpression,parseExpr.go")
	}
	for sym := range ft.Reciever.Iter() {
		newFt.Parameters.AddSymbol(sym)
	}
	if ft.Parameters != nil {
		for sym := range ft.Parameters.Iter() {
			newFt.Parameters.AddSymbol(sym)
		}
	}
	if ft.Results != nil {
		newFt.Results = st.NewSymbolTable(pp.Package)
		for sym := range ft.Results.Iter() {
			newFt.Results.AddSymbol(sym)
		}
	}
	res.FunctionType = newFt
	return
}
