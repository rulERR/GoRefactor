package testPack

import "os"

var inm_global int

func inm1(a bool, t os.Error) {
	if a {
		println()
		inm_global = 0
	}
	if t == os.EOF {
		println(t.String())
	}
	b := false
	println(b)
}

func inm2(a bool, t os.Error) {
	if a {
	}
	if t == os.EOF {
		b := false
		println(b)
	}
}

func inm_test() {

	b := false
	var ttt os.Error
	aaa := true

	inm1(aaa, ttt)

	inm2(aaa, ttt)

	println(b)
}
