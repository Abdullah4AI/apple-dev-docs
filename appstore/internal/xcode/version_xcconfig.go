package xcode

import (
	"errors"
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
	continued     bool
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

	for index := 0; index < len(lines); index++ {
		line := lines[index]
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
		joined := body[valueStart:valueEnd]
		endIndex := index
		logical, _ := maskXCConfigComments(joined, false)
		value, quote, err := parseXCConfigValue(logical)
		for err != nil && xcconfigValueHasLineContinuation(joined) && endIndex+1 < len(lines) {
			endIndex++
			nextBody := strings.TrimSuffix(lines[endIndex], "\n")
			nextBody = strings.TrimSuffix(nextBody, "\r")
			joined = trimXCConfigLineContinuation(joined) + nextBody
			logical, _ = maskXCConfigComments(joined, false)
			value, quote, err = parseXCConfigValue(logical)
		}
		if err != nil {
			return xcconfigDocument{}, fmt.Errorf("xcconfig line %d: %w", index+1, err)
		}
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
			continued:     endIndex > index || xcconfigValueHasLineContinuation(masked[valueStart:valueEnd]),
		})
		index = endIndex
	}

	if inBlockComment {
		return xcconfigDocument{}, fmt.Errorf("unterminated block comment in xcconfig")
	}
	return document, nil
}

func xcconfigValueHasLineContinuation(value string) bool {
	trimmed := strings.TrimRight(value, " \t")
	backslashes := 0
	for index := len(trimmed) - 1; index >= 0 && trimmed[index] == '\\'; index-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func trimXCConfigLineContinuation(value string) string {
	trimmed := strings.TrimRight(value, " \t")
	if !xcconfigValueHasLineContinuation(trimmed) {
		return value
	}
	return strings.TrimSuffix(trimmed, "\\")
}

func parseXCConfigValue(raw string) (string, string, error) {
	value := strings.TrimSpace(raw)
	if err := validateXCConfigValueQuotes(value); err != nil {
		return "", "", err
	}
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		decoded, err := decodeXCConfigQuotedValue(value[1:len(value)-1], value[0])
		if err != nil {
			return "", "", err
		}
		return decoded, string(value[0]), nil
	}
	return value, "", nil
}

// decodeXCConfigQuotedValue reverses the escaping emitted by
// quoteXCConfigValue. Backslashes are significant only inside a quoted value:
// a doubled backslash represents one literal backslash, and a backslash before
// the matching delimiter represents that delimiter. Unquoted values retain
// their existing literal backslashes and continuation behavior.
func decodeXCConfigQuotedValue(value string, quote byte) (string, error) {
	var decoded strings.Builder
	decoded.Grow(len(value))
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' {
			decoded.WriteByte(value[index])
			continue
		}
		if index+1 >= len(value) {
			return "", fmt.Errorf("dangling escape in quoted xcconfig value")
		}
		next := value[index+1]
		if next != '\\' && next != quote {
			// Version and signing parsers share this decoder. Preserve
			// generic escapes such as `\ ` so an unrelated assignment cannot
			// abort the whole document.
			decoded.WriteByte('\\')
			decoded.WriteByte(next)
			index++
			continue
		}
		decoded.WriteByte(next)
		index++
	}
	return decoded.String(), nil
}

