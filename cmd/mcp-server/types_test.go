package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// roundtrip marshals v → JSON → back into a fresh T and reports whether the
// fields survived intact. Used across all input/output structs without
// per-type reflection.
func roundtrip[T any](t *testing.T, v T) T {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got T
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	return got
}

// jsonschemaTag returns the jsonschema struct-tag value for field f, or "" if
// none is set. Used to audit required/constraint coverage via reflection.
func jsonschemaTag(t *testing.T, v any, fieldName string) string {
	t.Helper()

	f, ok := reflect.TypeOf(v).Elem().FieldByName(fieldName)
	if !ok {
		t.Fatalf("field %q not found on %T", fieldName, v)
	}

	return f.Tag.Get("jsonschema")
}

// containsWord reports whether word occurs as a whole token in tag (not as a
// substring of another tag, e.g. "required" inside "description=required").
func containsWord(tag, word string) bool {
	for _, tok := range strings.Split(tag, ",") {
		if strings.TrimSpace(tok) == word {
			return true
		}
	}

	return false
}

// -----------------------------------------------------------------------------
// 1. Round-trip smoke per tool pair — every input/output survives JSON I/O.
// -----------------------------------------------------------------------------

func TestTypes_StoreDocuments_Roundtrip(t *testing.T) {
	in := StoreDocumentsInput{
		CollectionID: "my-docs",
		Documents:    []string{"hello", "world"},
		IDs:          []string{"h", "w"},
		Metadatas:    []map[string]any{{"src": "a"}, {"src": "b"}},
	}
	if got := roundtrip(t, in); !reflect.DeepEqual(in, got) {
		t.Errorf("input mismatch:\nwant %+v\ngot  %+v", in, got)
	}

	out := StoreDocumentsOutput{Count: 2, IDs: []string{"h", "w"}}
	if got := roundtrip(t, out); !reflect.DeepEqual(out, got) {
		t.Errorf("output mismatch: want %+v got %+v", out, got)
	}
}

func TestTypes_QueryDocuments_Roundtrip(t *testing.T) {
	in := QueryDocumentsInput{
		CollectionID: "mem",
		QueryTexts:   []string{"first", "second"},
		NResults:     4,
	}
	if got := roundtrip(t, in); !reflect.DeepEqual(in, got) {
		t.Errorf("input: want %+v got %+v", in, got)
	}

	out := QueryDocumentsOutput{
		IDs:        [][]string{{"a", "b"}, {"c"}},
		Documents:  [][]string{{"alpha", "beta"}, {"gamma"}},
		Metadatas:  [][]map[string]any{{{}}, nil},
		Distances:  [][]float64{{0.1, 0.2}, {0.3}},
		DurationMs: 12,
	}
	if got := roundtrip(t, out); !reflect.DeepEqual(out, got) {
		t.Errorf("output: want %+v got %+v", out, got)
	}
}

func TestTypes_CollectionList_EmptyInput(t *testing.T) {
	var in CollectionListInput // zero-value
	if got := roundtrip(t, in); !reflect.DeepEqual(in, got) {
		t.Errorf("empty struct round-trip changed: %+v", got)
	}

	out := CollectionListOutput{
		Collections: []CollectionSummary{
			{Name: "a", ID: "id-a"},
			{Name: "b", ID: "id-b"},
		},
	}
	if got := roundtrip(t, out); !reflect.DeepEqual(out, got) {
		t.Errorf("mismatch: want %+v got %+v", out, got)
	}
}

func TestTypes_CollectionCreate_Roundtrip(t *testing.T) {
	in := CollectionCreateInput{Name: "team-mem"}
	if got := roundtrip(t, in); !reflect.DeepEqual(in, got) {
		t.Errorf("input: %+v", got)
	}

	out := CollectionCreateOutput{ID: "uuid-x", Name: "team-mem"}
	if got := roundtrip(t, out); !reflect.DeepEqual(out, got) {
		t.Errorf("output: %+v", got)
	}
}

func TestTypes_CollectionDelete_Roundtrip(t *testing.T) {
	in := CollectionDeleteInput{Name: "team-mem"}
	if got := roundtrip(t, in); !reflect.DeepEqual(in, got) {
		t.Errorf("input: %+v", got)
	}

	out := CollectionDeleteOutput{Deleted: true, Name: "team-mem"}
	if got := roundtrip(t, out); !reflect.DeepEqual(out, got) {
		t.Errorf("output: %+v", got)
	}
}

