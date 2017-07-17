package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ignores errors if one of the slices is longer than the other
func compareTokenLexItems(got []Lexeme, wanted []Lexeme) error {
	minLen := min(len(got), len(wanted))
	for i := 0; i < minLen; i++ {
		if got[i].token != wanted[i].token {
			return fmt.Errorf("token %d of different type. Got '%s'(%d, '%s'). Wanted '%s'(%d, '%s')",
				i,
				got[i].token, got[i].token, got[i].content,
				wanted[i].token, wanted[i].token, wanted[i].content)
		}
	}
	return nil
}

func compareContentLexItems(got []Lexeme, wanted []Lexeme) error {
	minLen := min(len(got), len(wanted))
	for i := 0; i < minLen; i++ {
		if got[i].content != wanted[i].content {
			return fmt.Errorf("token %d of different content. Got '%s'. Wanted '%s'",
				i, got[i].content, wanted[i].content)
		}
	}
	return nil
}

func comparePosLexItems(got []Lexeme, wanted []Lexeme) error {
	minLen := min(len(got), len(wanted))
	for i := 0; i < minLen; i++ {
		if got[i].start.pos != wanted[i].start.pos {
			return fmt.Errorf("token %d of different start position. Got '%d'. Wanted '%d'",
				i, got[i].start.pos, wanted[i].start.pos)
		}
	}

	for i := 0; i < minLen; i++ {
		if got[i].end.pos != wanted[i].end.pos {
			return fmt.Errorf("token %d of different end position. Got '%d'. Wanted '%d'",
				i, got[i].end.pos, wanted[i].end.pos)
		}
	}

	return nil
}

func comparePositionLexItems(got []Lexeme, wanted []Lexeme) error {
	minLen := min(len(got), len(wanted))
	for i := 0; i < minLen; i++ {
		if got[i].start != wanted[i].start {
			return fmt.Errorf("token %d of different start position. Got '%s'. Wanted '%s'",
				i, got[i].start, wanted[i].start)
		}
	}

	for i := 0; i < minLen; i++ {
		if got[i].end != wanted[i].end {
			return fmt.Errorf("token %d of different end position. Got '%s'. Wanted '%s'",
				i, got[i].end, wanted[i].end)
		}
	}

	return nil
}

func compareLengthLexItems(got []Lexeme, wanted []Lexeme) error {
	if len(got) > len(wanted) {
		return fmt.Errorf("Got more items (%d) than wanted (%d). Got unexpected '%s' instead of EOF", len(got), len(wanted), got[len(wanted)].token)
	}
	if len(got) < len(wanted) {
		return fmt.Errorf("Got fewer items (%d) than wanted (%d) expecting '%s' instead of EOF", len(got), len(wanted), wanted[len(got)].token)
	}
	return nil
}

type CompareFunc ([]func(got []Lexeme, wanted []Lexeme) error)

func compareAll(functions []func(got []Lexeme, wanted []Lexeme) error, got []Lexeme, wanted []Lexeme) error {
	for _, fn := range functions {
		var err error = fn(got, wanted)
		if err != nil {
			return err
		}
	}
	return nil
}

func compareLexItems(got []Lexeme, wanted []Lexeme) error {
	cmpFuncs := []func(got []Lexeme, wanted []Lexeme) error{
		compareTokenLexItems,
		compareContentLexItems,
		compareLengthLexItems,
	}
	return compareAll(cmpFuncs, got, wanted)
}

func compareAllLexItems(got []Lexeme, wanted []Lexeme) error {
	cmpFuncs := []func(got []Lexeme, wanted []Lexeme) error{
		compareTokenLexItems,
		compareContentLexItems,
		compareLengthLexItems,
		comparePosLexItems,
	}
	return compareAll(cmpFuncs, got, wanted)
}

func compareAllPositionLexItems(got []Lexeme, wanted []Lexeme) error {
	cmpFuncs := []func(got []Lexeme, wanted []Lexeme) error{
		compareTokenLexItems,
		compareContentLexItems,
		compareLengthLexItems,
		comparePositionLexItems,
	}
	return compareAll(cmpFuncs, got, wanted)
}

func TestRuneReading(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("Test"))
	r, size, err := reader.ReadRune()
	if err != nil {
		t.Errorf("Error reading from rune")
	}
	if r != 'T' {
		t.Errorf("Expecting T")
	}
	if size != 1 {
		t.Errorf("Wrong size")
	}
}

type PosRune struct {
	Position
	r rune
}

