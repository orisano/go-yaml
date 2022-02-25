package parser

import (
	"strconv"
	"strings"

	"github.com/goccy/go-yaml/token"
)

// context context at parsing
type context struct {
	idx    int
	size   int
	tokens token.Tokens
	mode   Mode

	paths     []string
	isDirty   bool
	pathCache string
}

var pathSpecialChars = "$*.[]"

func containsPathSpecialChar(path string) bool {
	return strings.ContainsAny(path, pathSpecialChars)
}

func normalizePath(path string) string {
	if containsPathSpecialChar(path) {
		return "'" + path + "'"
	}
	return path
}

func (c *context) path() string {
	if c.isDirty {
		c.pathCache = strings.Join(c.paths, "")
		c.isDirty = false
	}
	return c.pathCache
}

func (c *context) childPath(path string) string {
	return c.path() + "." + normalizePath(path)
}

func (c *context) withChild(path string) *context {
	c.paths = append(c.paths, "."+normalizePath(path))
	c.isDirty = true
	return c
}

func (c *context) withIndex(idx uint) *context {
	c.paths = append(c.paths, "["+strconv.FormatUint(uint64(idx), 10)+"]")
	c.isDirty = true
	return c
}

func (c *context) indexPath(idx uint) string {
	return c.path() + "[" + strconv.FormatUint(uint64(idx), 10) + "]"
}

func (c *context) popPath() {
	if len(c.paths) > 0 {
		c.paths = c.paths[:len(c.paths)-1]
		c.isDirty = true
	}
}

func (c *context) next() bool {
	return c.idx < c.size
}

func (c *context) previousToken() *token.Token {
	if c.idx > 0 {
		return c.tokens[c.idx-1]
	}
	return nil
}

func (c *context) insertToken(idx int, tk *token.Token) {
	if c.size < idx {
		return
	}
	if c.size == idx {
		curToken := c.tokens[c.size-1]
		tk.Next = curToken
		curToken.Prev = tk

		c.tokens = append(c.tokens, tk)
		c.size = len(c.tokens)
		return
	}

	curToken := c.tokens[idx]
	tk.Next = curToken
	curToken.Prev = tk

	c.tokens = append(c.tokens[:idx+1], c.tokens[idx:]...)
	c.tokens[idx] = tk
	c.size = len(c.tokens)
}

func (c *context) currentToken() *token.Token {
	if c.idx >= c.size {
		return nil
	}
	return c.tokens[c.idx]
}

func (c *context) nextToken() *token.Token {
	if c.idx+1 >= c.size {
		return nil
	}
	return c.tokens[c.idx+1]
}

func (c *context) afterNextToken() *token.Token {
	if c.idx+2 >= c.size {
		return nil
	}
	return c.tokens[c.idx+2]
}

func (c *context) nextNotCommentToken() *token.Token {
	for i := c.idx + 1; i < c.size; i++ {
		tk := c.tokens[i]
		if tk.Type == token.CommentType {
			continue
		}
		return tk
	}
	return nil
}

func (c *context) afterNextNotCommentToken() *token.Token {
	notCommentTokenCount := 0
	for i := c.idx + 1; i < c.size; i++ {
		tk := c.tokens[i]
		if tk.Type == token.CommentType {
			continue
		}
		notCommentTokenCount++
		if notCommentTokenCount == 2 {
			return tk
		}
	}
	return nil
}

func (c *context) enabledComment() bool {
	return c.mode&ParseComments != 0
}

func (c *context) isCurrentCommentToken() bool {
	tk := c.currentToken()
	if tk == nil {
		return false
	}
	return tk.Type == token.CommentType
}

func (c *context) progressIgnoreComment(num int) {
	if c.size <= c.idx+num {
		c.idx = c.size
	} else {
		c.idx += num
	}
}

func (c *context) progress(num int) {
	if c.isCurrentCommentToken() {
		return
	}
	c.progressIgnoreComment(num)
}

func newContext(tokens token.Tokens, mode Mode) *context {
	filteredTokens := []*token.Token{}
	if mode&ParseComments != 0 {
		filteredTokens = tokens
	} else {
		for _, tk := range tokens {
			if tk.Type == token.CommentType {
				continue
			}
			// keep prev/next reference between tokens containing comments
			// https://github.com/goccy/go-yaml/issues/254
			filteredTokens = append(filteredTokens, tk)
		}
	}
	return &context{
		idx:     0,
		size:    len(filteredTokens),
		tokens:  filteredTokens,
		mode:    mode,
		paths:   []string{"$"},
		isDirty: true,
	}
}
