package jqeval

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestEvalBasic(t *testing.T) {
	ctx := context.Background()

	t.Run("Identity", func(t *testing.T) {
		out, err := Eval(ctx, ".", map[string]any{"a": 1}, nil)
		if err != nil {
			t.Fatal(err)
		}
		m, ok := out.(map[string]any)
		if !ok || m["a"] != 1 {
			t.Errorf("unexpected output: %v", out)
		}
	})

	t.Run("Expression", func(t *testing.T) {
		out, err := Eval(ctx, "{doubled: (.n * 2)}", map[string]any{"n": 21}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if out.(map[string]any)["doubled"] != 42 {
			t.Errorf("unexpected output: %v", out)
		}
	})

	t.Run("ParseError", func(t *testing.T) {
		_, err := Eval(ctx, ".foo |", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "jq parse error") {
			t.Errorf("expected parse error, got: %v", err)
		}
	})

	t.Run("RuntimeError", func(t *testing.T) {
		_, err := Eval(ctx, ".a + 1", map[string]any{"a": "x"}, nil)
		if err == nil || !strings.Contains(err.Error(), "jq error") {
			t.Errorf("expected runtime error, got: %v", err)
		}
	})

	t.Run("EmptyProgram", func(t *testing.T) {
		_, err := Eval(ctx, "  ", nil, nil)
		if err == nil {
			t.Error("expected error for empty program")
		}
	})
}

func TestEvalExactlyOneOutput(t *testing.T) {
	ctx := context.Background()

	t.Run("ZeroOutputs", func(t *testing.T) {
		_, err := Eval(ctx, "empty", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "no output") {
			t.Errorf("expected zero-output error, got: %v", err)
		}
	})

	t.Run("MultipleOutputs", func(t *testing.T) {
		_, err := Eval(ctx, ".[]", []any{1, 2}, nil)
		if err == nil || !strings.Contains(err.Error(), "multiple outputs") {
			t.Errorf("expected multiple-output error, got: %v", err)
		}
	})

	t.Run("WrappedStreamIsOK", func(t *testing.T) {
		out, err := Eval(ctx, "[.[] | . * 2]", []any{1, 2}, nil)
		if err != nil {
			t.Fatal(err)
		}
		arr := out.([]any)
		if len(arr) != 2 || arr[0] != 2 || arr[1] != 4 {
			t.Errorf("unexpected output: %v", out)
		}
	})
}

func TestEvalTimeout(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	_, err := Eval(ctx, "def f: f; f", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "jq timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestEvalArgs(t *testing.T) {
	ctx := context.Background()

	t.Run("SingleArg", func(t *testing.T) {
		out, err := Eval(ctx, ". + $delta", 40, map[string]any{"delta": 2})
		if err != nil {
			t.Fatal(err)
		}
		if out != 42 {
			t.Errorf("unexpected output: %v", out)
		}
	})

	t.Run("MultipleArgsSortedBinding", func(t *testing.T) {
		// two args, ensure each variable gets its own value (not positional mixups)
		out, err := Eval(ctx, "{a: $a, z: $z}", nil, map[string]any{"z": "zed", "a": "ay"})
		if err != nil {
			t.Fatal(err)
		}
		m := out.(map[string]any)
		if m["a"] != "ay" || m["z"] != "zed" {
			t.Errorf("unexpected output: %v", out)
		}
	})

	t.Run("DollarPrefixedKeys", func(t *testing.T) {
		out, err := Eval(ctx, "$x", nil, map[string]any{"$x": true})
		if err != nil {
			t.Fatal(err)
		}
		if out != true {
			t.Errorf("unexpected output: %v", out)
		}
	})

	t.Run("CacheKeyingWithArgs", func(t *testing.T) {
		// The same program with and without args must compile to different
		// cache entries: variables bind at compile time.
		program := ". + 1"
		out, err := Eval(ctx, program, 1, nil)
		if err != nil || out != 2 {
			t.Fatalf("plain eval failed: %v %v", out, err)
		}
		// same program, now with an (unused) variable: must not reuse the
		// no-args compiled code
		out, err = Eval(ctx, program, 1, map[string]any{"unused": 0})
		if err != nil || out != 2 {
			t.Fatalf("eval with args failed: %v %v", out, err)
		}
		// and again without args: must still work (cache hit on the first entry)
		out, err = Eval(ctx, program, 1, nil)
		if err != nil || out != 2 {
			t.Fatalf("second plain eval failed: %v %v", out, err)
		}
		// referencing a variable works only in the args-keyed entry
		_, err = Eval(ctx, "$v", nil, nil)
		if err == nil {
			t.Error("expected compile error for unbound variable")
		}
		out, err = Eval(ctx, "$v", nil, map[string]any{"v": 7})
		if err != nil || out != 7 {
			t.Errorf("eval with bound variable failed: %v %v", out, err)
		}
	})
}

func TestProgramSizeCap(t *testing.T) {
	ctx := context.Background()
	big := "." + strings.Repeat(" ", MaxProgramBytes()) + "a"
	_, err := Eval(ctx, big, map[string]any{"a": 1}, nil)
	if err == nil || !strings.Contains(err.Error(), "maximum allowed size") {
		t.Errorf("expected size cap error, got: %v", err)
	}
	if err := Parse(big, nil); err == nil {
		t.Error("expected size cap error from Parse")
	}
}

func TestParse(t *testing.T) {
	if err := Parse(".a.b", nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := Parse(".a |", nil); err == nil {
		t.Error("expected parse error")
	}
	// variable valid only if declared in args
	if err := Parse("$x + 1", nil); err == nil {
		t.Error("expected compile error for unbound variable")
	}
	if err := Parse("$x + 1", map[string]any{"x": nil}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCacheEviction(t *testing.T) {
	// a tiny cache still returns correct results while evicting
	old := cfg
	Configure(&Config{Enabled: true, CacheEntries: 2})
	defer Configure(&old)

	ctx := context.Background()
	programs := []string{". + 1", ". + 2", ". + 3", ". + 1"}
	expected := []any{11, 12, 13, 11}
	for i, p := range programs {
		out, err := Eval(ctx, p, 10, nil)
		if err != nil || out != expected[i] {
			t.Fatalf("program %q: got %v, %v", p, out, err)
		}
	}
	if cache.ll.Len() > 2 {
		t.Errorf("cache exceeded its size: %d", cache.ll.Len())
	}
}

func TestUnmarshal(t *testing.T) {
	t.Run("BigIntegersPreserved", func(t *testing.T) {
		// 2^60 + 1 is not representable as float64
		in := []byte(`{"id": 1152921504606846977, "f": 1.5}`)
		v, err := Unmarshal(in)
		if err != nil {
			t.Fatal(err)
		}
		out, err := Eval(context.Background(), ".", v, nil)
		if err != nil {
			t.Fatal(err)
		}
		b, err := Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "1152921504606846977") {
			t.Errorf("integer precision lost: %s", b)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		_, err := Unmarshal([]byte(`{"a":`))
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}
