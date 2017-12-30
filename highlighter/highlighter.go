package highlighter

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"sync"

	"github.com/fatih/color"
	textdistance "github.com/masatana/go-textdistance"
	"github.com/pkg/errors"
)

// Distance is the Damerau–Levenshtein distance for a given word.  If a token's
// distance is less than or equal to the Distance, it will be highlighted using
// the specified Color.  If the Distance is 0, a strict case-insensitive
// comparison is performed.
type TokenColor struct {
	// Token is the keyword that will be searched for in the input stream.
	Token string

	// Submatch, when true, will cause a case insensitive search of the incoming
	// token.
	Submatch bool

	// Distance is the Damerau–Levenshtein distance to use when searching for a
	// token.  Distance is configured by extracting the number from the token name
	// itself, e.g. "foo~1" would have a distance of 1 for the token "foo".
	Distance int

	// Color is the color definition for a matching term.
	Color *color.Color
}

type Highlighter struct {
	w    io.Writer
	d    int
	toks []*TokenColor

	// ioLock guards buf and scanner
	ioLock sync.Mutex
	buf    bytes.Buffer
}

type NewInput struct {
	Writer io.Writer
	Tokens []*TokenColor
}

// New creates a new highlighter.  A Highlighter satisfies the io.Writer
// interface.  A Highlighter creates an internal bufio instance and scans the
// input for tokens.  Each token is evaluated to see if it is within the
// Damerau–Levenshtein distance.  If a token is equal or less than the
// Damerau–Levenshtein distance, the token is highlighted.
//
// TODO(seanc@): highlight the sentence containing the token.
func New(cfg NewInput) (*Highlighter, error) {
	h := &Highlighter{
		w:    cfg.Writer,
		toks: make([]*TokenColor, 0, len(cfg.Tokens)),
	}

	for _, input := range cfg.Tokens {
		input := input
		h.toks = append(h.toks, input)
	}

	return h, nil
}

// Flush writes any remaining bytes in the buffer.  If there are any un-written
// bytes in the buffer, return an error.
func (h *Highlighter) Flush() (err error) {
	h.ioLock.Lock()
	b := h.buf.Bytes()
	bLen := len(b)
	defer func() {
		h.ioLock.Lock()
		defer h.ioLock.Unlock()
	}()
	h.ioLock.Unlock()

	var n int
	if n, err = h.Write(b); err != nil {
		return errors.Wrap(err, "unable to flush buffer")
	}

	if n < bLen {
		return errors.New("unable to flush all bytes")
	}

	return nil
}

// Write tokenizes the input and Write()'s the ASCII-colorized result to the
// underlying io.Writer.  It is mandatory to call Flush() on the Highlighter to
// drain the remaining bytes.
func (h *Highlighter) Write(p []byte) (n int, err error) {
	h.ioLock.Lock()
	defer h.ioLock.Unlock()

	n, err = h.buf.Write(p)
	if n != len(p) {
		// Extraneous error checking: buf.Write will always succeeds or panics
		return n, errors.Wrap(err, "unable to write to buffer")
	}

	var bytesWritten int

	type replacement struct {
		old string
		new *TokenColor
	}
	const defaultNumReplacements = 16
	replacements := make([]replacement, 0, defaultNumReplacements)

	// 1) Scan for a line of input
	// 2) Scan inside of a line for tokens
	// 3) If a token matches, save the token for replacement
	// 4) Replace all tokens once tokenization is complete
	// 5) Write the updated line to the Writer
	buf := h.buf.Bytes()
	lineReader := bufio.NewReader(bytes.NewReader(buf))

	continueReading := true
	for continueReading {
		line, err := lineReader.ReadBytes('\n')
		switch {
		case err != nil && err == io.EOF:
			// NOTE(seanc@): There is a chance that on a short read, we could be
			// attempting to process a token that is currently at the EOF boundary
			// before we refill the buffer.  Because we don't know the size of the
			// input at this point, it's impossible for us to distinguish between a
			// real EOF and a partial read.  Ignore this problem for now because any
			// possible solution makes the code extremely ugly.  Maybe a path shows
			// itself to introduce a clean way to pass down the expected length and
			// read until we've sucked in the entire buffer or at least one newline.
			//
			// For now, given the ROI in the effort, if a token comes in that is a
			// partial read, we probably won't highlight that token correctly.

			continueReading = false
			defer h.buf.Reset()

		case err != nil && err != io.EOF:
			return bytesWritten, errors.Wrap(err, "unable to find newline in buffer")
		}

		tokS := bufio.NewScanner(bytes.NewReader(line))
		tokS.Split(bufio.ScanWords)
		for tokS.Scan() {
			tok := tokS.Text()
			for _, t := range h.toks {
				switch {
				case t.Submatch:
					if strings.Contains(strings.ToLower(tok), strings.ToLower(t.Token)) {
						replacements = append(replacements, replacement{
							old: tok,
							new: t,
						})
					}

				case t.Distance == 0:
					if strings.ToLower(tok) == strings.ToLower(t.Token) {
						replacements = append(replacements, replacement{
							old: tok,
							new: t,
						})
					}

				case t.Distance > 0:
					dist := textdistance.DamerauLevenshteinDistance(strings.ToLower(tok), strings.ToLower(t.Token))
					if dist <= t.Distance {
						replacements = append(replacements, replacement{
							old: tok,
							new: t,
						})
					}
				}
			}
		}
		if err := tokS.Err(); err != nil {
			return -1, errors.Wrap(err, "unable to scan for tokens in line")
		}

		// Replace the line
		replacementToks := make([]string, 0, 2*len(replacements))
		for _, repl := range replacements {
			replacementToks = append(replacementToks, repl.old)
			replacementToks = append(replacementToks, repl.new.Color.Sprintf("%s", repl.old))
		}
		replacer := strings.NewReplacer(replacementToks...)

		rN, err := replacer.WriteString(h.w, string(line))
		if err != nil {
			return bytesWritten, errors.Wrap(err, "unable to replace matching tokens in line")
		}
		bytesWritten += rN

		// if _, err := h.w.Write([]byte("\n")); err != nil {
		// 	return bytesWritten, errors.Wrap(err, "unable to write newline")
		// }

		bytesWritten += 1
	}

	// NOTE(seanc@): In order to satisfy the Writer interface, we must not return
	// a length different than len(p).  I'm not sure I agree with this.  By that I
	// mean, how is it possible to chain Write handlers where any given handler
	// could mutate the result and write more bytes than the input?  For now,
	// return bytesWritten, not len(p).  I'm quite concerned this will break
	// something upstream in a subtle way, but I think it is a specification flaw
	// to require that the Write() interface return `n <= len(p)` for bytes
	// written.
	return bytesWritten, nil
}
