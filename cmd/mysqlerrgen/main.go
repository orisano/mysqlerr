package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type language struct {
	longName  string
	shortName string
	charset   string
}

type message struct {
	langShortName string
	text          string
}

type mysqlError struct {
	name      string
	code      int
	sqlState  string
	odbcState string
	messages  []message
	obsolete  bool
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	pkg := flag.String("pkg", "", "package name")
	url := flag.String("url", "", "source url")
	flag.Parse()

	var r io.Reader
	if *url != "" {
		resp, err := http.Get(*url)
		if err != nil {
			return fmt.Errorf("get: %w", err)
		}
		defer resp.Body.Close()
		r = resp.Body
	} else {
		r = os.Stdin
	}

	s := bufio.NewScanner(r)
	defaultLanguage := "eng"
	errorCodeOffset := 1000
	rCount := 0
	var languages []language
	var errs []mysqlError
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "language"):
			languages = parseLanguage(line)
		case strings.HasPrefix(line, "start-error-number"):
			_, line = consumeWord(line)
			line = trimDelimiters(line)
			offsetStr, rest := consumeWord(line)
			if rest != "" {
				return fmt.Errorf("invalid format: %q", s.Text())
			}
			errorCodeOffset, _ = strconv.Atoi(offsetStr)
			rCount = 0
		case strings.HasPrefix(line, "default-language"):
			_, line = consumeWord(line)
			line = trimDelimiters(line)
			shortName, rest := consumeWord(line)
			if rest != "" {
				return fmt.Errorf("invalid format: %q", s.Text())
			}
			defaultLanguage = shortName
		case strings.HasPrefix(line, "\t"), strings.HasPrefix(line, " "):
			line = strings.TrimLeft(line, " \t")
			langShortName := ""
			if i := strings.IndexAny(line, " \t"); i >= 0 {
				langShortName, line = line[:i], line[i:]
			} else {
				langShortName, line = line, ""
			}
			line = strings.TrimLeft(line, " \t")
			if !strings.HasPrefix(line, `"`) {
				return fmt.Errorf("unexpected EOL: %q", s.Text())
			}
			text, err := parseQuoted(line[1:])
			if err != nil {
				return fmt.Errorf("parse quote(%q): %w", s.Text(), err)
			}
			curErr := &errs[len(errs)-1]
			curErr.messages = append(curErr.messages, message{
				langShortName: langShortName,
				text:          text,
			})
		case strings.HasPrefix(line, "ER_"), strings.HasPrefix(line, "WARN_"), strings.HasPrefix(line, "OBSOLETE_ER_"), strings.HasPrefix(line, "OBSOLETE_WARN_"):
			var errorName, sqlState, odbcState string
			errorName, line = consumeWord(line)
			line = trimDelimiters(line)
			sqlState, line = consumeWord(line)
			line = trimDelimiters(line)
			odbcState, line = consumeWord(line)
			errorCode := errorCodeOffset + rCount
			rCount++
			errs = append(errs, mysqlError{
				name:      errorName,
				code:      errorCode,
				sqlState:  sqlState,
				odbcState: odbcState,
				obsolete:  strings.HasPrefix(errorName, "OBSOLETE_"),
			})
		case strings.HasPrefix(line, "#"), line == "":
			// comment
		case strings.HasPrefix(line, "reserved-error-section"):
		default:
			// unknown format
			return fmt.Errorf("unknown format: %q", line)
		}
	}
	_ = defaultLanguage
	_ = languages

	if err := os.MkdirAll(*pkg, 0777); err != nil {
		return fmt.Errorf("make package dir: %w", err)
	}

	cs := &constants{}
	constantsPath := filepath.Join(*pkg, "constants.go")
	if _, err := os.Stat(constantsPath); err == nil {
		cs, err = parseConstantsGo(constantsPath)
		if err != nil {
			return fmt.Errorf("parse constants.go: %w", err)
		}
	}

	f, err := os.Create(constantsPath)
	if err != nil {
		return fmt.Errorf("create constants.go: %w", err)
	}
	defer f.Close()

	fmt.Fprintln(f, "// Code generated mysqlerrgen DO NOT EDIT.")
	writeLicense(f)
	fmt.Fprintln(f, "package", *pkg)
	for _, mysqlErr := range errs {
		for _, d := range cs.deprecates(mysqlErr.name, mysqlErr.code) {
			fmt.Fprintln(f, "// Deprecated: should not be used")
			fmt.Fprintln(f, "const", d.name, "=", d.code)
		}
		fmt.Fprintln(f, "const", mysqlErr.name, "=", mysqlErr.code)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close constants.go: %w", err)
	}
	return nil
}

func consumeWord(s string) (string, string) {
	i := strings.IndexAny(s, " ,\t\r\n=")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i:]
}

func trimDelimiters(s string) string {
	return strings.TrimLeft(s, " ,\t=")
}

// <keyword> <lang>[, <lang>]* ;
// keyword := language[s]
// lang := <long_name>=<short_name> <charset>
func parseLanguage(s string) []language {
	_, s = consumeWord(s) // skip keyword
	s = trimDelimiters(s)

	var languages []language
	for !(strings.HasPrefix(s, ";") || s == "") {
		longName, x := consumeWord(s)
		x = trimDelimiters(x)
		shortName, x := consumeWord(x)
		x = trimDelimiters(x)
		charset, x := consumeWord(x)
		s = trimDelimiters(x)
		languages = append(languages, language{
			longName:  longName,
			shortName: shortName,
			charset:   charset,
		})
	}
	return languages
}

func parseQuoted(s string) (string, error) {
	var b strings.Builder
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		if r[i] == '"' {
			return b.String(), nil
		}
		if r[i] == '\\' && i+1 < len(r) {
			i++
			switch r[i] {
			case 'n':
				b.WriteByte('\n')
			case '0', '1', '2', '3', '4', '5', '6', '7':
				n := 0
				for j := 0; j < 3 && i < len(r); i, j = i+1, j+1 {
					if r[i] < '0' && '7' < r[i] {
						i--
						break
					}
					n = n*8 + int(r[i]-'0')
				}
				b.WriteByte(byte(n))
			default:
				b.WriteRune(r[i])
			}
		} else {
			b.WriteRune(r[i])
		}
	}
	return "", fmt.Errorf("unexpected EOL")
}

func writeLicense(w io.Writer) {
	fmt.Fprintln(w, `// Copyright 2021-2023 Nao Yonashiro 
// 
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// 
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`)
}

type constants struct {
	byName map[string]int
	byCode map[int][]string
}

func (c *constants) add(name string, code int) {
	if c.byName == nil {
		c.byName = map[string]int{}
	}
	c.byName[name] = code
	if c.byCode == nil {
		c.byCode = map[int][]string{}
	}
	c.byCode[code] = append(c.byCode[code], name)
}

func (c *constants) deprecates(name string, code int) []mysqlError {
	var ds []mysqlError
	names := c.byCode[code]
	for _, n := range names {
		if n == name {
			continue
		}
		ds = append(ds, mysqlError{name: n, code: code})
	}
	return ds
}

func parseConstantsGo(name string) (*constants, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var c constants
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "const ") {
			continue
		}
		tokens := strings.Split(line, " ")
		key := tokens[1]
		val, _ := strconv.Atoi(tokens[3])
		c.add(key, val)
	}
	return &c, nil
}
