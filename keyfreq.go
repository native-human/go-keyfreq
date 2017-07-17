package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"unicode"
)

type Token uint

const (
	OPAREN Token = iota
	CPAREN
	DOT
	IDENT
	NUMBER
)

func (t Token) String() string {
	switch t {
	case OPAREN:
		return "OPAREN"
	case CPAREN:
		return "CPAREN"
	case DOT:
		return "DOT"
	case IDENT:
		return "IDENT"
	case NUMBER:
		return "NUMBER"
	}
	panic(fmt.Sprintf("unexpected token value '%d'", t))
}

type Position struct {
	pos uint
	col uint
	row uint
}

func (p Position) String() string {
	return fmt.Sprintf(":%d:%d (%d)", p.row, p.col, p.pos)
}

type Lexeme struct {
	token   Token
	content string

	start Position
	end   Position
}

type PosReader struct {
	Position
	r       rune
	size    int
	colsize uint
	eof     bool
	err     PosError
	reader  *bufio.Reader
}

type Lexer struct {
	PosReader
	item     Lexeme
	startPos Position
	content  string
}

func (pr *PosReader) Next() bool {
	r, size, err := pr.reader.ReadRune()

	if err == io.EOF {
		pr.eof = true
		pr.size = size
		return false
	}

	pr.pos += uint(pr.size)
	if pr.r == '\n' { // XXX: care for CR as well
		pr.col = 0
		pr.row += 1
	} else {
		pr.col += pr.colsize
	}
	pr.colsize = 1
	pr.r = r
	pr.size = size

	if err != nil {
		pr.err = PosErrorf(pr.Position, "error while reading from stream: %s", err)
		return false
	}
	return true
}

func isIdentRune(r rune) bool {
	if !unicode.IsNumber(r) && !unicode.IsLetter(r) &&
		r != '-' && r != '+' && r != ':' && r != '*' && r != '&' && r != '/' {
		return false
	}
	return true
}

func (l *Lexer) newLexeme(token Token) {
	l.item = Lexeme{
		start:   l.startPos,
		end:     l.Position,
		content: l.content,
		token:   token,
	}
}

// return if the rune was matched with the current rune
// return true in case of an error so that the callee handles the error state.
func (l *Lexer) acceptRune(r rune, t Token) bool {
	if l.r == r {
		l.content = l.content + string(l.r)
		l.PosReader.Next()
		if l.err != nil {
			return true
		}
		l.newLexeme(t)
		return true
	}
	return false
}

// accept all subsequent runes that are accepted by fn. Return true if at least one
// rune is accepted
// return true in case of an error so that the callee handles the error state.
func (l *Lexer) acceptFunc(fn func(rune) bool, t Token) bool {
	if fn(l.r) {
		for !l.eof && fn(l.r) {
			l.content = l.content + string(l.r)
			if l.PosReader.Next(); l.err != nil {
				return true
			}

		}
		l.newLexeme(t)
		return true
	}
	return false
}

func (l *Lexer) Next() bool {
	// var content string
	l.content = ""
	if l.PosReader.eof {
		return false
	}

	// skip leading spaces
	for unicode.IsSpace(l.r) && l.PosReader.Next() {
	}
	if l.err != nil {
		return false
	}

	l.startPos = l.Position
	if l.acceptRune('(', OPAREN) {
		return l.err == nil
	}
	if l.acceptRune(')', CPAREN) {
		return l.err == nil
	}
	if l.acceptRune('.', DOT) {
		return l.err == nil
	}

	if l.acceptFunc(unicode.IsNumber, NUMBER) {
		return l.err == nil
	}
	if l.acceptFunc(isIdentRune, IDENT) {
		return l.err == nil
	}
	return false
}

func (l *Lexer) Scan() Lexeme {
	return l.item
}

type PosError interface {
	error
	GetRow() uint
	GetCol() uint
}

type LexPosError struct {
	Position
	msg string
}

func (e LexPosError) GetRow() uint {
	return e.Position.row
}

func (e LexPosError) GetCol() uint {
	return e.Position.col
}

