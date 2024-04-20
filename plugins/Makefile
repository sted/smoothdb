GO := go

SRC_DIR := ./plugins
PLUGINS_DIR := ../_plugins

PLUGINS := $(shell find $(SRC_DIR) -type f -name 'go.mod' -exec dirname {} \; | xargs -I {} basename {})

all: $(PLUGINS)

$(PLUGINS_DIR):
	@mkdir -p $@

$(PLUGINS): 
	$(GO) build -trimpath -buildmode=plugin -o $(PLUGINS_DIR)/$@.plugin $(SRC_DIR)/$@/*.go

clean:
	@$(foreach plugin,$(PLUGINS),rm -f $(PLUGINS_DIR)/$(plugin).plugin;)

.PHONY: all clean
