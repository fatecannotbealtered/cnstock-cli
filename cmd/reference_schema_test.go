package cmd

import "testing"

// TestReference_EveryCommandHasRealSchemaAndExample guards against output_schema
// regressing to a stub: every command must reference a schema label that exists
// in the Schemas map with a non-empty field list, and must carry at least one
// runnable example. This keeps `reference` a usable source of truth for agents.
func TestReference_EveryCommandHasRealSchemaAndExample(t *testing.T) {
	ref := buildReference()
	if len(ref.Commands) == 0 {
		t.Fatal("reference enumerated zero commands")
	}
	for _, c := range ref.Commands {
		if c.OutputSchema == "" {
			t.Errorf("%s: empty output_schema", c.Path)
			continue
		}
		schema, ok := ref.Schemas[c.OutputSchema]
		if !ok {
			t.Errorf("%s: output_schema %q not defined in schemas map", c.Path, c.OutputSchema)
			continue
		}
		if len(schema.Fields) == 0 {
			t.Errorf("%s: schema %q has no fields (stub)", c.Path, c.OutputSchema)
		}
		if len(c.Examples) == 0 {
			t.Errorf("%s: no examples", c.Path)
		}
	}
}