func (e LexPosError) Error() string {
	return fmt.Sprintf(":%d:%d %s", e.row, e.col, e.msg)
}

func PosErrorf(pos Position, msg string, args ...interface{}) LexPosError {
	var err LexPosError
	err.Position = pos
	err.msg = fmt.Sprintf(msg, args...)
	return err
}

func NewPosReader(r io.Reader) PosReader {
	pr := PosReader{
		Position: Position{
			row: 0,
			col: 0,
			pos: 0,
		},
		eof:    false,
		reader: bufio.NewReader(r),
	}
	return pr
}

func NewLexer(r io.Reader) *Lexer {
	l := Lexer{
		PosReader: NewPosReader(r),
	}
	// l.PosReader.Next()
	l.r = ' '
	return &l
}

type Parser struct {
	lexer     *Lexer
	totalFunc map[string]uint64
	totalMode map[string]uint64
}

type ModeFunc struct {
	Function string
	Mode     string
}

func (p *Parser) readRoot() PosError {
	p.lexer.Next()
	if p.lexer.err != nil {
		return p.lexer.err
	}

	startItem := p.lexer.Scan()

	if startItem.token != OPAREN {
		return PosErrorf(startItem.start, "expected symbol '(' in readRoot but got '%s'", startItem.content)
	}

	var success bool = true
	for success {
		var err PosError
		success, err = p.readCount()
		if err != nil {
			return err
		}
	}

	endItem := p.lexer.Scan()
	if endItem.token != CPAREN {
		return PosErrorf(endItem.start, "expected symbol ')' in readRoot but got '%s'", endItem.content)
	}
	return nil
}

func (p *Parser) readModeFunction() (ModeFunc, PosError) {
	var mf ModeFunc
	p.lexer.Next()
	if p.lexer.err != nil {
		return mf, p.lexer.err
	}

	startParen := p.lexer.Scan()
	if startParen.token != OPAREN {
		return mf, PosErrorf(startParen.start, "expected symbol '('  in readMode but got '%s'", startParen.content)
	}

	p.lexer.Next()
	if p.lexer.err != nil {
		return mf, p.lexer.err
	}

	modeItem := p.lexer.Scan()

	if modeItem.token != IDENT {
		return mf, PosErrorf(modeItem.start, "expected IDENT but got '%s'", modeItem.content)
	}
	mf.Mode = modeItem.content

	p.lexer.Next()
	if p.lexer.err != nil {
		return mf, p.lexer.err
	}

	dot := p.lexer.Scan()
	if dot.token != DOT {
		return mf, PosErrorf(dot.start, "expected symbol '.' but got '%s'", dot.content)

	}

	p.lexer.Next()
	if p.lexer.err != nil {
		return mf, p.lexer.err
	}
	function := p.lexer.Scan()

	if function.token != IDENT {
		return mf, PosErrorf(function.start, "expected IDENT but got '%s'", function.content)
	}
	mf.Function = function.content

	p.lexer.Next()
	if p.lexer.err != nil {
		return mf, p.lexer.err
	}

	endParen := p.lexer.Scan()
	if endParen.token != CPAREN {
		return mf, PosErrorf(endParen.start, "expected symbol ')' in readMode but got '%s'", endParen.content)
	}
	return mf, nil
}

func (p *Parser) readCount() (bool, PosError) {
	p.lexer.Next()
	if p.lexer.err != nil {
		return false, p.lexer.err
	}

	startParen := p.lexer.Scan()
	if startParen.token != OPAREN {
		return false, nil
	}

	mf, err := p.readModeFunction()
	if err != nil {
		return false, err
	}

	p.lexer.Next()
	if p.lexer.err != nil {
		return false, p.lexer.err
	}

	dot := p.lexer.Scan()
	if dot.token != DOT {
		return false, PosErrorf(dot.start, "expected IDENT but got '%s'", dot.content)
	}

	p.lexer.Next()
	if p.lexer.err != nil {
		return false, p.lexer.err
	}

	count := p.lexer.Scan()
	if count.token != NUMBER {
		return false, PosErrorf(count.start, "expected number but got '%s'", count.content)
	}
	u, converr := strconv.ParseUint(count.content, 10, 64)
	if converr != nil {
		return false, PosErrorf(count.start, "can't convert count '%s' to unsigned integer: %s", count.content, err)
	}
	p.totalFunc[mf.Function] += u
	p.totalMode[mf.Mode] += u

	p.lexer.Next()
	if p.lexer.err != nil {
		return false, p.lexer.err
	}

	endParen := p.lexer.Scan()
	if endParen.token != CPAREN {
		return false, PosErrorf(endParen.start, "expected symbol ')' but got '%s'", endParen.content)
	}
	return true, nil
}

