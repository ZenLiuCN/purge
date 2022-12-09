package main

import (
	"bufio"
	"errors"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/gurkankaymak/hocon"
	"io/fs"
	logger "log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	Conf        *hocon.Config
	IgnoreFiles []string
	Garbage     gitignore.Matcher
	Keep        gitignore.Matcher
	Root        string
	Pwd         string
)

//goland:noinspection SpellCheckingInspection
var (
	defaultIgnoreFiles = []string{".gitignore"}
	defaultKeep        = []string{".git/"}
	log                *logger.Logger
	f                  *os.File
)

func execPath() string {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(err)
	}
	re, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		panic(err)
	}
	return re
}
func initial(test bool) {
	var err error
	Root = execPath()
	cfg := path.Join(Root, `purge.conf`)
	if Pwd == "" {
		Pwd, err = os.Getwd()
		if err != nil {
			panic(err)
		}
	}
	if logging {
		f, _ = os.OpenFile(path.Join(Pwd, "purge_"+time.Now().Format("2006-01-02_15")+".log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		log = logger.New(f, "", logger.Ldate|logger.Ltime)
	} else {
		log = logger.Default()
		log.SetFlags(logger.Ltime)
	}
	if _, err = os.Stat(cfg); errors.Is(err, os.ErrNotExist) {
		Conf, err = hocon.ParseString("")
		if err != nil {
			panic(err)
		}
		if test {
			log.Printf("not found config %s ", cfg)
		}
	} else {
		Conf, err = hocon.ParseResource(cfg)
		if err != nil {
			panic(err)
		}
		if test {
			log.Printf("use config %s", cfg)
		}
	}
	IgnoreFiles = Conf.GetStringSlice("files")
	IgnoreFiles = append(IgnoreFiles, defaultIgnoreFiles...)
	if test {
		log.Printf(`ignore files: %v`, IgnoreFiles)
	}
	garbage := Conf.GetStringSlice("garbage")
	if test {
		log.Printf(`garbage: %v`, garbage)
	}
	var ps []gitignore.Pattern
	for _, s := range garbage {
		ps = append(ps, gitignore.ParsePattern(s, nil))
	}
	Garbage = gitignore.NewMatcher(ps)
	keep := Conf.GetStringSlice("keep")
	keep = append(keep, defaultKeep...)
	if test {
		log.Printf(`keep: %v`, keep)
	}
	ps = nil
	for _, s := range keep {
		ps = append(ps, gitignore.ParsePattern(s, nil))
	}
	Keep = gitignore.NewMatcher(ps)
}

func load(pwd string, predicate func(p string) bool) (gitignore.Matcher, gitignore.Matcher) {
	var p []string
	for _, file := range IgnoreFiles {
		if _, err := os.Stat(path.Join(pwd, file)); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err == nil {
			file, err := os.OpenFile(path.Join(pwd, file), os.O_RDONLY, os.ModePerm)
			if err != nil {
				continue //TODO
			}
			sc := bufio.NewScanner(file)
			sc.Split(bufio.ScanLines)
			for sc.Scan() {
				l := strings.TrimSpace(sc.Text())
				if len(l) > 0 && l[0] != '#' && predicate(l) {
					p = append(p, l)
				}
			}
			_ = file.Close()
		} else {
			panic(err)
		}
	}
	if test && len(p) > 0 {
		log.Printf("pattern from files in %s : %+v \n", pwd, p)
	}
	var pa []gitignore.Pattern
	for _, s := range p {
		pa = append(pa, gitignore.ParsePattern(s, nil))
	}
	p = nil
	if _, err := os.Stat(path.Join(pwd, ".keep")); errors.Is(err, os.ErrNotExist) {
		return gitignore.NewMatcher(pa), nil
	} else if err == nil {
		file, err := os.OpenFile(path.Join(pwd, ".keep"), os.O_RDONLY, os.ModePerm)
		if err != nil {
			return gitignore.NewMatcher(pa), nil
		}
		sc := bufio.NewScanner(file)
		sc.Split(bufio.ScanLines)
		for sc.Scan() {
			l := strings.TrimSpace(sc.Text())
			if len(l) > 0 && l[0] != '#' {
				p = append(p, l)
			}
		}
		_ = file.Close()
	} else {
		panic(err)
	}
	if test {
		log.Printf("keep load: %+v", p)
	}
	var pa1 []gitignore.Pattern
	for _, s := range p {
		pa1 = append(pa1, gitignore.ParsePattern(s, nil))
	}
	return gitignore.NewMatcher(pa), gitignore.NewMatcher(pa1)
}

type Walker struct {
	ignores  []gitignore.Matcher
	keep     []gitignore.Matcher
	just     gitignore.Matcher
	patterns []string
	pwd      string
	test     bool
	removed  bool
}

func (w *Walker) predicate(p string) bool {
	for _, pattern := range w.patterns {
		if pattern == p {
			return false
		}
	}
	w.patterns = append(w.patterns, p)
	return true
}
func (w *Walker) Initial() {
	i, k := load(w.pwd, w.predicate)
	w.ignores = append(w.ignores, i)
	if k != nil {
		w.keep = append(w.keep, Keep, k)
	} else {
		w.keep = append(w.keep, Keep)
	}
	if just != "" {
		w.just = gitignore.NewMatcher([]gitignore.Pattern{gitignore.ParsePattern(just, nil)})
	}
}
func (w *Walker) push(p string) {
	if w.just != nil {
		return
	}
	i, k := load(p, w.predicate)
	w.ignores = append(w.ignores, i)
	w.keep = append(w.keep, k)
}
func (w *Walker) pop() {
	if w.just != nil {
		return
	}
	if len(w.ignores) == 1 {
		return
	}
	w.ignores = w.ignores[:len(w.ignores)-1]
	if len(w.keep) == 1 {
		return
	}
	w.keep = w.keep[:len(w.keep)-1]
}
func (w *Walker) shouldPurge(pth string, isDir bool) (bool, bool) {
	tar := strings.Split(pth, string(filepath.Separator))
	if w.just != nil {
		return w.just.Match(tar, isDir), false
	}
	matched := false
	for _, ignore := range w.ignores {
		if ignore.Match(tar, isDir) {
			matched = true
			if w.test {
				log.Print("matched by ignore file")
			}
			break
		}
	}
	if !matched {
		matched = Garbage.Match(tar, isDir)
		if matched && w.test {
			log.Print("matched by Garbage:")
		}
	}
	if matched {
		for _, k := range w.keep {
			if k != nil && k.Match(tar, isDir) {
				if w.test {
					log.Print("ignore by Keep:" + pth)
				}
				return false, true
			}
		}
		return true, false
	}
	for _, k := range w.keep {
		if k != nil && k.Match(tar, isDir) {
			return matched, true
		}
	}
	return matched, false
}
func (w *Walker) Walk() (err error) {
	if w.test {
		log.Printf("process files :\t%s\n", w.pwd)
	}
	last := w.pwd
	return filepath.Walk(w.pwd, func(path string, info fs.FileInfo, err error) error {
		dir := info.IsDir()
		if dir && !strings.HasPrefix(last, path) {
			w.pop()
			last = path
		}
		p, k := w.shouldPurge(path, dir)
		if p {
			log.Printf("purge:\t%s\n", path)
			if !w.test && purge {
				err := os.RemoveAll(path)
				if err != nil {
					return err
				}
			}
			if dir {
				return filepath.SkipDir
			}
			return nil
		} else if w.test && !w.removed {
			if k {
				log.Printf("ignore by keep:\t%s\n", path)
			} else {
				log.Printf("ignore:\t%s\n", path)
			}
		}
		if k && dir {
			return filepath.SkipDir
		}
		if dir && path != w.pwd {
			w.push(path)
		}
		return nil
	})
}