func TestTypes_CollectionStats_Roundtrip(t *testing.T) {
	in := CollectionStatsInput{CollectionID: "mem-x"}
	if got := roundtrip(t, in); !reflect.DeepEqual(in, got) {
		t.Errorf("input: %+v", got)
	}

	out := CollectionStatsOutput{Collection: "mem-x", Count: 7, SampleIDs: []string{"a", "b"}}
	if got := roundtrip(t, out); !reflect.DeepEqual(out, got) {
		t.Errorf("output: %+v", got)
	}
}

func TestTypes_Forget_Roundtrip(t *testing.T) {
	in := ForgetInput{CollectionID: "mem-x", IDs: []string{"a", "b"}}
	if got := roundtrip(t, in); !reflect.DeepEqual(in, got) {
		t.Errorf("input: %+v", got)
	}

	inAll := ForgetInput{CollectionID: "mem-x", All: true}
	if got := roundtrip(t, inAll); !reflect.DeepEqual(inAll, got) {
		t.Errorf("input all: %+v", got)
	}

	out := ForgetOutput{DeletedCount: 2, Mode: "ids"}
	if got := roundtrip(t, out); !reflect.DeepEqual(out, got) {
		t.Errorf("output: %+v", got)
	}
}

// -----------------------------------------------------------------------------
// 2. JSON wire-shape assertions — fields land at the snake_case keys Claude
//    Code reads. Guards against accidental CamelCase drift in field tags.
// -----------------------------------------------------------------------------

func TestTypes_WireShape_SnakeCase(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string // substring that MUST appear in marshalled JSON
	}{
		{"store input", StoreDocumentsInput{CollectionID: "c"}, `"collection_id":"c"`},
		{"store output count", StoreDocumentsOutput{Count: 1, IDs: []string{"i"}}, `"count":1`},
		{"query input", QueryDocumentsInput{CollectionID: "c", QueryTexts: []string{"q"}}, `"query_texts":["q"]`},
		{"query output duration", QueryDocumentsOutput{DurationMs: 5}, `"duration_ms":5`},
		{"create input name", CollectionCreateInput{Name: "n"}, `"name":"n"`},
		{"delete output deleted", CollectionDeleteOutput{Deleted: true, Name: "n"}, `"deleted":true`},
		{"stats input", CollectionStatsInput{CollectionID: "c"}, `"collection_id":"c"`},
		{"forget input ids", ForgetInput{CollectionID: "c", IDs: []string{"x"}}, `"ids":["x"]`},
		{"forget output count", ForgetOutput{DeletedCount: 1, Mode: "ids"}, `"deleted_count":1`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.val)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			if !strings.Contains(string(b), tc.want) {
				t.Errorf("JSON for %s missing %q in %s", tc.name, tc.want, string(b))
			}
		})
	}
}

func TestTypes_WireShape_SampleIDs(t *testing.T) {
	b, _ := json.Marshal(CollectionStatsOutput{SampleIDs: []string{"x"}})
	if !strings.Contains(string(b), `"sample_ids":["x"]`) {
		t.Errorf("sample_ids wire key missing: %s", string(b))
	}
}

func TestTypes_WireShape_DistancesAsFloat64(t *testing.T) {
	b, _ := json.Marshal(QueryDocumentsOutput{Distances: [][]float64{{0.1, 0.2}}})
	// Numbers may encode with trailing zeros, but should be float-shaped.
	if !strings.Contains(string(b), `"distances":[[0.1`) {
		t.Errorf("distances not serialized as nested float-array: %s", string(b))
	}
}

// -----------------------------------------------------------------------------
// 3. jsonschema tag audits — required fields are flagged and constraint tags
//    are present. Cheap insurance against accidental tag drift.
// -----------------------------------------------------------------------------

