package protocol

import "testing"

func TestFormSchemaHidesFallbacks(t *testing.T) {
	modules := []Module{
		newVMessModule(),
		newVLESSModule(),
		newTrojanModule(),
		newShadowsocksModule(),
		newDokodemoModule("dokodemo-door"),
		newSocksModule(),
		newHTTPModule(),
		newMixedModule(),
		newDokodemoModule("tunnel"),
	}

	for _, module := range modules {
		schema := module.FormSchema()
		for _, field := range schema.Fields {
			if field.Name == "fallbacks" || field.Type == "fallbacks" {
				t.Fatalf("%s schema must not expose fallbacks: %#v", module.Name(), field)
			}
		}
	}
}
