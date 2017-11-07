package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"sort"
	"strings"
)

var (
	withStandard = flag.Bool("standard", false, "include standard packages")
	withTests    = flag.Bool("test", false, "include dependencies for tests")
)

func IsStandardPackage(importPath string) bool {
	x := strings.SplitN(importPath, "/", 2)
	return !strings.Contains(x[0], ".")
}

type FilterFunc func(string) bool

type DepsPrinter struct {
	keep    FilterFunc
	printed map[string]struct{}
}

func NewDepsPrinter(keep FilterFunc) *DepsPrinter {
	return &DepsPrinter{
		keep:    keep,
		printed: make(map[string]struct{}),
	}
}

func (d *DepsPrinter) Print(importPath string, srcDir string, withTests bool, p1, p2 string) error {
	pkg, err := build.Import(importPath, srcDir, 0)
	if err != nil {
		return err
	}

	if _, ok := d.printed[pkg.ImportPath]; ok {
		fmt.Printf("%s%s (see above)\n", p1, pkg.ImportPath)
		return nil
	} else {
		fmt.Printf("%s%s\n", p1, pkg.ImportPath)
	}

	d.printed[pkg.ImportPath] = struct{}{}

	depsUnique := make(map[string]bool)
	for _, im := range pkg.Imports {
		if !d.keep(im) {
			continue
		}
		depsUnique[im] = true
	}
	if withTests {
		for _, im := range pkg.TestImports {
			if !d.keep(im) {
				continue
			}
			depsUnique[im] = true
		}
	}

	deps := make([]string, 0, len(depsUnique))
	for dep := range depsUnique {
		deps = append(deps, dep)
	}

	sort.Strings(deps)

	for i, dep := range deps {
		var s1, s2 string
		if i != len(deps)-1 {
			s1 = "|- "
			s2 = "|  "
		} else {
			s1 = "`- "
			s2 = "   "
		}
		if err := d.Print(dep, pkg.Dir, false, p2+s1, p2+s2); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <importPath>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	var filter FilterFunc
	if *withStandard {
		filter = func(importPath string) bool {
			return importPath != "C"
		}
	} else {
		filter = func(importPath string) bool {
			return importPath != "C" && !IsStandardPackage(importPath)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	d := NewDepsPrinter(filter)
	pkg := flag.Args()[0]
	err = d.Print(pkg, cwd, *withTests, "", "")
	if err != nil {
		log.Fatal(err)
	}
}
