package publisher

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// UpdateFrontmatter reads a Markdown file, updates its YAML frontmatter losslessly
// to include uri, cid, and published: true, and writes it back, preserving the content
// and any YAML comments.
func UpdateFrontmatter(filePath, uri, cid string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Split into frontmatter and markdown body
	parts := bytes.SplitN(content, []byte("\n---"), 2)

	// Ensure we have a valid split (start marker could be '---' or first line '---')
	var frontmatterBytes []byte
	var bodyBytes []byte

	if bytes.HasPrefix(content, []byte("---\n")) {
		// Valid typical frontmatter format
		parts = bytes.SplitN(content[4:], []byte("\n---"), 2)
		if len(parts) == 2 {
			frontmatterBytes = parts[0]
			// The second part might start with '\n' depending on OS/spacing, let's keep everything after the delimiter
			bodyBytes = parts[1]
		}
	}

	if len(frontmatterBytes) == 0 {
		return fmt.Errorf("could not find valid YAML frontmatter in %s", filePath)
	}

	// Parse YAML losslessly using yaml.Node
	var root yaml.Node
	if err := yaml.Unmarshal(frontmatterBytes, &root); err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Root should be DocumentNode, with a single MappingNode child
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("invalid YAML structure in frontmatter")
	}

	mappingNode := root.Content[0]

	// Helper to find index of a key
	findKeyIndex := func(key string) int {
		for i := 0; i < len(mappingNode.Content); i += 2 {
			if mappingNode.Content[i].Value == key {
				return i
			}
		}
		return -1
	}

	// Update or append fields
	updateOrAppend := func(key, value string, isBool bool) {
		idx := findKeyIndex(key)
		if idx != -1 {
			// Update existing value
			mappingNode.Content[idx+1].Value = value
			if isBool {
				mappingNode.Content[idx+1].Tag = "!!bool"
			}
		} else {
			// Append new key-value pair
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
			valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
			if isBool {
				valNode.Tag = "!!bool"
			}
			mappingNode.Content = append(mappingNode.Content, keyNode, valNode)
		}
	}

	updateOrAppend("uri", uri, false)
	updateOrAppend("cid", cid, false)
	updateOrAppend("published", "true", true)

	// Marshal back
	var newFrontmatter bytes.Buffer
	encoder := yaml.NewEncoder(&newFrontmatter)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return fmt.Errorf("failed to encode frontmatter: %w", err)
	}
	encoder.Close()

	// Assemble final content
	var finalContent bytes.Buffer
	finalContent.WriteString("---\n")

	// Clean up newFrontmatter string since encode adds a trailing newline
	fmStr := newFrontmatter.String()
	fmStr = strings.TrimPrefix(fmStr, "---\n")
	finalContent.WriteString(fmStr)

	finalContent.WriteString("---")
	finalContent.Write(bodyBytes)

	// We can write back to file
	// Add some safety check, if bodyBytes doesn't start with newline we might need it

	return os.WriteFile(filePath, finalContent.Bytes(), 0644)
}
