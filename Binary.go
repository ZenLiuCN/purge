package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ignored    strs
	ignoreFile strs
	dir        string
	just       string
	test       bool
	help       bool
	removed    bool
	logging    bool
	purge      bool
)

type strs []string

func (f *strs) String() string {
	return fmt.Sprintf("%v", []string(*f))
}

func (f *strs) Set(value string) error {
	*f = append(*f, value)
	return nil
}
func main() {
	flag.Var(&ignored, "i", "extra ignore(not purge) pattern for current run")
	flag.Var(&ignoreFile, "f", "extra ignore files (will be purge) for current run")
	flag.BoolVar(&test, "t", false, "test will print extra info and not real purge.")
	flag.StringVar(&dir, "d", "", "target dir,default is cwd")
	flag.StringVar(&just, "j", "", "just purge with one pattern,config and other setting won't take effect.")
	flag.BoolVar(&logging, "l", false, "logging to file,default print at console")
	flag.BoolVar(&purge, "p", false, "do purge when test not set")
	flag.BoolVar(&help, "h", false, "print help")
	flag.BoolVar(&removed, "r", true, "logging/print purge files only")
	flag.Parse()
	if help || (purge && test) || (!purge && !test && !logging) {
		flag.PrintDefaults()
		println("example:")
		println("\t purge -t -l  \t\t logging purge details into file")
		println("\t purge -l  \t\t only logging purge file list into file")
		println("\t purge -t -r=true -l  \t logging purge and ignore details into file")
		println("\t purge -p -l  \t\t * execute purge and logging purge file list into file")
		println("\t purge -p  \t\t * execute purge")
		os.Exit(0)
	}
	if ignored != nil {
		defaultKeep = append(defaultKeep, ignored...)
	}
	if ignoreFile != nil {
		IgnoreFiles = append(IgnoreFiles, ignoreFile...)
	}
	if dir != "" {
		var err error
		Pwd, err = filepath.Abs(dir)
		if err != nil {
			panic(err)
		}
	}
	initial(test)
	if f != nil {
		defer f.Close()
	}
	walker := &Walker{
		ignores: nil,
		pwd:     Pwd,
		test:    test,
		removed: removed,
	}
	walker.Initial()
	err := walker.Walk()
	if err != nil && err != filepath.SkipDir {
		panic(err)
	}

}
