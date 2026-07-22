package xcode

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	xcconfigAssignmentPattern = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*(?:\[[^\]\r\n]+\])*)(\s*)(\+=|\?=|=)(\s*)(.*?)([ \t]*)$`)
	xcconfigIncludePattern    = regexp.MustCompile(`^\s*#include(\?)?\s+"([^"]+)"\s*$`)
)

type xcconfigAssignment struct {
	lineIndex     int
	key           string
	baseKey       string
	value         string
	operator      string
	quote         string
	operatorStart int
	operatorEnd   int
	valueStart    int
	valueEnd      int
}

type xcconfigInclude struct {
	lineIndex int
	path      string
	optional  bool
}

type xcconfigDocument struct {
	lines       []string
	assignments []xcconfigAssignment
	includes    []xcconfigInclude
}

type xcconfigResolvedValue struct {
	value            string
	path             string
	found            bool
	exact            bool
	missingInherited bool
	conditionals     []xcconfigConditionalValue
}

type xcconfigConditionalValue struct {
	key      string
	value    string
	operator string
	path     string
}

func parseXCConfig(data []byte) (xcconfigDocument, error) {
	lines := splitLinesPreservingEndings(string(data))
	document := xcconfigDocument{lines: lines}
	inBlockComment := false

	for index, line := range lines {
		body := strings.TrimSuffix(line, "\n")
		body = strings.TrimSuffix(body, "\r")
		masked, nextInBlock := maskXCConfigComments(body, inBlockComment)
		inBlockComment = nextInBlock

		if matches := xcconfigIncludePattern.FindStringSubmatch(masked); matches != nil {
			document.includes = append(document.includes, xcconfigInclude{
				lineIndex: index,
				path:      matches[2],
				optional:  matches[1] == "?",
			})
			continue
		}

		indices := xcconfigAssignmentPattern.FindStringSubmatchIndex(masked)
		if indices == nil {
			continue
		}
		key := masked[indices[4]:indices[5]]
		operatorStart, operatorEnd := indices[8], indices[9]
		valueStart, valueEnd := indices[12], indices[13]
		value, quote := parseXCConfigValue(body[valueStart:valueEnd])
		document.assignments = append(document.assignments, xcconfigAssignment{
			lineIndex:     index,
			key:           key,
			baseKey:       xcconfigBaseKey(key),
			value:         value,
			operator:      body[operatorStart:operatorEnd],
			quote:         quote,
			operatorStart: operatorStart,
			operatorEnd:   operatorEnd,
			valueStart:    valueStart,
			valueEnd:      valueEnd,
		})
	}

	if inBlockComment {
		return xcconfigDocument{}, fmt.Errorf("unterminated block comment in xcconfig")
	}
	return document, nil
}

func parseXCConfigValue(raw string) (string, string) {
	value := strings.TrimSpace(raw)
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		return value[1 : len(value)-1], string(value[0])
	}
	return value, ""
}

func splitLinesPreservingEndings(value string) []string {
	if value == "" {
		return []string{""}
	}
	lines := strings.SplitAfter(value, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func maskXCConfigComments(line string, inBlockComment bool) (string, bool) {
	masked := []byte(line)
	inQuote := byte(0)
	escaped := false

	for index := 0; index < len(masked); index++ {
		if inBlockComment {
			masked[index] = ' '
			if index+1 < len(masked) && line[index] == '*' && line[index+1] == '/' {
				masked[index+1] = ' '
				index++
				inBlockComment = false
			}
			continue
		}

		character := line[index]
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
				continue
			}
			if character == inQuote {
				inQuote = 0
			}
			continue
		}

		if character == '"' || character == '\'' {
			inQuote = character
			continue
		}
		if index+1 >= len(masked) {
			continue
		}
		if line[index:index+2] == "//" {
			for rest := index; rest < len(masked); rest++ {
				masked[rest] = ' '
			}
			break
		}
		if line[index:index+2] == "/*" {
			masked[index] = ' '
			masked[index+1] = ' '
			index++
			inBlockComment = true
		}
	}
	return string(masked), inBlockComment
}

func xcconfigBaseKey(key string) string {
	if index := strings.Index(key, "["); index >= 0 {
		return key[:index]
	}
	return key
}

func resolveXCConfigInclude(containingPath string, include xcconfigInclude) (string, error) {
	if strings.Contains(include.path, "$(") || strings.Contains(include.path, "${") {
		return "", fmt.Errorf("xcconfig include contains unresolved build setting: %s", include.path)
	}
	path := include.path
	if filepath.Ext(path) == "" {
		path += ".xcconfig"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(containingPath), path)
	}
	return filepath.Clean(path), nil
}

