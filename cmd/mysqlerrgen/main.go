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
				// error
			}
			errorCodeOffset, _ = strconv.Atoi(offsetStr)
			rCount = 0
		case strings.HasPrefix(line, "default-language"):
			_, line = consumeWord(line)
			line = trimDelimiters(line)
			shortName, rest := consumeWord(line)
			if rest != "" {
				// error
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
				// Unexpected EOL
			}
			text := parseQuoted(line[1:])
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
				obsolete:  strings.HasPrefix(errorName, "OBSOLETE_ER_"),
			})
		case strings.HasPrefix(line, "#"), line == "":
			// comment
		case strings.HasPrefix(line, "reserved-error-section"):
		default:
			// unknown format
			panic("unknown format: " + strconv.Quote(line))
		}
	}
	_ = defaultLanguage
	_ = languages

	if err := os.MkdirAll(*pkg, 0777); err != nil {
		return fmt.Errorf("make package dir: %w", err)
	}
	f, err := os.Create(filepath.Join(*pkg, "constants.go"))
	if err != nil {
		return fmt.Errorf("create constants.go: %w", err)
	}
	defer f.Close()

	fmt.Fprintln(f, "// Code generated mysqlerrgen DO NOT EDIT.")
	writeLicense(f)
	fmt.Fprintln(f, "package", *pkg)
	for _, mysqlErr := range errs {
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

func parseQuoted(s string) string {
	var b strings.Builder
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		if r[i] == '"' {
			return b.String()
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
	panic("unexpected EOL")
}

func writeLicense(w io.Writer) {
	fmt.Fprintln(w, `// Copyright (C) 2021 Nao Yonashiro 
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.`)
}