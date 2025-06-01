package gen

import (
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
)

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toPascalCase":         toPascalCase,
		"toCamelCase":          toCamelCase,
		"toSnakeCase":          toSnakeCase,
		"toKebabCase":          toKebabCase,
		"toUpperCase":          strings.ToUpper,
		"toLowerCase":          strings.ToLower,
		"pluralize":            pluralize,
		"singularize":          singularize,
		"hasPrefix":            strings.HasPrefix,
		"hasSuffix":            strings.HasSuffix,
		"trimPrefix":           strings.TrimPrefix,
		"trimSuffix":           strings.TrimSuffix,
		"contains":             strings.Contains,
		"replace":              strings.ReplaceAll,
		"split":                strings.Split,
		"join":                 strings.Join,
		"sanitizeGoName":       sanitizeGoName,
		"isBuiltinType":        isBuiltinType,
		"needsPointer":         needsPointer,
		"buildPath":            buildPath,
		"extractPathParams":    extractPathParams,
		"hasPathParams":        hasPathParams,
		"hasQueryParams":       hasQueryParams,
		"hasHeaderParams":      hasHeaderParams,
		"filterParamsByIn":     filterParamsByIn,
		"buildMethodSignature": buildMethodSignature,
		"goDoc":                goDoc,
		"inc":                  inc,
		"dec":                  dec,
		"startsWith":           strings.HasPrefix,
	}
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// First, handle camelCase by inserting spaces before uppercase letters
	var words []string
	var currentWord strings.Builder

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			// Check if previous character is lowercase or if next character is lowercase
			prevIsLower := unicode.IsLower(rune(s[i-1]))
			nextIsLower := i+1 < len(s) && unicode.IsLower(rune(s[i+1]))

			if prevIsLower || nextIsLower {
				// Start a new word
				if currentWord.Len() > 0 {
					words = append(words, currentWord.String())
					currentWord.Reset()
				}
			}
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			currentWord.WriteRune(r)
		} else {
			// Non-alphanumeric character, finalize current word
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
	}

	// Don't forget the last word
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	// Build result
	result := ""
	for i, word := range words {
		if word == "" {
			continue
		}

		// Special handling for known acronyms
		upperWord := strings.ToUpper(word)
		if upperWord == "ID" || upperWord == "URL" || upperWord == "API" ||
			upperWord == "HTTP" || upperWord == "HTTPS" || upperWord == "JSON" ||
			upperWord == "XML" {
			result += upperWord
		} else {
			// Check if word starts with a digit
			if len(word) > 0 && unicode.IsDigit(rune(word[0])) {
				// For words starting with digits, find where letters begin
				letterStart := -1
				for j, r := range word {
					if unicode.IsLetter(r) {
						letterStart = j
						break
					}
				}

				if letterStart > 0 {
					// Split at the letter boundary
					result += word[:letterStart]
					if i == 0 && len(result) == letterStart {
						// First word after numbers should be capitalized
						result += strings.ToUpper(word[letterStart : letterStart+1])
						if letterStart+1 < len(word) {
							result += strings.ToLower(word[letterStart+1:])
						}
					} else {
						// Already have content, capitalize normally
						result += strings.ToUpper(word[letterStart : letterStart+1])
						if letterStart+1 < len(word) {
							result += strings.ToLower(word[letterStart+1:])
						}
					}
				} else {
					result += word
				}
			} else {
				// Regular word - capitalize first letter
				result += strings.ToUpper(word[:1])
				if len(word) > 1 {
					result += strings.ToLower(word[1:])
				}
			}
		}
	}

	return result
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if pascal == "" {
		return ""
	}

	// Find the first lowercase letter position
	for i, r := range pascal {
		if unicode.IsLower(r) {
			if i == 0 {
				return pascal
			}
			// Keep acronyms uppercase
			if i > 1 {
				return strings.ToLower(pascal[:i-1]) + pascal[i-1:]
			}
			return strings.ToLower(pascal[:1]) + pascal[1:]
		}
	}

	// All uppercase, convert all to lowercase
	return strings.ToLower(pascal)
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	// Handle empty string
	if s == "" {
		return ""
	}

	// Insert underscores before uppercase letters
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) && (i+1 < len(s) && unicode.IsLower(rune(s[i+1])) || unicode.IsLower(rune(s[i-1]))) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}

	// Replace non-alphanumeric with underscores
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return re.ReplaceAllString(result.String(), "_")
}

// toKebabCase converts a string to kebab-case
func toKebabCase(s string) string {
	snake := toSnakeCase(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// sanitizeGoName ensures a name is a valid Go identifier
func sanitizeGoName(s string) string {
	// Store the original to check if it started with invalid chars
	original := s

	// Replace invalid characters
	s = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(s, "_")

	// Ensure it starts with a letter or underscore
	// If the original started with a special char (not letter/digit/_), prepend V
	if s != "" && !unicode.IsLetter(rune(s[0])) && s[0] != '_' {
		s = "V" + s
	} else if original != "" && s != "" && original[0] != s[0] {
		// Original started with an invalid character that got replaced with _
		s = "V" + s
	}

	// Avoid Go keywords
	keywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}

	if keywords[strings.ToLower(s)] {
		s = s + "_"
	}

	return s
}

// isBuiltinType checks if a type is a Go builtin type
func isBuiltinType(t string) bool {
	builtins := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "byte": true, "rune": true,
		"interface{}": true, "error": true,
	}

	// Handle slices and maps
	if strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[") {
		return true
	}

	return builtins[t]
}

