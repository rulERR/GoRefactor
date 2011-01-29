package refactoring

import (
	"unicode"
	"st"
	"utils"
	"errors"
	"program"
)

func IsGoIdent(name string) bool {

	if name == "_" || name == "nil" || name == "true" || name == "false" {
		return false
	}
	if st.IsPredeclaredIdentifier(name) {
		return false
	}

	if !(unicode.IsLetter(int(name[0])) || name[0] == '_') {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !(unicode.IsLetter(int(name[i])) || unicode.IsDigit(int(name[i])) || name[0] == '_') {
			return false
		}
	}
	return true
}
func CheckRenameParameters(filename string, line int, column int, newName string) (bool, *errors.GoRefactorError) {
	switch {
	case filename == "" || !utils.IsGoFile(filename):
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	case line < 1:
		return false, errors.ArgumentError("line", "Must be > 1")
	case column < 1:
		return false, errors.ArgumentError("column", "Must be > 1")
	case !IsGoIdent(newName):
		return false, errors.ArgumentError("newName", "It's not a valid go identifier")
	}
	return true, nil
}
func Rename(programTree *program.Program, filename string, line int, column int, newName string) (bool, int, *errors.GoRefactorError) {

	if ok, err := CheckRenameParameters(filename, line, column, newName); !ok {
		return false, 0, err
	}

	var count int
	if sym, err := programTree.FindSymbolByPosition(filename, line, column); err == nil {

		if _, ok := sym.(*st.PointerTypeSymbol); ok {
			panic("find by position returned pointer type!!!")
		}
		if st.IsPredeclaredIdentifier(sym.Name()) {
			return false, 0, errors.UnrenamableIdentifierError(sym.Name(), " It's a basic language symbol")
		}
		if sym.PackageFrom().IsGoPackage {
			return false, 0, errors.UnrenamableIdentifierError(sym.Name(), " It's a symbol,imported from go library")
		}

		if _, ok := sym.Scope().LookUp(newName, filename); ok {
			return false, 0, errors.IdentifierAlreadyExistsError(newName)
		}

		if meth, ok := sym.(*st.FunctionSymbol); ok {
			if meth.IsInterfaceMethod {
				return false, 0, errors.UnrenamableIdentifierError(sym.Name(), " It's an interface method")
			}
		}
		count = renameSymbol(sym, newName)
	} else {
		return false, 0, err
	}
	return true, count, nil
}

func renameSymbol(sym st.Symbol, newName string) int {
	for ident, _ := range sym.Identifiers() {
		ident.Name = newName
	}
	return len(sym.Positions())
}