func validateXCConfigValueQuotes(value string) error {
	var quote byte
	escaped := false
	for index := 0; index < len(value); index++ {
		character := value[index]
		if quote == 0 {
			if (character == '"' || character == '\'') && xcconfigQuoteStartsAt(value, index) {
				quote = character
			}
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if character == '\\' {
			escaped = true
			continue
		}
		if character == quote {
			quote = 0
		}
	}
	if quote != 0 {
		return fmt.Errorf("unterminated quote %q in xcconfig value", string(quote))
	}
	return nil
}

func xcconfigQuoteStartsAt(value string, index int) bool {
	if index == 0 {
		return true
	}
	switch value[index-1] {
	case ' ', '\t', '=':
		return true
	default:
		return false
	}
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

		if (character == '"' || character == '\'') && xcconfigQuoteStartsAt(line, index) {
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
	return collectXCConfigFilesWithReader(root, os.ReadFile, nil)
}

// collectStableXCConfigFiles walks a stable version command's include graph
// with per-directory case semantics. Stable commands retain ordinary
// os.Stat/os.ReadFile behavior (including selected symlinks), but a Windows
// case-sensitive directory must not collapse two case-distinct include paths
// into one traversal key.
func collectStableXCConfigFiles(root string) ([]string, error) {
	var identify func(string) (os.FileInfo, error)
	if xcconfigUsesIdentityTraversal() {
		identify = os.Stat
	}
	return collectXCConfigFilesWithHooksAndIdentity(root, os.ReadFile, nil, nil, nil, identify)
}

// collectXCConfigFilesWithReader walks an xcconfig include graph using the
// caller's reader and authorization hook. Security-sensitive callers should
// authorize every path before the reader or existence check touches it.
func collectXCConfigFilesWithReader(root string, read func(string) ([]byte, error), authorize func(string) error) ([]string, error) {
	return collectXCConfigFilesWithHooks(root, read, authorize, nil, nil)
}

// collectXCConfigFilesWithHooks is the instrumentable form of the collector.
// onPath runs for every normalized root or include target before authorization,
// stat, or read. onError receives the path responsible for a collection
// failure. Security-sensitive callers use these hooks to retain lexical
// provenance even when a path is missing or malformed and therefore never
// becomes part of the successfully collected file list.
func collectXCConfigFilesWithHooks(
	root string,
	read func(string) ([]byte, error),
	authorize func(string) error,
	onPath func(string),
	onError func(string, error),
) ([]string, error) {
	return collectXCConfigFilesWithHooksAndIdentity(root, read, authorize, onPath, onError, nil)
}

// collectXCConfigFilesWithHooksAndIdentity is the signing-specific collector
// extension for filesystems whose path spelling semantics can vary by
// directory. The identity callback runs only after authorization and lets a
// caller distinguish two case-variant paths that are distinct files without
// weakening the no-read-before-authorization contract.
func collectXCConfigFilesWithHooksAndIdentity(
	root string,
	read func(string) ([]byte, error),
	authorize func(string) error,
	onPath func(string),
	onError func(string, error),
	identify func(string) (os.FileInfo, error),
) ([]string, error) {
	return collectXCConfigFilesWithHooksAndIdentityAndOptionalMissing(root, read, authorize, onPath, onError, identify, nil)
}

// collectXCConfigFilesWithHooksAndIdentityAndOptionalMissing is the
// instrumentable collector used by signing plan generation. onOptionalMissing
// receives each lexically resolved optional include whose target is absent.
// The callback runs after the authorization check and before the missing target
// is ignored, so callers can persist an absence assertion without granting
// access to an untrusted path.
func collectXCConfigFilesWithHooksAndIdentityAndOptionalMissing(
	root string,
	read func(string) ([]byte, error),
	authorize func(string) error,
	onPath func(string),
	onError func(string, error),
	identify func(string) (os.FileInfo, error),
	onOptionalMissing func(string),
) ([]string, error) {
	seen := make(map[string]bool)
	type collectedIdentity struct {
		path string
		info os.FileInfo
	}
	var collected []collectedIdentity
	var paths []string
	traversalKey := func(path string) string {
		// With an identity callback, preserve the exact spelling. Identity
		// checks below can safely coalesce an alias only after the platform's
		// directory case semantics prove that the spellings name one path. A
		// generic collector has no such proof and retains its historical
		// platform lexical key for cycle protection.
		if identify != nil {
			return normalizeSigningLexicalPath(path)
		}
		return signingLexicalPathKey(path)
	}
	var visit func(string, map[string][]os.FileInfo) (error, bool)
	visit = func(path string, stack map[string][]os.FileInfo) (error, bool) {
		path = filepath.Clean(path)
		pathKey := traversalKey(path)
		if onPath != nil {
			onPath(path)
		}
		if authorize != nil {
			if err := authorize(path); err != nil {
				if onError != nil {
					onError(path, err)
				}
				return err, false
			}
		}
		var identity os.FileInfo
		if identify != nil {
			var err error
			identity, err = identify(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				if onError != nil {
					onError(path, err)
				}
				return err, false
			}
		}
		sameIdentity := func(infos []os.FileInfo) bool {
			if identity == nil {
				return false
			}
			for _, info := range infos {
				if info != nil && os.SameFile(identity, info) {
					return true
				}
			}
			return false
		}
		// Case variants can refer to one inode on a case-insensitive volume.
		// Coalesce only spellings that differ by case: general hard links may
		// live in different directories, where their relative includes resolve
		// against different bases and must still be traversed independently.
		if identity != nil {
			for _, entry := range collected {
				if signingPathCaseEquivalent(entry.path, path) && entry.info != nil && os.SameFile(identity, entry.info) {
					return nil, false
				}
			}
		}
		if stackInfos, ok := stack[pathKey]; ok {
			if identify == nil || sameIdentity(stackInfos) {
				return nil, false
			}
		}
		if seen[pathKey] {
			if identify == nil {
				return nil, false
			}
			for _, entry := range collected {
				if traversalKey(entry.path) == pathKey && entry.info != nil && os.SameFile(identity, entry.info) {
					return nil, false
				}
			}
		}
		data, err := read(path)
		if err != nil {
			if onError != nil {
				onError(path, err)
			}
			return err, errors.Is(err, os.ErrNotExist)
		}
		document, err := parseXCConfig(data)
		if err != nil {
			if onError != nil {
				onError(path, err)
			}
			return fmt.Errorf("parse %s: %w", path, err), false
		}
		if identify != nil && identity == nil {
			identity, err = identify(path)
			if err != nil {
				if onError != nil {
					onError(path, err)
				}
				return err, false
			}
		}
		seen[pathKey] = true
		paths = append(paths, path)
		if identity != nil {
			collected = append(collected, collectedIdentity{path: path, info: identity})
		}
		nextStack := make(map[string][]os.FileInfo, len(stack)+1)
		for key, infos := range stack {
			nextStack[key] = append([]os.FileInfo(nil), infos...)
		}
		nextStack[pathKey] = append(nextStack[pathKey], identity)
		var includeErrors []error
		for _, include := range document.includes {
			includePath, err := resolveXCConfigInclude(path, include)
			if err != nil {
				if onError != nil {
					onError(path, err)
				}
				includeErrors = append(includeErrors, err)
				continue
			}
			// Let the same authorization-aware reader perform existence and type
			// checks. In particular, never stat an include before the authorization
			// hook has accepted its lexical path. Optional missing includes are the
			// one intentional not-exist case and are ignored after that check.
			childErr, missingTarget := visit(includePath, nextStack)
			if childErr != nil {
				if include.optional && missingTarget {
					if onOptionalMissing != nil {
						onOptionalMissing(includePath)
					}
					continue
				}
				includeErrors = append(includeErrors, childErr)
			}
		}
		if len(includeErrors) == 1 {
			return includeErrors[0], false
		}
		if len(includeErrors) > 1 {
			return errors.Join(includeErrors...), false
		}
		return nil, false
	}
	if err, _ := visit(root, make(map[string][]os.FileInfo)); err != nil {
		return nil, err
	}
	return paths, nil
}

// signingPathCaseEquivalent reports whether two path spellings may be the same
// path solely because their containing directory is case-insensitive.
// Equal-folding a path is not enough: a case-sensitive directory can contain
// two hard-linked files whose path operations must remain distinct. Unknown
// filesystem metadata therefore keeps both spellings rather than coalescing.
func signingPathCaseEquivalent(left, right string) bool {
	left = normalizeSigningLexicalPath(left)
	right = normalizeSigningLexicalPath(right)
	if left == right {
		return true
	}
	if !strings.EqualFold(left, right) {
		return false
	}
	leftInsensitive, leftKnown := signingCaseInsensitiveVolumeFn(filepath.Dir(left))
	rightInsensitive, rightKnown := signingCaseInsensitiveVolumeFn(filepath.Dir(right))
	return leftKnown && rightKnown && leftInsensitive && rightInsensitive
}

func resolveXCConfigSetting(root, setting string) (xcconfigResolvedValue, error) {
	return resolveXCConfigSettingWithBase(root, setting, xcconfigResolvedValue{})
}

func resolveXCConfigSettingWithBase(root, setting string, base xcconfigResolvedValue) (xcconfigResolvedValue, error) {
	var identify func(string) (os.FileInfo, error)
	if xcconfigUsesIdentityTraversal() {
		// Identity-aware traversal coalesces case-variant aliases on
		// case-insensitive volumes, including Linux vfat/exfat/ntfs mounts.
		// os.Stat supplies the identity; case-semantics checks keep genuinely
		// distinct files separate.
		identify = os.Stat
	}
	return resolveXCConfigSettingWithBaseReaderAndIdentity(root, setting, base, os.ReadFile, os.Stat, identify)
}

func resolveXCConfigSettingWithBaseReader(
	root, setting string,
	base xcconfigResolvedValue,
	read func(string) ([]byte, error),
	stat func(string) (os.FileInfo, error),
) (xcconfigResolvedValue, error) {
	return resolveXCConfigSettingWithBaseReaderAndIdentity(root, setting, base, read, stat, nil)
}

func resolveXCConfigSettingWithBaseReaderAndIdentity(
	root, setting string,
	base xcconfigResolvedValue,
	read func(string) ([]byte, error),
	stat func(string) (os.FileInfo, error),
	identify func(string) (os.FileInfo, error),
) (xcconfigResolvedValue, error) {
	resolved, conditional, err := resolveXCConfigSettingStateWithReaderAndIdentity(
		root, setting, base, read, stat, identify, nil,
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

// xcconfigAssignmentObserver receives each matching assignment in the same
// include/event order used by the resolver, including assignments that the
// resolver later skips because a lower or earlier value wins. Security-
// sensitive callers use this to retain provenance without implementing a
// second include walker or authorization path.
type xcconfigAssignmentObserver func(path string, assignment xcconfigAssignment)

// resolveXCConfigSettingStateWithReaderAndIdentity exposes the raw traversal
// state to narrow provenance consumers. Unlike the public resolution wrapper,
// it does not convert conditional-only or divergent assignments into a
// resolution error; operational read/parse/include failures still propagate.
func resolveXCConfigSettingStateWithReaderAndIdentity(
	root, setting string,
	base xcconfigResolvedValue,
	read func(string) ([]byte, error),
	stat func(string) (os.FileInfo, error),
	identify func(string) (os.FileInfo, error),
	observe xcconfigAssignmentObserver,
) (xcconfigResolvedValue, bool, error) {
	return resolveXCConfigSettingRecursiveWithReaderAndIdentity(
		filepath.Clean(root), setting, make(map[string]bool), nil, base, read, stat, identify, observe,
	)
}

type xcconfigResolutionPath struct {
	path string
	info os.FileInfo
}

func resolveXCConfigSettingRecursiveWithReaderAndIdentity(
	path string,
	setting string,
	stack map[string]bool,
	stackPaths []xcconfigResolutionPath,
	resolved xcconfigResolvedValue,
	read func(string) ([]byte, error),
	stat func(string) (os.FileInfo, error),
	identify func(string) (os.FileInfo, error),
	observe xcconfigAssignmentObserver,
) (xcconfigResolvedValue, bool, error) {
	path = filepath.Clean(path)
	pathKey := signingLexicalPathKey(path)
	var identity os.FileInfo
	if identify != nil {
		var err error
		identity, err = identify(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return xcconfigResolvedValue{}, false, err
		}
		pathKey = normalizeSigningLexicalPath(path)
		for _, entry := range stackPaths {
			if identity != nil && entry.info != nil && signingPathCaseEquivalent(entry.path, path) && os.SameFile(identity, entry.info) {
				return resolved, false, nil
			}
		}
	}
	if stack[pathKey] {
		return resolved, false, nil
	}
	data, err := read(path)
	if err != nil {
		return xcconfigResolvedValue{}, false, err
	}
	document, err := parseXCConfig(data)
	if err != nil {
		return xcconfigResolvedValue{}, false, fmt.Errorf("parse %s: %w", path, err)
	}
	nextStack := clonePathSet(stack)
	nextStack[pathKey] = true
	nextStackPaths := append([]xcconfigResolutionPath(nil), stackPaths...)
	if identify != nil && identity != nil {
		nextStackPaths = append(nextStackPaths, xcconfigResolutionPath{path: path, info: identity})
	}

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

	for _, item := range events {
		if item.include != nil {
			includePath, err := resolveXCConfigInclude(path, *item.include)
			if err != nil {
				return xcconfigResolvedValue{}, false, err
			}
			if _, err := stat(includePath); err != nil {
				if item.include.optional && os.IsNotExist(err) {
					continue
				}
				return xcconfigResolvedValue{}, false, fmt.Errorf("read xcconfig include %s: %w", includePath, err)
			}
			included, _, err := resolveXCConfigSettingRecursiveWithReaderAndIdentity(includePath, setting, nextStack, nextStackPaths, resolved, read, stat, identify, observe)
			if err != nil {
				return xcconfigResolvedValue{}, false, err
			}
			resolved = included
			continue
		}

		assignment := item.assignment
		if assignment.baseKey != setting {
			continue
		}
		if observe != nil {
			observe(path, *assignment)
		}
		if assignment.key != setting {
			if assignment.operator == "?=" && resolved.found {
				continue
			}
			if assignment.operator == "=" {
				selector := signingXCConfigSelectorIdentity(assignment.key)
				filtered := make([]xcconfigConditionalValue, 0, len(resolved.conditionals))
				for _, existing := range resolved.conditionals {
					if signingXCConfigSelectorIdentity(existing.key) == selector {
						continue
					}
					filtered = append(filtered, existing)
				}
				resolved.conditionals = filtered
			}
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
		conditionals := append([]xcconfigConditionalValue(nil), resolved.conditionals...)
		if assignment.operator == "=" && !hasInherited {
			// A later unconditional replacement shadows earlier conditional
			// defaults, but an explicit conditional assignment remains relevant
			// in its build context and must still be reconciled below.
			conditionals = conditionals[:0]
			for _, conditional := range resolved.conditionals {
				if conditional.operator != "?=" {
					conditionals = append(conditionals, conditional)
				}
			}
		}
		resolved = xcconfigResolvedValue{
			value:            strings.TrimSpace(value),
			path:             path,
			found:            true,
			exact:            true,
			missingInherited: hasInherited && !hadLowerValue,
			conditionals:     conditionals,
		}
	}
	return resolved, len(resolved.conditionals) > 0, nil
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
		quotedValue := quoteXCConfigValue(value, assignment.quote)
		document.lines[index] = line[:assignment.operatorStart] + "=" +
			line[assignment.operatorEnd:assignment.valueStart] + quotedValue + line[assignment.valueEnd:]
		changed = true
	}
	return []byte(strings.Join(document.lines, "")), oldValues, changed, nil
}

func quoteXCConfigValue(value, quote string) string {
	if quote == "" {
		return value
	}
	var encoded strings.Builder
	encoded.Grow(len(value) + len(quote)*2)
	encoded.WriteString(quote)
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character == '\\' || character == quote[0] {
			encoded.WriteByte('\\')
		}
		encoded.WriteByte(character)
	}
	encoded.WriteString(quote)
	return encoded.String()
}

func clonePathSet(source map[string]bool) map[string]bool {
	clone := make(map[string]bool, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
