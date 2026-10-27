package keywords

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// CleanProfile filters out dimensions with empty keyword arrays or empty strings.
func CleanProfile(profile KeywordsProfile) KeywordsProfile {
	clean := make(KeywordsProfile)
	for k, list := range profile {
		var items []string
		for _, item := range list {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				items = append(items, trimmed)
			}
		}
		if len(items) > 0 {
			clean[k] = items
		}
	}
	return clean
}

// ExtractKeywordsFromMarkdown extracts the keywords mapping from a Markdown file's YAML frontmatter.
// Provides alias backward compatibility by ensuring canonical 4 anchor dimension keys are also populated.
func ExtractKeywordsFromMarkdown(rawContent []byte) (KeywordsProfile, error) {
	strContent := string(rawContent)
	if !strings.HasPrefix(strContent, "---\n") && !strings.HasPrefix(strContent, "---\r\n") {
		return make(KeywordsProfile), nil
	}
	parts := strings.SplitN(strContent, "---", 3)
	if len(parts) < 3 {
		return make(KeywordsProfile), nil
	}

	var meta struct {
		Keywords KeywordsProfile `yaml:"keywords"`
	}
	if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}
	if meta.Keywords == nil {
		return make(KeywordsProfile), nil
	}

	clean := CleanProfile(meta.Keywords)
	// Provide backward/forward compatibility alias mapping
	for k, items := range clean {
		canonical := NormalizeDimension(k)
		if canonical != k {
			if _, exists := clean[canonical]; !exists {
				clean[canonical] = append([]string{}, items...)
			}
		}
	}

	return clean, nil
}

// MergeKeywordsToMarkdown merges the given KeywordsProfile into the YAML frontmatter of rawContent.
// Existing frontmatter fields (id, type, created, updated, source, tags, images, etc.) and markdown
// body are preserved losslessly.
// AI extraction never overwrites or duplicates manual '*' entries.
func MergeKeywordsToMarkdown(rawContent []byte, profile KeywordsProfile) ([]byte, error) {
	strContent := string(rawContent)
	cleanProfile := CleanProfile(profile)

	hasFrontmatter := strings.HasPrefix(strContent, "---\n") || strings.HasPrefix(strContent, "---\r\n")
	var yamlStr string
	var body string

	if hasFrontmatter {
		parts := strings.SplitN(strContent, "---", 3)
		if len(parts) >= 3 {
			yamlStr = parts[1]
			body = parts[2]
		} else {
			hasFrontmatter = false
			body = strContent
		}
	} else {
		body = strContent
	}

	// Extract existing manual '*' keyword entries from existing frontmatter
	manualEntries := make(map[string][]string) // canonical dimension -> list of manual entries (preserving '*')
	manualSet := make(map[string]map[string]bool)

	if hasFrontmatter && strings.TrimSpace(yamlStr) != "" {
		var meta struct {
			Keywords KeywordsProfile `yaml:"keywords"`
		}
		if err := yaml.Unmarshal([]byte(yamlStr), &meta); err == nil && meta.Keywords != nil {
			for dim, list := range meta.Keywords {
				canonical := NormalizeDimension(dim)
				for _, item := range list {
					trimmed := strings.TrimSpace(item)
					if strings.HasSuffix(trimmed, "*") {
						manualEntries[canonical] = append(manualEntries[canonical], trimmed)
						if manualSet[canonical] == nil {
							manualSet[canonical] = make(map[string]bool)
						}
						norm := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(trimmed, "*")))
						manualSet[canonical][norm] = true
					}
				}
			}
		}
	}

	// Also account for any manual '*' entries already in cleanProfile
	for dim, list := range cleanProfile {
		canonical := NormalizeDimension(dim)
		for _, item := range list {
			trimmed := strings.TrimSpace(item)
			if strings.HasSuffix(trimmed, "*") {
				norm := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(trimmed, "*")))
				if manualSet[canonical] == nil {
					manualSet[canonical] = make(map[string]bool)
				}
				if !manualSet[canonical][norm] {
					manualEntries[canonical] = append(manualEntries[canonical], trimmed)
					manualSet[canonical][norm] = true
				}
			}
		}
	}

	// Merge keywords: manual '*' entries take precedence and are never overwritten or duplicated
	mergedProfile := make(KeywordsProfile)
	processedCanonical := make(map[string]bool)

	for dim, list := range cleanProfile {
		canonical := NormalizeDimension(dim)
		processedCanonical[canonical] = true

		var mergedList []string
		seenInDim := make(map[string]bool)

		// 1. Add manual entries first
		if manuals, ok := manualEntries[canonical]; ok {
			for _, m := range manuals {
				norm := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(m, "*")))
				if !seenInDim[norm] {
					seenInDim[norm] = true
					mergedList = append(mergedList, m)
				}
			}
		}

		// 2. Add incoming AI keywords, skipping duplicates of manual entries
		for _, item := range list {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			norm := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(trimmed, "*")))
			if seenInDim[norm] {
				continue
			}
			seenInDim[norm] = true
			mergedList = append(mergedList, trimmed)
		}

		if len(mergedList) > 0 {
			mergedProfile[dim] = mergedList
		}
	}

	// 3. Preserve any dimension with manual entries that was not in cleanProfile
	for canonical, manuals := range manualEntries {
		if !processedCanonical[canonical] && len(manuals) > 0 {
			mergedProfile[canonical] = append([]string{}, manuals...)
		}
	}

	var root yaml.Node
	if hasFrontmatter && strings.TrimSpace(yamlStr) != "" {
		if err := yaml.Unmarshal([]byte(yamlStr), &root); err != nil {
			return nil, fmt.Errorf("failed to parse existing frontmatter: %w", err)
		}
	} else {
		root = yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Tag:  "!!map",
				},
			},
		}
	}

	if len(root.Content) == 0 {
		root.Content = []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
			},
		}
	}

	mappingNode := root.Content[0]
	if mappingNode.Kind != yaml.MappingNode {
		mappingNode = &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
		root.Content[0] = mappingNode
	}

	// Prepare keywords YAML node
	var kwNode *yaml.Node
	if len(mergedProfile) == 0 {
		kwNode = &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
	} else {
		kwBytes, err := yaml.Marshal(mergedProfile)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal keywords profile: %w", err)
		}
		var kwDoc yaml.Node
		if err := yaml.Unmarshal(kwBytes, &kwDoc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal keywords doc: %w", err)
		}
		if len(kwDoc.Content) > 0 {
			kwNode = kwDoc.Content[0]
		} else {
			kwNode = &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
			}
		}
	}

	// Find or insert keywords field
	foundIndex := -1
	for i := 0; i < len(mappingNode.Content); i += 2 {
		if mappingNode.Content[i].Value == "keywords" {
			foundIndex = i
			break
		}
	}

	if foundIndex != -1 {
		mappingNode.Content[foundIndex+1] = kwNode
	} else {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "keywords",
		}
		mappingNode.Content = append(mappingNode.Content, keyNode, kwNode)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return nil, fmt.Errorf("failed to encode frontmatter: %w", err)
	}
	updatedYAML := buf.String()

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(updatedYAML)
	if !strings.HasSuffix(updatedYAML, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("---")

	if hasFrontmatter {
		sb.WriteString(body)
	} else {
		if body != "" {
			if !strings.HasPrefix(body, "\n") && !strings.HasPrefix(body, "\r\n") {
				sb.WriteString("\n\n")
			}
			sb.WriteString(body)
		} else {
			sb.WriteString("\n")
		}
	}

	return []byte(sb.String()), nil
}
