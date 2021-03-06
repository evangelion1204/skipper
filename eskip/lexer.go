// Copyright 2015 Zalando SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eskip

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// used for wrapping tokenizer expressions and extending
// them with metadata
type tokenRx struct {
	token         int
	expression    string
	captureGroups int
	matchIndex    int
}

// implements the lexer instance
type eskipLex struct {
	tokenRxs     []*tokenRx
	rx           *regexp.Regexp
	code         string
	routes       []*parsedRoute
	filters      []*Filter
	lastToken    string
	lastRaw      string
	lastPosition int
	err          error
}

// creates and initializes a lexer instance
func newLexer(code string) *eskipLex {
	const (
		rxFmt                = "^(\\s*|//.*\r?\n|//.*$)*(%s)(\\s*|//.*\r?\n|//.*$)*"
		initialCaptureGroups = 3
	)

	tokenRxs := []*tokenRx{
		&tokenRx{
			token:         and,
			expression:    "[&][&]",
			captureGroups: 0},

		&tokenRx{
			token:         arrow,
			expression:    "->",
			captureGroups: 0},

		&tokenRx{
			token:         closeparen,
			expression:    "\\)",
			captureGroups: 0},

		&tokenRx{
			token:         colon,
			expression:    ":",
			captureGroups: 0},

		&tokenRx{
			token:         comma,
			expression:    ",",
			captureGroups: 0},

		&tokenRx{
			token:         number,
			expression:    "[0-9]*[.]?[0-9]+",
			captureGroups: 0},

		&tokenRx{
			token:         openparen,
			expression:    "\\(",
			captureGroups: 0},

		&tokenRx{
			token:         regexpliteral,
			expression:    "/(\\\\\\\\|\\\\/|[^/])*/",
			captureGroups: 1},

		&tokenRx{
			token:         semicolon,
			expression:    ";",
			captureGroups: 0},

		&tokenRx{
			token:         shunt,
			expression:    "<shunt>",
			captureGroups: 0},

		&tokenRx{
			token:         stringliteral,
			expression:    "\"(\\\\\\\\|\\\\\"|[^\"])*\"",
			captureGroups: 1},

		&tokenRx{
			token:         stringliteral,
			expression:    "`(\\\\\\\\|\\\\`|[^\"])*`",
			captureGroups: 1},

		&tokenRx{
			token:         symbol,
			expression:    "[a-zA-Z_]\\w*",
			captureGroups: 0}}

	// mapping between the token expressions and the related capture groups
	// in the merged token expression
	tokenRxss := make([]string, len(tokenRxs))
	captureGroups := initialCaptureGroups
	for i, trx := range tokenRxs {
		trx.matchIndex = i + captureGroups
		captureGroups += trx.captureGroups
		tokenRxss[i] = fmt.Sprintf("(%s)", trx.expression)
	}

	// compile all token expressions into a single expression
	// let it panic, expression not coming from external source
	rx := regexp.MustCompile(fmt.Sprintf(rxFmt, strings.Join(tokenRxss, "|")))

	return &eskipLex{tokenRxs: tokenRxs, rx: rx, code: code}
}

// unescape tokens
func unescape(s string, chars string) string {
	r := make([]string, 0, len(s))
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i : i+1]
		switch {
		case escaped && strings.Index(chars, c) >= 0:
			r = append(r, c)
			escaped = false
		case escaped:
			r = append(r, "\\", c)
			escaped = false
		case c == "\\":
			escaped = true
		default:
			r = append(r, c)
		}
	}

	return strings.Join(r, "")
}

// conversion error ignored, tokenizer expression already checked format
func convertNumber(s string) float64 {
	n, _ := strconv.ParseFloat(s, 64)
	return n
}

// unescaping only '"' and '\'
func convertString(s string) string {
	if s[0:1] == "`" {
		return s[1 : len(s)-1]
	}

	return unescape(s[1:len(s)-1], "\\\"")
}

// unescaping only '/'
func convertRegexp(s string) string {
	return unescape(s[1:len(s)-1], "/")
}

// match a token at the current position
func (l *eskipLex) matchToken() []string {
	m := l.rx.FindStringSubmatch(l.code)
	if len(m) == 0 {
		l.lastRaw = ""
		return m
	}

	l.lastRaw = m[0]
	l.code = l.code[len(m[0]):]
	return m
}

// get the matched token based on the matched capture group
func (l *eskipLex) getToken(m []string) (int, string) {
	for _, trx := range l.tokenRxs {
		s := m[trx.matchIndex]
		if len(s) != 0 {
			return trx.token, s
		}
	}

	return -1, ""
}

// lexer implementation
func (l *eskipLex) Lex(lval *eskipSymType) int {
	// done
	if len(l.code) == 0 {
		return -1
	}

	// step position
	l.lastPosition += len(l.lastRaw)
	m := l.matchToken()

	// no match, error, done
	if len(m) == 0 {
		l.Error("invalid token")
		return -1
	}

	t, s := l.getToken(m)
	lval.token = s
	l.lastToken = s

	return t
}

// error with approximate position
func (l *eskipLex) Error(err string) {
	l.err = errors.New(fmt.Sprintf(
		"parse failed after token %s, position %d: %s",
		l.lastToken, l.lastPosition, err))
}
