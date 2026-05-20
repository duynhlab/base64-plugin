package main

import (
	"encoding/base64"
	"fmt"

	"github.com/invopop/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

const (
	toolEncode = "base64_encode"
	toolDecode = "base64_decode"
)

func ptrString(s string) *string { return &s }
func ptrBool(b bool) *bool       { return &b }

// ListTools advertises the two base64 tools.
func ListTools(_ ListToolsRequest) (*ListToolsResult, error) {
	inputProp := orderedmap.New[string, *jsonschema.Schema]()
	inputProp.Set("input", &jsonschema.Schema{
		Type:        "string",
		Description: "The string to encode/decode.",
	})
	inputProp.Set("url_safe", &jsonschema.Schema{
		Type:        "boolean",
		Description: "Use URL-safe base64 alphabet (RFC 4648 §5). Default: false.",
	})

	schema := jsonschema.Schema{
		Type:       "object",
		Properties: inputProp,
		Required:   []string{"input"},
	}

	readOnly := true
	idempotent := true
	annotations := &ToolAnnotations{
		ReadOnlyHint:   &readOnly,
		IdempotentHint: &idempotent,
	}

	return &ListToolsResult{
		Tools: []Tool{
			{
				Name:        toolEncode,
				Description: ptrString("Encode a UTF-8 string to base64."),
				InputSchema: schema,
				Annotations: annotations,
			},
			{
				Name:        toolDecode,
				Description: ptrString("Decode a base64-encoded string to UTF-8."),
				InputSchema: schema,
				Annotations: annotations,
			},
		},
	}, nil
}

// CallTool executes one of the base64 tools.
func CallTool(input CallToolRequest) (*CallToolResult, error) {
	raw, ok := input.Request.Arguments["input"].(string)
	if !ok {
		return errResult(`missing or non-string "input" argument`), nil
	}
	urlSafe, _ := input.Request.Arguments["url_safe"].(bool)

	switch input.Request.Name {
	case toolEncode:
		enc := pickEncoding(urlSafe)
		return textResult(enc.EncodeToString([]byte(raw))), nil
	case toolDecode:
		enc := pickEncoding(urlSafe)
		decoded, err := enc.DecodeString(raw)
		if err != nil {
			return errResult(fmt.Sprintf("base64 decode failed: %v", err)), nil
		}
		return textResult(string(decoded)), nil
	default:
		return errResult(fmt.Sprintf("unknown tool: %s", input.Request.Name)), nil
	}
}

func pickEncoding(urlSafe bool) *base64.Encoding {
	if urlSafe {
		return base64.URLEncoding
	}
	return base64.StdEncoding
}

func textResult(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentBlock{
			{Text: &TextContent{Text: text}},
		},
	}
}

func errResult(msg string) *CallToolResult {
	return &CallToolResult{
		IsError: ptrBool(true),
		Content: []ContentBlock{
			{Text: &TextContent{Text: msg}},
		},
	}
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
