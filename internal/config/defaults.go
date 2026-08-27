package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EnsureDefaults adds missing configuration keys to path without changing any
// value already present in the user's YAML file. It preserves YAML comments
// through yaml.Node and replaces the file atomically only when a key is added.
func EnsureDefaults(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read config for defaults: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return false, fmt.Errorf("parse config for defaults: %w", err)
	}
	if len(document.Content) == 0 {
		return false, fmt.Errorf("config document is empty")
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return false, fmt.Errorf("config root must be a YAML mapping")
	}

	changed, err := ensureYTDDefaults(root)
	if err != nil || !changed {
		return changed, err
	}

	updated, err := yaml.Marshal(&document)
	if err != nil {
		return false, fmt.Errorf("marshal config defaults: %w", err)
	}
	if err := atomicReplace(path, updated); err != nil {
		return false, err
	}
	return true, nil
}

func ensureYTDDefaults(root *yaml.Node) (bool, error) {
	ytd := mappingValue(root, "ytd")
	if ytd == nil {
		ytd = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		appendMappingPair(root, "ytd", ytd)
	}
	if ytd.Kind != yaml.MappingNode {
		return false, fmt.Errorf("ytd config must be a YAML mapping")
	}
	if mappingValue(ytd, "block_playlist_album_download") != nil {
		return false, nil
	}
	appendMappingPair(ytd, "block_playlist_album_download", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	return true, nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func appendMappingPair(mapping *yaml.Node, key string, value *yaml.Node) {
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
}

func atomicReplace(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat config for defaults: %w", err)
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".config.yaml.tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("preserve config permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write config defaults: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace config with defaults: %w", err)
	}
	return nil
}
