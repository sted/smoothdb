package plugins

import (
	"fmt"
	"path/filepath"
	"plugin"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

// Host interface is the gateway for plugins to smoothdb basic subsystems
type Host interface {
	GetLogger() *logging.Logger
	GetRouter() *heligo.Router
	GetDBE() *database.DbEngine
}

// Plugin is the interface that plugins must implement
type Plugin interface {
	// Prepare is called during smoothdb initialization, when
	// the accessible subsystems are ready
	Prepare(Host) error
	// Run is called right after Prepare to run the plugin.
	// It can just return or spawn goroutines if necessary.
	Run() error
}

type hostProxy struct {
	Host
	logger *logging.Logger
}

func (hp hostProxy) GetLogger() *logging.Logger {
	return hp.logger
}

// PluginManager
type PluginManager struct {
	host    hostProxy
	dir     string
	plugins []string
	logger  *logging.Logger
}

// InitPluginManager initializes the Plugin Manager
func InitPluginManager(host Host, dir string, plugins []string) *PluginManager {
	zlogger := host.GetLogger().With().Str("domain", "PLUG").Logger()
	logger := &logging.Logger{Logger: &zlogger}
	return &PluginManager{hostProxy{host, logger}, dir, plugins, logger}
}

// pluginFromSymbol returns the Plugin exported by the plugin name, or an
// error when the exported symbol does not implement Plugin.
func pluginFromSymbol(name string, symbol plugin.Symbol) (Plugin, error) {
	pluginInstance, ok := symbol.(Plugin)
	if !ok || pluginInstance == nil {
		return nil, fmt.Errorf("not a proper plugin %q: its Plugin symbol is a %T, which does not implement plugins.Plugin", name, symbol)
	}
	return pluginInstance, nil
}

// Load prepares and runs plugins
func (pm PluginManager) Load() error {
	for _, name := range pm.plugins {
		begin := time.Now()

		path := filepath.Join(pm.dir, name) + ".plugin"

		p, err := plugin.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open plugin %q at %q (%w)", name, path, err)
		}
		symbol, err := p.Lookup("Plugin")
		if err != nil {
			return fmt.Errorf("cannot lookup plugin instance %q (%w)", name, err)
		}
		pluginInstance, err := pluginFromSymbol(name, symbol)
		if err != nil {
			return err
		}
		err = pluginInstance.Prepare(pm.host)
		if err != nil {
			return fmt.Errorf("error preparing plugin %q (%w)", name, err)
		}
		err = pluginInstance.Run()
		if err != nil {
			return fmt.Errorf("error running plugin %q (%w)", name, err)
		}
		pm.logger.Info().Dur("elapsed", time.Since(begin)).Msg(fmt.Sprintf("Loaded plugin %q", name))
	}
	return nil
}
