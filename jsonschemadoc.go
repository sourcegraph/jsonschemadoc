package jsonschemadoc

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/sourcegraph/go-jsonschema/jsonschema"
)

// Generate generates a JSON document that describes the JSON Schema's properties.
func Generate(schema *jsonschema.Schema) (string, error) {
	var buf bytes.Buffer

	buf.WriteByte('{')
	enc := json.NewEncoder(&buf)

	groups, err := generate(schema)
	if err != nil {
		return "", err
	}

	// Sort for determinism.
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].name < groups[j].name
	})
	for _, group := range groups {
		sort.Slice(group.properties, func(i, j int) bool {
			if group.properties[i].first == group.properties[j].first {
				return group.properties[i].name < group.properties[j].name
			}
			return group.properties[i].first
		})
	}

	totalProps := 0
	for _, group := range groups {
		totalProps += len(group.properties)
	}

	pi := 0
	for j, group := range groups {
		if j == 0 {
			if group.name != "" {
				buf.WriteByte('\n')
			}
		} else {
			buf.WriteString("\n\n")
		}
		if group.name != "" {
			writeJSONComment(&buf, "", "", "////////////////////////////////////////////////////////////")
			buf.WriteByte('\n')
			writeJSONComment(&buf, "", " ", group.name)
			buf.WriteByte('\n')
			writeJSONComment(&buf, "", "", "////////////////////////////////////////////////////////////")
			buf.WriteByte('\n')
		}

		for i, prop := range group.properties {
			pi++
			if i == 0 {
				buf.WriteByte('\n')
			} else {
				buf.WriteString("\n\n")
			}
			if prop.comment != "" {
				if err := writeJSONComment(&buf, "\t", " ", prop.comment); err != nil {
					return "", err
				}
				buf.WriteByte('\n')
			}
			buf.WriteByte('\t')
			enc.SetIndent("\t", "\t")
			if err := writeJSONValue(enc, &buf, prop.name); err != nil {
				return "", err
			}
			buf.WriteString(": ")
			if err := writeJSONValue(enc, &buf, prop.value); err != nil {
				return "", err
			}
			if pi != totalProps {
				buf.WriteByte(',')
			}

			if len(prop.examples) > 0 {
				buf.WriteByte('\n')
				if err := writeJSONComment(&buf, "\t", " ", "Other example values:"); err != nil {
					return "", err
				}
				buf.WriteByte('\n')
				for i, x := range prop.examples {
					if i > 0 {
						buf.WriteByte('\n')
					}
					b, err := marshalIndentIfLong(x, "  ", "  ")
					if err != nil {
						return "", err
					}
					if err := writeJSONComment(&buf, "\t", " ", "- "+string(b)); err != nil {
						return "", err
					}
				}
			}
		}
	}

	if totalProps > 0 {
		buf.WriteByte('\n')
	}
	buf.WriteByte('}')

	return buf.String(), nil
}

func marshalIndentIfLong(v interface{}, prefix, indent string) ([]byte, error) {
	const longChars = 30
	b, err := json.Marshal(v)
	if len(b) > longChars {
		b, err = json.MarshalIndent(v, prefix, indent)
	}
	return b, err
}

func writeJSONComment(buf *bytes.Buffer, indent, space, text string) error {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(indent)
		buf.Write([]byte("//"))
		if line != "" {
			buf.WriteString(space)
		}
		buf.WriteString(line)
	}
	return nil
}

func writeJSONValue(enc *json.Encoder, buf *bytes.Buffer, v interface{}) error {
	if err := enc.Encode(v); err != nil {
		return err
	}
	buf.Truncate(buf.Len() - 1) // remove trailing newline written by enc.Encode
	return nil
}

func generate(schema *jsonschema.Schema) ([]*propertyGroup, error) {
	if schema.Properties == nil {
		return nil, nil
	}

	var groups []*propertyGroup
	byName := map[string]*propertyGroup{}
	var v jsonschema.Visitor
	v = visitorFunc(func(schema *jsonschema.Schema, rel []jsonschema.ReferenceToken) (w jsonschema.Visitor) {
		if schema == nil || schema.Properties == nil {
			return
		}
		for name, prop := range *schema.Properties {
			var extra struct {
				Hide  bool
				Group string `json:"group"`
			}
			if err := json.Unmarshal(*prop.Raw, &extra); err != nil {
				panic(err)
			}
			if extra.Hide {
				continue
			}

			p := property{
				name:     name,
				examples: prop.Examples,
			}
			if prop.Const != nil {
				p.value = prop.Const
				p.first = true // put const properties first
			} else if prop.Default != nil {
				p.value = prop.Default
			}
			if prop.Description != nil {
				p.comment = *prop.Description
			}

			groupName := extra.Group
			group := byName[groupName]
			if group == nil {
				group = &propertyGroup{name: groupName}
				byName[groupName] = group
				groups = append(groups, group)
			}
			group.properties = append(group.properties, p)
		}
		return nil
	})
	jsonschema.Walk(v, schema)

	return groups, nil
}

func isType(schema *jsonschema.Schema, typ jsonschema.PrimitiveType) bool {
	return len(schema.Type) == 1 && schema.Type[0] == typ
}

func extraField(schema *jsonschema.Schema, name string) string {
	var m map[string]interface{}
	if schema.Raw == nil {
		return ""
	}
	if err := json.Unmarshal(*schema.Raw, &m); err != nil {
		return ""
	}
	v, _ := m[name].(string)
	return v
}

type visitorFunc func(schema *jsonschema.Schema, rel []jsonschema.ReferenceToken) (w jsonschema.Visitor)

func (v visitorFunc) Visit(schema *jsonschema.Schema, rel []jsonschema.ReferenceToken) (w jsonschema.Visitor) {
	return v(schema, rel)
}

type propertyGroup struct {
	name       string
	properties []property
}

// property represents a jsonschema.Schema.Properties and its name in a single structure.
type property struct {
	name     string        // property name
	comment  string        // doc comment
	value    *interface{}  // default value (or const value)
	examples []interface{} // other example values
	first    bool          // show this property at the top
}