func collectXCConfigFiles(root string) ([]string, error) {
	seen := make(map[string]bool)
	var paths []string
	var visit func(string, map[string]bool) error
	visit = func(path string, stack map[string]bool) error {
		path = filepath.Clean(path)
		if stack[path] || seen[path] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		document, err := parseXCConfig(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		seen[path] = true
		paths = append(paths, path)
		nextStack := clonePathSet(stack)
		nextStack[path] = true
		for _, include := range document.includes {
			includePath, err := resolveXCConfigInclude(path, include)
			if err != nil {
				return err
			}
			if _, err := os.Stat(includePath); err != nil {
				if include.optional && os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("read xcconfig include %s: %w", includePath, err)
			}
			if err := visit(includePath, nextStack); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root, make(map[string]bool)); err != nil {
		return nil, err
	}
	return paths, nil
}

func resolveXCConfigSetting(root, setting string) (xcconfigResolvedValue, error) {
	return resolveXCConfigSettingWithBase(root, setting, xcconfigResolvedValue{})
}

func resolveXCConfigSettingWithBase(root, setting string, base xcconfigResolvedValue) (xcconfigResolvedValue, error) {
	resolved, conditional, err := resolveXCConfigSettingRecursive(
		filepath.Clean(root), setting, make(map[string]bool), base,
	)
	if err != nil {
		return xcconfigResolvedValue{}, err
	}
	if !resolved.exact && conditional {
		return xcconfigResolvedValue{}, fmt.Errorf(
			"%s is defined only by conditional xcconfig assignments; SDK-aware resolution requires Xcode",
			setting,
		)
	}
	if resolved.exact {
		for _, conditionalValue := range resolved.conditionals {
			if conditionalValue.operator != "=" || strings.TrimSpace(conditionalValue.value) != strings.TrimSpace(resolved.value) {
				return xcconfigResolvedValue{}, fmt.Errorf(
					"%s has differing conditional xcconfig assignment %s %s %q in %s (unconditional value %q); narrow the scope or use Xcode-aware resolution",
					setting,
					conditionalValue.key,
					conditionalValue.operator,
					conditionalValue.value,
					conditionalValue.path,
					resolved.value,
				)
			}
		}
	}
	if resolved.missingInherited {
		return xcconfigResolvedValue{}, fmt.Errorf("%s uses $(inherited) without a lower-layer value", setting)
	}
	return resolved, nil
}

func resolveXCConfigSettingRecursive(
	path string,
	setting string,
	stack map[string]bool,
	resolved xcconfigResolvedValue,
) (xcconfigResolvedValue, bool, error) {
	if stack[path] {
		return resolved, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return xcconfigResolvedValue{}, false, err
	}
	document, err := parseXCConfig(data)
	if err != nil {
		return xcconfigResolvedValue{}, false, fmt.Errorf("parse %s: %w", path, err)
	}
	nextStack := clonePathSet(stack)
	nextStack[path] = true

	type event struct {
		line       int
		assignment *xcconfigAssignment
		include    *xcconfigInclude
	}
	var events []event
	for index := range document.assignments {
		assignment := &document.assignments[index]
		events = append(events, event{line: assignment.lineIndex, assignment: assignment})
	}
	for index := range document.includes {
		include := &document.includes[index]
		events = append(events, event{line: include.lineIndex, include: include})
	}
	sort.SliceStable(events, func(left, right int) bool {
		return events[left].line < events[right].line
	})

	conditionalFound := false
	for _, item := range events {
		if item.include != nil {
			includePath, err := resolveXCConfigInclude(path, *item.include)
			if err != nil {
				return xcconfigResolvedValue{}, false, err
			}
			if _, err := os.Stat(includePath); err != nil {
				if item.include.optional && os.IsNotExist(err) {
					continue
				}
				return xcconfigResolvedValue{}, false, fmt.Errorf("read xcconfig include %s: %w", includePath, err)
			}
			included, includedConditional, err := resolveXCConfigSettingRecursive(includePath, setting, nextStack, resolved)
			if err != nil {
				return xcconfigResolvedValue{}, false, err
			}
			resolved = included
			conditionalFound = conditionalFound || includedConditional
			continue
		}

		assignment := item.assignment
		if assignment.baseKey != setting {
			continue
		}
		if assignment.key != setting {
			conditionalFound = true
			resolved.conditionals = append(resolved.conditionals, xcconfigConditionalValue{
				key:      assignment.key,
				value:    assignment.value,
				operator: assignment.operator,
				path:     path,
			})
			continue
		}
		value := assignment.value
		hadLowerValue := resolved.found
		hasInherited := strings.Contains(value, "$(inherited)") || strings.Contains(value, "${inherited}")
		value = strings.ReplaceAll(value, "$(inherited)", resolved.value)
		value = strings.ReplaceAll(value, "${inherited}", resolved.value)
		switch assignment.operator {
		case "?=":
			if resolved.found {
				continue
			}
		case "+=":
			if !hasInherited {
				value = strings.TrimSpace(strings.TrimSpace(resolved.value) + " " + strings.TrimSpace(value))
			}
		}
		resolved = xcconfigResolvedValue{
			value:            strings.TrimSpace(value),
			path:             path,
			found:            true,
			exact:            true,
			missingInherited: hasInherited && !hadLowerValue,
			conditionals:     resolved.conditionals,
		}
	}
	return resolved, conditionalFound, nil
}

func editXCConfig(data []byte, setting, value string) ([]byte, []string, bool, error) {
	document, err := parseXCConfig(data)
	if err != nil {
		return nil, nil, false, err
	}
	assignmentsByLine := make(map[int]xcconfigAssignment)
	var oldValues []string
	for _, assignment := range document.assignments {
		if assignment.baseKey != setting {
			continue
		}
		assignmentsByLine[assignment.lineIndex] = assignment
		oldValues = append(oldValues, assignment.value)
	}
	if len(assignmentsByLine) == 0 {
		return data, nil, false, nil
	}

	changed := false
	for index, assignment := range assignmentsByLine {
		line := document.lines[index]
		if assignment.value == value && assignment.operator == "=" {
			continue
		}
		quotedValue := assignment.quote + value + assignment.quote
		document.lines[index] = line[:assignment.operatorStart] + "=" +
			line[assignment.operatorEnd:assignment.valueStart] + quotedValue + line[assignment.valueEnd:]
		changed = true
	}
	return []byte(strings.Join(document.lines, "")), oldValues, changed, nil
}

func clonePathSet(source map[string]bool) map[string]bool {
	clone := make(map[string]bool, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