func (p *Parser) init(r io.Reader) {
	p.lexer = NewLexer(r)
	p.totalFunc = make(map[string]uint64)
	p.totalMode = make(map[string]uint64)
}

type Countee struct {
	key   string
	count uint64
}

type Countees []Countee

func (c Countees) Len() int {
	return len(c)
}

func (c Countees) Less(i, j int) bool {
	return c[i].count > c[j].count
}

func (c Countees) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (p *Parser) printFuncResults(w io.Writer) {
	var orderedFuncs Countees
	var total uint64 = 0
	for f, c := range p.totalFunc {
		function := Countee{
			key:   f,
			count: c,
		}
		orderedFuncs = append(orderedFuncs, function)
		total += c
	}
	sort.Sort(orderedFuncs)
	for _, countee := range orderedFuncs {
		fmt.Fprintf(w, "%s,%d,%f\n", countee.key, countee.count, 100.0*float64(countee.count)/float64(total))
	}
}

func (p *Parser) printModeResults(w io.Writer) {
	var orderedModes Countees
	var total uint64 = 0

	for m, c := range p.totalMode {
		mode := Countee{
			key:   m,
			count: c,
		}
		orderedModes = append(orderedModes, mode)
		total += c
	}

	sort.Sort(orderedModes)
	for _, countee := range orderedModes {
		fmt.Fprintf(w, "%s,%d,%f\n", countee.key, countee.count, 100.0*float64(countee.count)/float64(total))
	}
}

func (p *Parser) printResults() {
	fmt.Printf("\n\nFuncs\n------\n\n")
	p.printFuncResults(os.Stdout)
	fmt.Printf("\n\nModes\n------\n\n")
	p.printModeResults(os.Stdout)
}

type OutMode uint

const (
	ALL OutMode = iota
	MODES
	FUNCTIONS
)

func (om OutMode) String() string {
	switch om {
	case ALL:
		return "ALL"
	case MODES:
		return "MODES"
	case FUNCTIONS:
		return "FUNCTIONS"
	}
	panic(fmt.Sprintf("unexpected OutMode value '%d'", om))
}

func OutModeParse(value string) (OutMode, error) {
	switch value {
	case "all":
		return ALL, nil
	case "modes":
		return MODES, nil
	case "functions":
		return FUNCTIONS, nil
	default:
		return ALL, fmt.Errorf("don't know mode '%s'. Valid values are 'all', 'modes', 'functions'", value)
	}
}

type Opts struct {
	inputFilename string
	mode          OutMode
}

func (o *Opts) readArgs() error {
	flag.StringVar(&o.inputFilename, "i", path.Join(os.Getenv("HOME"), ".emacs.keyfreq"), "input filename")
	outMode := flag.String("mode", "all", "specify what to output. Choose between all, modes and functions")
	flag.Parse()

	var err error
	o.mode, err = OutModeParse(*outMode)
	if err != nil {
		return err
	}
	return nil
}

func Usage(message string, errcode int) {
	os.Exit(errcode)
}

func main() {
	var opts Opts
	err := opts.readArgs()
	if err != nil {
		Usage("message", 1)
	}
	file, err := os.Open(opts.inputFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var parser *Parser
	parser = new(Parser)
	parser.init(file)
	parser.readRoot()
	switch opts.mode {
	case ALL:
		parser.printResults()
	case MODES:
		parser.printModeResults(os.Stdout)
	case FUNCTIONS:
		parser.printFuncResults(os.Stdout)
	default:
		panic(fmt.Sprintf("Unknown mode: %d", opts.mode))
	}
}