func TestPosReader(t *testing.T) {
	testcases := map[string]struct {
		input  string
		wanted []PosRune
	}{
		"basic": {
			input: "Test",
			wanted: []PosRune{
				{
					Position: Position{
						col: 0,
						row: 0,
						pos: 0,
					},
					r: 'T',
				},
				{
					Position: Position{
						col: 1,
						row: 0,
						pos: 1,
					},
					r: 'e',
				}, {
					Position: Position{
						col: 2,
						row: 0,
						pos: 2,
					},
					r: 's',
				},
				{
					Position: Position{
						col: 3,
						row: 0,
						pos: 3,
					},
					r: 't',
				},
			}}}
	for name, tc := range testcases {
		reader := strings.NewReader(tc.input)
		pr := NewPosReader(reader)
		if pr.err != nil {
			t.Errorf("Unexpected error in PosReader TC '%s': %s", name, pr.err)
		}
		mlen := min(len(tc.input), len(tc.wanted))
		for i := 0; i < mlen; i++ {
			if !pr.Next() || pr.err != nil {
				t.Errorf("Unexpected error in PosReader TC '%s' token %d: %s", name, i, pr.err)
			}
			if pr.Position != tc.wanted[i].Position {
				t.Errorf("Position error in PosReader TC '%s' token %d. Got: %s. Wanted :%s", name, i, pr.Position, tc.wanted[i].Position)
			}
		}
		if len(tc.input) > len(tc.wanted) {
			t.Errorf("Error in PosReader TC '%s': Wanted %d tokens but got %d", name, len(tc.input), len(tc.wanted))
		} else if len(tc.input) > len(tc.wanted) {
			t.Errorf("Error in PosReader TC '%s': Got %d tokens but wanted %d", name, len(tc.input), len(tc.wanted))
		}

	}
}

func TestLexer(t *testing.T) {
	testcases := map[string]struct {
		compare func([]Lexeme, []Lexeme) error
		input   string
		wanted  []Lexeme
	}{
		"basic": {
			compare: compareLexItems,
			input:   "(((fundamental-mode . ido-find-file) . 8))",
			wanted: []Lexeme{
				{
					token:   OPAREN,
					content: "(",
				},
				{
					token:   OPAREN,
					content: "(",
				},
				{
					token:   OPAREN,
					content: "(",
				},
				{
					token:   IDENT,
					content: "fundamental-mode",
				},
				{
					token:   DOT,
					content: ".",
				},
				{
					token:   IDENT,
					content: "ido-find-file",
				},
				{
					token:   CPAREN,
					content: ")",
				},
				{
					token:   DOT,
					content: ".",
				},
				{
					token:   NUMBER,
					content: "8",
				},
				{
					token:   CPAREN,
					content: ")",
				},
				{
					token:   CPAREN,
					content: ")",
				},
			},
		},
		"mode-func": {
			compare: compareLexItems,
			input:   "(my-mode . my-function)",
			wanted: []Lexeme{
				{
					token:   OPAREN,
					content: "(",
				},
				{
					token:   IDENT,
					content: "my-mode",
				},
				{
					token:   DOT,
					content: ".",
				},
				{
					token:   IDENT,
					content: "my-function",
				},
				{
					token:   CPAREN,
					content: ")",
				},
			},
		},
		"simple": {
			compare: compareLexItems,
			input:   ")",
			wanted: []Lexeme{
				{
					token:   CPAREN,
					content: ")",
				},
			},
		},

		"pos": {
			compare: compareAllLexItems,
			input:   "(hello  world ",
			wanted: []Lexeme{
				{
					token:   OPAREN,
					content: "(",
					start: Position{
						pos: 0,
					},
					end: Position{
						pos: 1,
					},
				},
				{
					token:   IDENT,
					content: "hello",
					start: Position{
						pos: 1,
					},
					end: Position{
						pos: 6,
					},
				},
				{
					token:   IDENT,
					content: "world",
					start: Position{
						pos: 8,
					},
					end: Position{
						pos: 13,
					},
				},
			},
		},
		"position": {
			compare: compareAllPositionLexItems,
			input:   "( hello\n  world ",
			wanted: []Lexeme{
				{
					token:   OPAREN,
					content: "(",
					start: Position{
						pos: 0,
						row: 0,
						col: 0,
					},
					end: Position{
						pos: 1,
						row: 0,
						col: 1,
					},
				},
				{
					token:   IDENT,
					content: "hello",
					start: Position{
						pos: 2,
						row: 0,
						col: 2,
					},
					end: Position{
						pos: 7,
						row: 0,
						col: 7,
					},
				},
				{
					token:   IDENT,
					content: "world",
					start: Position{
						pos: 10,
						row: 1,
						col: 2,
					},
					end: Position{
						pos: 15,
						col: 7,
						row: 1,
					},
				},
			},
		},
	}

	for name, tc := range testcases {
		var got []Lexeme
		reader := strings.NewReader(tc.input)
		lexer := NewLexer(reader)

		for lexer.Next() {
			token := lexer.Scan()
			got = append(got, token)
		}
		err := tc.compare(got, tc.wanted)
		if err != nil {
			t.Errorf("Lexer TC '%s' failed: %s", name, err)
		}
	}
}

