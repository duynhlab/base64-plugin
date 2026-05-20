package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/extism/go-sdk"
)

type req struct {
	Context map[string]any `json:"context"`
	Request struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments,omitempty"`
	} `json:"request"`
}

type resp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError *bool `json:"isError,omitempty"`
}

func main() {
	ctx := context.Background()
	mani := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: "../plugin.wasm"},
		},
	}
	plugin, err := extism.NewPlugin(ctx, mani, extism.PluginConfig{EnableWasi: true}, nil)
	must(err)
	defer plugin.Close(ctx)

	cases := []struct {
		desc, tool string
		args       map[string]any
		expectErr  bool
		expectText string
	}{
		{"encode std", "base64_encode", map[string]any{"input": "hello"}, false, "aGVsbG8="},
		{"decode std", "base64_decode", map[string]any{"input": "aGVsbG8="}, false, "hello"},
		{"encode url-safe", "base64_encode", map[string]any{"input": "??>>", "url_safe": true}, false, "Pz8-Pg=="},
		{"decode url-safe", "base64_decode", map[string]any{"input": "Pz8-Pg==", "url_safe": true}, false, "??>>"},
		{"decode invalid", "base64_decode", map[string]any{"input": "!!!not-base64!!!"}, true, ""},
		{"unknown tool", "nope", map[string]any{"input": "x"}, true, ""},
		{"missing input", "base64_encode", map[string]any{}, true, ""},
	}

	fail := 0
	for _, c := range cases {
		var r req
		r.Context = map[string]any{}
		r.Request.Name = c.tool
		r.Request.Arguments = c.args
		in, _ := json.Marshal(r)

		_, out, err := plugin.Call("call_tool", in)
		must(err)

		var rr resp
		if jerr := json.Unmarshal(out, &rr); jerr != nil {
			fmt.Printf("[FAIL] %s: bad json: %v -- raw=%s\n", c.desc, jerr, out)
			fail++
			continue
		}
		isErr := rr.IsError != nil && *rr.IsError
		text := ""
		if len(rr.Content) > 0 {
			text = rr.Content[0].Text
		}
		ok := isErr == c.expectErr && (c.expectErr || text == c.expectText)
		status := "PASS"
		if !ok {
			status = "FAIL"
			fail++
		}
		fmt.Printf("[%s] %-22s isError=%v text=%q\n", status, c.desc, isErr, text)
	}
	if fail > 0 {
		fmt.Printf("\n%d failure(s)\n", fail)
		os.Exit(1)
	}
	fmt.Println("\nall tests passed")
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
