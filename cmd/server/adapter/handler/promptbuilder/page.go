// Package promptbuilder はブラウザ内の依頼文作成画面を構築する。
package promptbuilder

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
)

//go:embed index.html
var indexHTML string

//go:embed style.css
var style string

//go:embed prompt_builder.mjs
var script string

// Render は入力 schema を安全に埋め込んだ HTML と静的資産の CSP を返す。
// CSS/JavaScript は埋め込み済みの固定ソースだけを使い、入力値や外部通信を扱わない。
func Render(schema map[string]any) ([]byte, string, error) {
	definition, err := json.Marshal(schema)
	if err != nil {
		return nil, "", fmt.Errorf("encode prompt builder input definition: %w", err)
	}

	page, err := template.New("prompt").Parse(indexHTML)
	if err != nil {
		return nil, "", fmt.Errorf("parse prompt builder template: %w", err)
	}

	var output bytes.Buffer
	err = page.Execute(&output, struct {
		Schema string
		Style  template.CSS
		Script template.JS
	}{string(definition), template.CSS(style), template.JS(script)})
	if err != nil {
		return nil, "", fmt.Errorf("render prompt builder page: %w", err)
	}

	styleHash := sha256.Sum256([]byte(style))
	scriptHash := sha256.Sum256([]byte(script))
	policy := fmt.Sprintf("default-src 'none'; style-src 'sha256-%s'; script-src 'sha256-%s'; connect-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'",
		base64.StdEncoding.EncodeToString(styleHash[:]), base64.StdEncoding.EncodeToString(scriptHash[:]))
	return output.Bytes(), policy, nil
}