func TestToken(t *testing.T) {
	testcases := map[string]struct {
		input  Token
		wanted string
	}{
		"oparen token": {
			input:  OPAREN,
			wanted: "OPAREN",
		},
		"closed parenthesis": {
			input:  CPAREN,
			wanted: "CPAREN",
		},
		"dot": {
			input:  DOT,
			wanted: "DOT",
		},
		"ident": {
			input:  IDENT,
			wanted: "IDENT",
		},
		"number": {
			input:  NUMBER,
			wanted: "NUMBER",
		},
	}
	for name, tc := range testcases {
		got := tc.input.String()
		if got != tc.wanted {
			t.Errorf("%s: Got '%s' but wanted '%s'", name, got, tc.wanted)
		}
	}
}

func TestPosition(t *testing.T) {
	testcases := map[string]struct {
		pos    Position
		wanted string
	}{
		"position stringer": {
			pos: Position{
				pos: 3,
				col: 1,
				row: 2,
			},
			wanted: ":2:1 (3)",
		},
	}
	for name, tc := range testcases {
		got := fmt.Sprintf("%s", tc.pos)
		if got != tc.wanted {
			t.Errorf("%s: Got '%s' but wanted '%s'", name, got, tc.wanted)
		}
	}
}

func TestParserReadFunc(t *testing.T) {
	testcases := map[string]struct {
		input  string
		wanted ModeFunc
	}{
		"basic": {
			input: "(my-mode . my-function)",
			wanted: ModeFunc{
				Function: "my-function",
				Mode:     "my-mode",
			},
		},
	}
	for name, tc := range testcases {
		reader := strings.NewReader(tc.input)
		parser := new(Parser)
		parser.init(reader)

		got, err := parser.readModeFunction()
		if err != nil {
			t.Errorf("%s: unexpected error: '%s'", name, err)
			continue
		}
		if got != tc.wanted {
			t.Errorf("%s: Got '%s' but wanted '%s'", name, got, tc.wanted)
		}
	}
}

func TestOutMode(t *testing.T) {
	testcases := map[string]struct {
		input        string
		wanted       OutMode
		wantedString string
	}{
		"all": {
			input:        "all",
			wanted:       ALL,
			wantedString: "ALL",
		},
	}
	for name, tc := range testcases {
		om, err := OutModeParse(tc.input)
		if err != nil {
			t.Errorf("%s: readArgs returned unexpected error: %s", name, err)
			continue
		}
		if tc.wanted != om {
			t.Errorf("%s: parsing did not yield correct result. Wanted: '%s' Got: '%s'",
				name, tc.wanted, om)
		}
		toString := tc.wanted.String()
		if toString != tc.wantedString {
			t.Errorf("%s: String() did not yield correct result. Wanted: '%s' Got: '%s'",
				name, tc.wantedString, toString)
		}
	}
}

func TestOpts(t *testing.T) {
	path := "/home/.emacs.keyfreq"
	testcases := map[string]struct {
		input  []string
		wanted Opts
	}{
		"basic": {
			input: []string{"keyfreq", "-i", path},
			wanted: Opts{
				inputFilename: path,
				mode:          ALL,
			},
		},
		"modes": {
			input: []string{"keyfreq", "-i", path, "-mode", "modes"},
			wanted: Opts{
				inputFilename: path,
				mode:          MODES,
			},
		},
		"functions": {
			input: []string{"keyfreq", "-i", path, "-mode", "functions"},
			wanted: Opts{
				inputFilename: path,
				mode:          FUNCTIONS,
			},
		},
	}
	oldArgs := os.Args
	oldCmd := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
	}()
	for name, tc := range testcases {
		os.Args = tc.input
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

		var o Opts
		err := o.readArgs()
		if err != nil {
			t.Errorf("%s: readArgs returned unexpected error: %s", name, err)
			continue
		}
		if tc.wanted != o {
			t.Errorf("%s: Parsing Arguments failed. Wanted '%s'. Got '%s'",
				name, tc.wanted, o)
		}
	}
}
