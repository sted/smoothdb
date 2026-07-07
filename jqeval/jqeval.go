// Package jqeval provides bounded evaluation of jq programs via gojq.
//
// It is the shared core behind the smoothdb jq surfaces (POST /jq, jq-update,
// response transforms): compiled program cache, per-evaluation timeout,
// program size guard and the exactly-one-output convention.
//
// No custom functions and no I/O builtins are registered, so programs cannot
// touch anything beyond the JSON input they are handed.
package jqeval

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/itchyny/gojq"
)

// Error is the error type for all jq related failures (parse errors,
// evaluation errors, timeouts, guard violations)
type Error struct {
	msg string // description of error
}

func (e Error) Error() string { return e.msg }

// jqCache is an LRU cache of compiled jq programs. Variables bind at compile
// time in gojq, so entries are keyed by program + "\x00" + sorted variable names.
type jqCache struct {
	mu    sync.Mutex
	max   int
	ll    *list.List
	items map[string]*list.Element
}

type jqCacheEntry struct {
	key  string
	code *gojq.Code
	vars []string // sorted variable names ($-prefixed), Run values must follow this order
}

func newJQCache(max int) *jqCache {
	return &jqCache{max: max, ll: list.New(), items: map[string]*list.Element{}}
}

var cache = newJQCache(cacheEntries())

func (c *jqCache) get(key string) *jqCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*jqCacheEntry)
	}
	return nil
}

func (c *jqCache) put(entry *jqCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[entry.key]; ok {
		c.ll.MoveToFront(el)
		el.Value = entry
		return
	}
	c.items[entry.key] = c.ll.PushFront(entry)
	for c.ll.Len() > c.max {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.items, last.Value.(*jqCacheEntry).key)
	}
}

// normalizeArgs returns the args with "$"-prefixed keys and the sorted
// list of variable names
func normalizeArgs(args map[string]any) (map[string]any, []string) {
	if len(args) == 0 {
		return nil, nil
	}
	norm := make(map[string]any, len(args))
	vars := make([]string, 0, len(args))
	for k, v := range args {
		name := "$" + strings.TrimPrefix(k, "$")
		norm[name] = v
		vars = append(vars, name)
	}
	sort.Strings(vars)
	return norm, vars
}

// compile parses and compiles a program, going through the LRU cache.
// vars must be sorted and $-prefixed.
func compile(program string, vars []string) (*jqCacheEntry, error) {
	if strings.TrimSpace(program) == "" {
		return nil, &Error{"empty jq program"}
	}
	if len(program) > MaxProgramBytes() {
		return nil, &Error{"jq program exceeds the maximum allowed size (" +
			strconv.Itoa(MaxProgramBytes()) + " bytes)"}
	}
	key := program + "\x00" + strings.Join(vars, ",")
	if entry := cache.get(key); entry != nil {
		return entry, nil
	}
	q, err := gojq.Parse(program)
	if err != nil {
		return nil, &Error{"jq parse error: " + err.Error()}
	}
	var opts []gojq.CompilerOption
	if len(vars) > 0 {
		opts = append(opts, gojq.WithVariables(vars))
	}
	code, err := gojq.Compile(q, opts...)
	if err != nil {
		return nil, &Error{"jq compile error: " + err.Error()}
	}
	entry := &jqCacheEntry{key: key, code: code, vars: vars}
	cache.put(entry)
	return entry, nil
}

// Parse compile-checks a jq program with the given argument names, without
// evaluating it. Used for authoring-time validation (parse_only). Only the
// keys of args matter here; values may be nil.
func Parse(program string, args map[string]any) error {
	_, vars := normalizeArgs(args)
	_, err := compile(program, vars)
	return err
}

// Eval evaluates a jq program against a single input value, binding args as
// jq variables ({"step": 3} is accessible as $step). The input must be
// encoding/json-shaped (nil, bool, int, float64, *big.Int, string, []any,
// map[string]any - see Unmarshal).
//
// The program must produce exactly one output value: zero or multiple outputs
// are an error (programs wanting a stream can wrap it: [.items[] | ...]).
// Evaluation is bounded by the configured timeout.
func Eval(ctx context.Context, program string, input any, args map[string]any) (any, error) {
	norm, vars := normalizeArgs(args)
	entry, err := compile(program, vars)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(entry.vars))
	for i, name := range entry.vars {
		values[i] = norm[name]
	}
	dur := time.Duration(timeout()) * time.Millisecond
	tctx, cancel := context.WithTimeout(ctx, dur)
	defer cancel()
	iter := entry.code.RunWithContext(tctx, input, values...)
	var output any
	count := 0
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, &Error{"jq timeout: evaluation exceeded " + dur.String()}
			}
			return nil, &Error{"jq error: " + err.Error()}
		}
		count++
		if count > 1 {
			return nil, &Error{"jq program produced multiple outputs; exactly one is required (wrap streams in an array: [...])"}
		}
		output = v
	}
	if count == 0 {
		return nil, &Error{"jq program produced no output; exactly one is required"}
	}
	return output, nil
}

// Unmarshal decodes JSON bytes into the value shape gojq expects.
// Integers are preserved exactly (int or *big.Int instead of float64),
// so that ids and counters survive a jq round trip.
func Unmarshal(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var v any
	if err := decoder.Decode(&v); err != nil {
		return nil, &Error{"invalid JSON input for jq: " + err.Error()}
	}
	return normalizeNumbers(v), nil
}

// Marshal encodes a jq output value as JSON
func Marshal(v any) ([]byte, error) {
	b, err := gojq.Marshal(v)
	if err != nil {
		return nil, &Error{"cannot serialize jq output: " + err.Error()}
	}
	return b, nil
}

// normalizeNumbers converts json.Number values (produced by UseNumber)
// into gojq-compatible numbers: int, *big.Int or float64
func normalizeNumbers(v any) any {
	switch x := v.(type) {
	case json.Number:
		s := x.String()
		if !strings.ContainsAny(s, ".eE") {
			if i, err := strconv.Atoi(s); err == nil {
				return i
			}
			if b, ok := new(big.Int).SetString(s, 10); ok {
				return b
			}
		}
		f, err := x.Float64()
		if err != nil {
			return s
		}
		return f
	case map[string]any:
		for k, e := range x {
			x[k] = normalizeNumbers(e)
		}
	case []any:
		for i, e := range x {
			x[i] = normalizeNumbers(e)
		}
	}
	return v
}
