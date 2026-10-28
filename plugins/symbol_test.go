package plugins

import (
	"strings"
	"testing"
)

type fakePlugin struct{}

func (*fakePlugin) Prepare(Host) error { return nil }
func (*fakePlugin) Run() error         { return nil }

// The symbol a plugin exports was asserted to Plugin unchecked: a plugin
// exporting a value of another type panicked the server at startup.
func TestPluginFromSymbol(t *testing.T) {
	// control: a proper plugin
	if p, err := pluginFromSymbol("good", &fakePlugin{}); err != nil || p == nil {
		t.Fatalf("control: expected the plugin, got %v %v", p, err)
	}
	for _, symbol := range []any{42, fakePlugin{}, nil} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("symbol %#v: panicked: %v", symbol, r)
				}
			}()
			p, err := pluginFromSymbol("bad", symbol)
			if err == nil || p != nil {
				t.Errorf("symbol %#v: expected an error, got %v %v", symbol, p, err)
			} else if !strings.Contains(err.Error(), `"bad"`) {
				t.Errorf("symbol %#v: the error does not name the plugin: %v", symbol, err)
			}
		}()
	}
}
