package protocol

import (
	"encoding/json"
	"testing"
)

func TestInputSchemaDefaultsMatchNormalizedInput(t *testing.T) {
	properties := InputSchema()["properties"].(map[string]any)
	input := map[string]any{"app_id": 1}
	for _, name := range []string{"review_type", "language", "page_size", "max_reviews", "max_pages"} {
		property := properties[name].(map[string]any)
		value, ok := property["default"]
		if !ok {
			t.Fatalf("%s の既定値が schema にない", name)
		}

		input[name] = value
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	_, explicit, err := ParseInput(data)
	if err != nil {
		t.Fatal(err)
	}

	_, omitted, err := ParseInput([]byte(`{"app_id":1}`))
	if err != nil {
		t.Fatal(err)
	}

	if explicit.Config() != omitted.Config() {
		t.Fatalf("schema の既定値と実際の既定値が異なる: %v / %v", explicit.Config(), omitted.Config())
	}
}
