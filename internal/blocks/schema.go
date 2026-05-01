// Package blocks is the single source of truth for block-type metadata.
// One schema per block_type describes the data_json shape so the editor
// renders a friendly form, the agent picks the right keys, and the
// renderer reads the canonical fields. Renderers in internal/builder
// remain authoritative for output; this package only describes inputs.
package blocks

// FieldKind enumerates the form-control shapes the editor can render.
type FieldKind string

const (
	KindText     FieldKind = "text"
	KindTextarea FieldKind = "textarea"
	KindRichtext FieldKind = "richtext"
	KindURL      FieldKind = "url"
	KindImageID  FieldKind = "image_id"
	KindSelect   FieldKind = "select"
	KindBool     FieldKind = "bool"
	KindNumber   FieldKind = "number"
	KindArray    FieldKind = "array"
	KindObject   FieldKind = "object"
)

// Option is one entry in a select field.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Field describes one editable key inside data_json.
type Field struct {
	Key         string    `json:"key"`
	Label       string    `json:"label"`
	Kind        FieldKind `json:"kind"`
	Placeholder string    `json:"placeholder,omitempty"`
	Help        string    `json:"help,omitempty"`
	Required    bool      `json:"required,omitempty"`
	Options     []Option  `json:"options,omitempty"`
	ItemSchema  []Field   `json:"item_schema,omitempty"`
	Fields      []Field   `json:"fields,omitempty"`
}

// Schema describes one block_type end-to-end.
type Schema struct {
	Type        string  `json:"type"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Fields      []Field `json:"fields"`
}

var registry = map[string]Schema{}

// Register adds a schema. Last write wins.
func Register(s Schema) {
	registry[s.Type] = s
}

// Get returns the schema for a block_type, ok=false when unregistered.
func Get(t string) (Schema, bool) {
	s, ok := registry[t]
	return s, ok
}

// All returns every registered schema in deterministic order (by Type).
func All() []Schema {
	out := make([]Schema, 0, len(registry))
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		out = append(out, registry[k])
	}
	return out
}

// sortStrings is a small in-place sort to avoid pulling sort + a heavier
// import surface into a leaf package.
func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