func TestTypes_RequiredTagAudit(t *testing.T) {
	tests := []struct {
		name      string
		val       any
		field     string
		wantReq   bool
		mustMatch string // substring that MUST appear in tag
	}{
		// store_documents
		{"store.collection_id", &StoreDocumentsInput{}, "CollectionID", true, "description="},
		{"store.documents", &StoreDocumentsInput{}, "Documents", true, "minItems=1"},
		{"store.ids optional", &StoreDocumentsInput{}, "IDs", false, "description="},
		{"store.metadatas optional", &StoreDocumentsInput{}, "Metadatas", false, ""},
		// query_documents
		{"query.collection_id", &QueryDocumentsInput{}, "CollectionID", true, ""},
		{"query.query_texts", &QueryDocumentsInput{}, "QueryTexts", true, "minItems=1"},
		{"query.n_results default", &QueryDocumentsInput{}, "NResults", false, "default=5"},
		// list — empty by design
		// create
		{"create.name", &CollectionCreateInput{}, "Name", true, "minLength=1"},
		// delete
		{"delete.name", &CollectionDeleteInput{}, "Name", true, "minLength=1"},
		// stats
		{"stats.collection_id", &CollectionStatsInput{}, "CollectionID", true, ""},
		// forget
		{"forget.collection_id", &ForgetInput{}, "CollectionID", true, ""},
		{"forget.ids optional", &ForgetInput{}, "IDs", false, "minItems=1"},
		{"forget.all optional", &ForgetInput{}, "All", false, ""},
		// outputs
		{"store.output.count", &StoreDocumentsOutput{}, "Count", true, ""},
		{"store.output.ids", &StoreDocumentsOutput{}, "IDs", true, ""},
		{"query.output.ids", &QueryDocumentsOutput{}, "IDs", true, ""},
		{"forget.output.deleted_count", &ForgetOutput{}, "DeletedCount", true, ""},
		{"forget.output.mode", &ForgetOutput{}, "Mode", true, "enum=ids|all"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := jsonschemaTag(t, tc.val, tc.field)

			isReq := containsWord(tag, "required")
			if isReq != tc.wantReq {
				t.Errorf("%s.%s required mismatch: got %v, want %v (tag=%q)",
					reflect.TypeOf(tc.val).Elem().Name(), tc.field, isReq, tc.wantReq, tag)
			}

			if tc.mustMatch != "" && !strings.Contains(tag, tc.mustMatch) {
				t.Errorf("%s.%s missing %q in tag %q",
					reflect.TypeOf(tc.val).Elem().Name(), tc.field, tc.mustMatch, tag)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 4. JSON marshal smoke — every type round-trips without panic.
// -----------------------------------------------------------------------------

func TestTypes_AllTypes_MarshalNoPanic(t *testing.T) {
	type named struct {
		name string
		val  any
	}

	items := []named{
		{"StoreDocumentsInput", StoreDocumentsInput{CollectionID: "c", Documents: []string{"d"}}},
		{"StoreDocumentsOutput", StoreDocumentsOutput{Count: 1, IDs: []string{"i"}}},
		{"QueryDocumentsInput", QueryDocumentsInput{CollectionID: "c", QueryTexts: []string{"q"}}},
		{"QueryDocumentsOutput", QueryDocumentsOutput{}},
		{"CollectionListInput", CollectionListInput{}},
		{"CollectionListOutput", CollectionListOutput{}},
		{"CollectionSummary", CollectionSummary{Name: "n", ID: "i"}},
		{"CollectionCreateInput", CollectionCreateInput{Name: "n"}},
		{"CollectionCreateOutput", CollectionCreateOutput{ID: "i", Name: "n"}},
		{"CollectionDeleteInput", CollectionDeleteInput{Name: "n"}},
		{"CollectionDeleteOutput", CollectionDeleteOutput{Deleted: true, Name: "n"}},
		{"CollectionStatsInput", CollectionStatsInput{CollectionID: "c"}},
		{"CollectionStatsOutput", CollectionStatsOutput{Collection: "c", Count: 1}},
		{"ForgetInput", ForgetInput{CollectionID: "c", All: true}},
		{"ForgetOutput", ForgetOutput{DeletedCount: 1, Mode: "all"}},
	}
	for _, it := range items {
		t.Run(it.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s marshal panicked: %v", it.name, r)
				}
			}()

			b, err := json.Marshal(it.val)
			if err != nil {
				t.Errorf("%s marshal error: %v", it.name, err)
			}

			if len(b) == 0 {
				t.Errorf("%s marshalled to empty bytes", it.name)
			}
		})
	}
}
