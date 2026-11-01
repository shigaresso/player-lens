package promptbuilder

import (
	"strings"
	"testing"
)

func TestRenderInputDefinition(t *testing.T) {
	t.Run("入力定義のHTML特殊文字をタグにしない", func(t *testing.T) {
		attack := `"><script>window.untrusted=1</script>`
		page, policy, err := Render(map[string]any{"description": attack})
		if err != nil {
			t.Fatal(err)
		}

		if strings.Contains(string(page), attack) || strings.Count(string(page), "<script") != 1 || policy == "" {
			t.Fatal("入力定義をHTMLとして扱った")
		}
	})
	t.Run("表現できない定義なら部分的な画面を返さない", func(t *testing.T) {
		page, policy, err := Render(map[string]any{"invalid": func() {}})
		if err == nil || page != nil || policy != "" {
			t.Fatalf("page=%v policy=%q err=%v", page, policy, err)
		}
	})
}
