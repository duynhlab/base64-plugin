package main

import (
	"fmt"

	"github.com/duynhlab/base64-plugin/internal/base64ops"
	"github.com/invopop/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

const (
	toolEncode = "base64_encode"
	toolDecode = "base64_decode"
)

func ptr[T any](v T) *T { return &v }

// ListTools advertises the two base64 tools.
func ListTools(_ ListToolsRequest) (*ListToolsResult, error) {
	annotations := func() *ToolAnnotations {
		return &ToolAnnotations{
			ReadOnlyHint:   ptr(true),
			IdempotentHint: ptr(true),
		}
	}

	return &ListToolsResult{
		Tools: []Tool{
			{
				Name:        toolEncode,
				Description: ptr("Encode a UTF-8 string to base64."),
				InputSchema: encodeInputSchema(),
				Annotations: annotations(),
			},
			{
				Name:        toolDecode,
				Description: ptr("Decode a base64-encoded string to UTF-8. Accepts standard or URL-safe alphabets, padded or unpadded, and ignores ASCII whitespace."),
				InputSchema: decodeInputSchema(),
				Annotations: annotations(),
			},
		},
	}, nil
}

func encodeInputSchema() jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("input", &jsonschema.Schema{
		Type:        "string",
		Description: "The UTF-8 string to encode.",
	})
	props.Set("url_safe", &jsonschema.Schema{
		Type:        "boolean",
		Description: "If true, use the RFC 4648 §5 URL-safe alphabet (- and _ instead of + and /). Default: false.",
	})
	return jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"input"},
	}
}

func decodeInputSchema() jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("input", &jsonschema.Schema{
		Type:        "string",
		Description: "The base64 string to decode. Alphabet (standard or URL-safe) and padding are auto-detected.",
	})
	return jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"input"},
	}
}

// CallTool executes one of the base64 tools.
func CallTool(input CallToolRequest) (*CallToolResult, error) {
	args := input.Request.Arguments

	switch input.Request.Name {
	case toolEncode:
		raw, errResp := requireString(args, "input")
		if errResp != nil {
			return errResp, nil
		}
		urlSafe, errResp := optionalBool(args, "url_safe")
		if errResp != nil {
			return errResp, nil
		}
		return textResult(base64ops.Encode(raw, urlSafe)), nil

	case toolDecode:
		raw, errResp := requireString(args, "input")
		if errResp != nil {
			return errResp, nil
		}
		decoded, err := base64ops.Decode(raw)
		if err != nil {
			return errResultf("decode failed: %v", err), nil
		}
		return textResult(string(decoded)), nil

	default:
		return errResultf("unknown tool: %q", input.Request.Name), nil
	}
}

// requireString fetches a required string argument or returns a typed
// error result describing exactly what was wrong (missing vs wrong type).
func requireString(args map[string]any, name string) (string, *CallToolResult) {
	v, present := args[name]
	if !present {
		return "", errResultf("required argument %q is missing", name)
	}
	s, ok := v.(string)
	if !ok {
		return "", errResultf("argument %q must be a string, got %T", name, v)
	}
	return s, nil
}

// optionalBool fetches an optional boolean argument. Missing is fine
// (returns false); wrong type is a hard error rather than a silent
// coercion to false, because silently encoding with the wrong alphabet
// is a worse failure mode than rejecting the call.
func optionalBool(args map[string]any, name string) (bool, *CallToolResult) {
	v, present := args[name]
	if !present {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, errResultf("argument %q must be a boolean, got %T", name, v)
	}
	return b, nil
}

func textResult(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentBlock{{Text: &TextContent{Text: text}}},
	}
}

func errResult(msg string) *CallToolResult {
	return &CallToolResult{
		IsError: ptr(true),
		Content: []ContentBlock{{Text: &TextContent{Text: msg}}},
	}
}

func errResultf(format string, args ...any) *CallToolResult {
	return errResult(fmt.Sprintf(format, args...))
}

// ---- unused MCP handlers: return empty results so the host doesn't error ----
//
// Every list-shaped slice must be explicitly initialised to a non-nil empty
// slice. Go marshals a nil slice as JSON `null`, which strict MCP clients
// (e.g. Crush) reject with "invalid type: null, expected a sequence".

func Complete(_ CompleteRequest) (*CompleteResult, error) {
	return &CompleteResult{
		Completion: CompleteResultCompletion{Values: []string{}},
	}, nil
}

func GetPrompt(_ GetPromptRequest) (*GetPromptResult, error) {
	return &GetPromptResult{Messages: []PromptMessage{}}, nil
}

func ListPrompts(_ ListPromptsRequest) (*ListPromptsResult, error) {
	return &ListPromptsResult{Prompts: []Prompt{}}, nil
}

func ListResourceTemplates(_ ListResourceTemplatesRequest) (*ListResourceTemplatesResult, error) {
	return &ListResourceTemplatesResult{ResourceTemplates: []ResourceTemplate{}}, nil
}

func ListResources(_ ListResourcesRequest) (*ListResourcesResult, error) {
	return &ListResourcesResult{Resources: []Resource{}}, nil
}

func ReadResource(_ ReadResourceRequest) (*ReadResourceResult, error) {
	return &ReadResourceResult{Contents: []ResourceContents{}}, nil
}

func OnRootsListChanged(_ PluginNotificationContext) error {
	return nil
}

// TinyGo entry point. Real entry points are the //export functions in exports.go.
func main() {}