// needsPointer determines if a field should be a pointer
func needsPointer(field Field) bool {
	// Slices, maps, and interfaces don't need pointers for omitempty
	if strings.HasPrefix(field.Type, "[]") ||
		strings.HasPrefix(field.Type, "map[") ||
		field.Type == "interface{}" {
		return false
	}

	// Required fields don't need pointers unless nullable
	if field.Required && !field.Nullable {
		return false
	}

	// Optional fields need pointers for proper omitempty behavior
	return !field.Required || field.Nullable
}

// pluralize converts a word to its plural form (simple implementation)
func pluralize(s string) string {
	if s == "" {
		return ""
	}

	// Handle common irregular plurals
	irregular := map[string]string{
		"child":  "children",
		"person": "people",
		"man":    "men",
		"woman":  "women",
		"foot":   "feet",
		"tooth":  "teeth",
		"goose":  "geese",
		"mouse":  "mice",
	}

	if plural, ok := irregular[strings.ToLower(s)]; ok {
		if unicode.IsUpper(rune(s[0])) {
			return strings.ToUpper(plural[:1]) + plural[1:]
		}
		return plural
	}

	// Handle common patterns
	if strings.HasSuffix(s, "y") && !isVowel(s[len(s)-2]) {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh") {
		return s + "es"
	}

	return s + "s"
}

// singularize converts a word to its singular form (simple implementation)
func singularize(s string) string {
	if s == "" {
		return ""
	}

	// Handle common irregular plurals
	irregular := map[string]string{
		"children": "child",
		"people":   "person",
		"men":      "man",
		"women":    "woman",
		"feet":     "foot",
		"teeth":    "tooth",
		"geese":    "goose",
		"mice":     "mouse",
	}

	if singular, ok := irregular[strings.ToLower(s)]; ok {
		if unicode.IsUpper(rune(s[0])) {
			return strings.ToUpper(singular[:1]) + singular[1:]
		}
		return singular
	}

	// Handle common patterns
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "es") {
		if strings.HasSuffix(s[:len(s)-2], "s") || strings.HasSuffix(s[:len(s)-2], "x") ||
			strings.HasSuffix(s[:len(s)-2], "z") || strings.HasSuffix(s[:len(s)-2], "ch") ||
			strings.HasSuffix(s[:len(s)-2], "sh") {
			return s[:len(s)-2]
		}
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}

	return s
}

// isVowel checks if a byte is a vowel
func isVowel(b byte) bool {
	return b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u' ||
		b == 'A' || b == 'E' || b == 'I' || b == 'O' || b == 'U'
}

// buildPath builds a path with parameter substitution
func buildPath(path string, params []Parameter) string {
	result := path
	for _, param := range params {
		if param.In == "path" {
			placeholder := "{" + param.Name + "}"
			replacement := "%s"
			result = strings.ReplaceAll(result, placeholder, replacement)
		}
	}
	return result
}

// extractPathParams extracts path parameters from a path
func extractPathParams(path string) []string {
	var params []string
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

// hasPathParams checks if operation has path parameters
func hasPathParams(params []Parameter) bool {
	for _, p := range params {
		if p.In == "path" {
			return true
		}
	}
	return false
}

// hasQueryParams checks if operation has query parameters
func hasQueryParams(params []Parameter) bool {
	for _, p := range params {
		if p.In == "query" {
			return true
		}
	}
	return false
}

// hasHeaderParams checks if operation has header parameters
func hasHeaderParams(params []Parameter) bool {
	for _, p := range params {
		if p.In == "header" {
			return true
		}
	}
	return false
}

// filterParamsByIn filters parameters by their location
func filterParamsByIn(params []Parameter, in string) []Parameter {
	var filtered []Parameter
	for _, p := range params {
		if p.In == in {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// buildMethodSignature builds a Go method signature for an operation
func buildMethodSignature(op Operation) string {
	parts := []string{"ctx context.Context"}

	// Add path parameters
	for _, param := range op.Parameters {
		if param.In == "path" {
			// Use the parameter name as-is for method signatures
			parts = append(parts, param.Name+" "+param.Type)
		}
	}

	// Add request body
	if op.RequestBody != nil {
		parts = append(parts, "req "+op.RequestBody.Type)
	}

	// Add optional parameters struct if there are query/header params
	if hasQueryParams(op.Parameters) || hasHeaderParams(op.Parameters) {
		parts = append(parts, "params *"+op.Name+"Params")
	}

	return strings.Join(parts, ", ")
}

// goDoc formats a string as a Go doc comment
func goDoc(s string, prefix string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	lines := strings.Split(s, "\n")
	result := []string{}

	for _, line := range lines {
		if line == "" {
			result = append(result, prefix+"//")
		} else {
			result = append(result, prefix+"// "+line)
		}
	}

	return strings.Join(result, "\n")
}

// getSchemaRef extracts reference from schema if it exists
func getSchemaRef(schema *openapi3.Schema) string {
	// Check if this is a reference - in OpenAPI 3, references are handled
	// through SchemaRef objects, not directly in Schema
	// This would need to be enhanced to check the parent SchemaRef
	return ""
}

// generateOperationName generates a name for an operation without an ID
func generateOperationName(method, path string) string {
	// Remove path parameters
	cleanPath := regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(path, "")

	// Split path into parts
	parts := strings.Split(strings.Trim(cleanPath, "/"), "/")

	// Build name starting with method
	result := toPascalCase(method)

	// Add path parts
	for _, part := range parts {
		if part != "" {
			result += toPascalCase(part)
		}
	}

	return result
}

// inc increments a number (for templates)
func inc(i int) int {
	return i + 1
}

// dec decrements a number (for templates)
func dec(i int) int {
	return i - 1
}
