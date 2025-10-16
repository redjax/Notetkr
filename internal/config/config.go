package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

var K = koanf.New(".")

// Config holds the application configuration
type Config struct {
	ConfigFile string `koanf:"config.file"`
	NotesDir   string `koanf:"notes.dir"`
	JournalDir string `koanf:"journal.dir"`
	DataDir    string `koanf:"data.dir"`
}

func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	dataDir := filepath.Join(homeDir, ".notetkr")

	return &Config{
		ConfigFile: filepath.Join(dataDir, "notetkr.yml"),
		DataDir:    dataDir,
		NotesDir:   filepath.Join(dataDir, "notes"),
		JournalDir: filepath.Join(dataDir, "journal"),
	}
}

func LoadConfig(flagSet *pflag.FlagSet, configFile string) {
	if configFile != "" {
		parser, err := parserForFile(configFile)
		if err != nil {
			log.Fatalf("unsupported config file format: %v", err)
		}

		if err := K.Load(file.Provider(configFile), parser); err != nil {
			log.Printf("error loading config file: %v", err)
		}
	}

	// Load from environment variables
	K.Load(env.Provider("NOTETKR_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "NOTETKR_")), "_", ".", -1)
	}), nil)

	// Load from CLI args (highest precedence)
	K.Load(posflag.Provider(flagSet, ".", K), nil)
}

func parserForFile(path string) (koanf.Parser, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yml", ".yaml":
		return yaml.Parser(), nil
	case ".json":
		return json.Parser(), nil
	case ".toml":
		return toml.Parser(), nil
	case ".env":
		return dotenv.Parser(), nil
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}
