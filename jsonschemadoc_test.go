package jsonschemadoc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sourcegraph/go-jsonschema/jsonschema"
)

func TestGenerate(t *testing.T) {
	tests := map[string]struct {
		schema string
		want   string
	}{
		"unset properties": {
			schema: `{ "type": "object" }`,
			want:   `{}`,
		},

		"empty properties": {
			schema: `
{
  "type": "object",
  "properties": {}
}`,
			want: `{}`,
		},

		"single": {
			schema: `
{
  "type": "object",
  "properties": {
    "a": {
      "description": "b",
      "type": "number",
      "default": 1
    }
  }
}`,
			want: `{
	// b
	"a": 1
}`,
		},

		"const as default": {
			schema: `
{
  "type": "object",
  "properties": {
    "a": {
      "type": "number",
      "const": 1
    }
  }
}`,
			want: `{
	"a": 1
}`,
		},

		"hide": {
			schema: `
{
  "type": "object",
  "properties": {
    "a": {
      "description": "b",
      "type": "number",
      "default": 1,
      "hide": true
    }
  }
}`,
			want: `{}`,
		},

		"multiple": {
			schema: `
{
  "type": "object",
  "properties": {
    "a1": {
      "description": "b1",
      "type": "string",
      "default": "c1"
    },
    "a2": {
      "description": "b2",
      "type": "string",
      "default": "c2"
    }
  }
}`,
			want: `{
	// b1
	"a1": "c1",

	// b2
	"a2": "c2"
}`,
		},

		"groups": {
			schema: `
{
  "type": "object",
  "properties": {
    "a": {"group": "G1"},
    "b": {"group": "G2"},
    "c": {"group": "G1"}
  }
}`,
			want: `{
//////////////////////////////////////////////////////////////
// G1
//////////////////////////////////////////////////////////////

	"a": null,

	"c": null,

//////////////////////////////////////////////////////////////
// G2
//////////////////////////////////////////////////////////////

	"b": null
}`,
		},

		"object": {
			schema: `
{
  "type": "object",
  "properties": {
    "a": {
      "description": "b",
      "type": "object",
      "default": {
        "c": 1,
        "d": 2
      }
    }
  }
}`,
			want: `{
	// b
	"a": {
		"c": 1,
		"d": 2
	}
}`,
		},

		"multi-line description": {
			schema: `
{
  "type": "object",
  "properties": {
    "a": {
      "description": "b1\n\nb2\nb3\n",
      "type": "string",
      "default": "c"
    }
  }
}`,
			want: `{
	// b1
	//
	// b2
	// b3
	"a": "c"
}`,
		},

		"default and examples": {
			schema: `
{
  "type": "object",
  "properties": {
    "a": {
      "description": "b1\n\nb2\nb3\n",
      "type": "string",
      "default": "c",
      "examples": ["d1", "d2", ["d3"], ["d44444444444444", "d555555555555"]]
    }
  }
}`,
			want: `{
	// b1
	//
	// b2
	// b3
	"a": "c"
	// Other example values:
	// - "d1"
	// - "d2"
	// - ["d3"]
	// - [
	//     "d44444444444444",
	//     "d555555555555"
	//   ]
}`,
		},
	}
	for label, test := range tests {
		t.Run(label, func(t *testing.T) {
			schema := parseJSONSchema(t, test.schema)
			out, err := Generate(&schema)
			if err != nil {
				t.Fatal(err)
			}
			out = strings.TrimSpace(out)
			test.want = strings.TrimSpace(test.want)
			if out != test.want {
				t.Errorf("wrong output\n\ngot:\n%s\n\nwant:\n%s", out, test.want)
			}
		})
	}
}

func parseJSONSchema(t *testing.T, input string) jsonschema.Schema {
	t.Helper()
	var schema jsonschema.Schema
	if err := json.Unmarshal([]byte(input), &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}
