package ir

import (
	"encoding/json"
	"math/big"
	"testing"
)

// ---------------------------------------------------------------------------
// Test: LoadIRFromBytes with a minimal valid ANF IR
// ---------------------------------------------------------------------------

func TestLoadIRFromBytes_MinimalValid(t *testing.T) {
	irJSON := `{
		"contractName": "P2PKH",
		"properties": [
			{"name": "pubKeyHash", "type": "Addr", "readonly": true}
		],
		"methods": [
			{
				"name": "constructor",
				"params": [{"name": "pubKeyHash", "type": "Addr"}],
				"body": [],
				"isPublic": false
			},
			{
				"name": "unlock",
				"params": [
					{"name": "sig", "type": "Sig"},
					{"name": "pubKey", "type": "PubKey"}
				],
				"body": [
					{"name": "t0", "value": {"kind": "load_param", "name": "pubKey"}},
					{"name": "t1", "value": {"kind": "call", "func": "hash160", "args": ["t0"]}},
					{"name": "t2", "value": {"kind": "load_prop", "name": "pubKeyHash"}},
					{"name": "t3", "value": {"kind": "bin_op", "op": "===", "left": "t1", "right": "t2"}},
					{"name": "t4", "value": {"kind": "assert", "value": "t3"}}
				],
				"isPublic": true
			}
		]
	}`

	program, err := LoadIRFromBytes([]byte(irJSON))
	if err != nil {
		t.Fatalf("LoadIRFromBytes failed: %v", err)
	}

	if program.ContractName != "P2PKH" {
		t.Errorf("expected contractName=P2PKH, got %s", program.ContractName)
	}
	if len(program.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(program.Properties))
	}
	if program.Properties[0].Name != "pubKeyHash" {
		t.Errorf("expected property name=pubKeyHash, got %s", program.Properties[0].Name)
	}
	if len(program.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(program.Methods))
	}
	if program.Methods[1].Name != "unlock" {
		t.Errorf("expected method name=unlock, got %s", program.Methods[1].Name)
	}
	if len(program.Methods[1].Body) != 5 {
		t.Errorf("expected 5 bindings in unlock body, got %d", len(program.Methods[1].Body))
	}
}

// ---------------------------------------------------------------------------
// Test: LoadIRFromBytes with load_const decodes constants correctly
// ---------------------------------------------------------------------------

func TestLoadIRFromBytes_DecodesConstants(t *testing.T) {
	irJSON := `{
		"contractName": "ConstTest",
		"properties": [],
		"methods": [
			{
				"name": "constructor",
				"params": [],
				"body": [],
				"isPublic": false
			},
			{
				"name": "check",
				"params": [{"name": "x", "type": "bigint"}],
				"body": [
					{"name": "t0", "value": {"kind": "load_const", "value": 42}},
					{"name": "t1", "value": {"kind": "load_const", "value": true}},
					{"name": "t2", "value": {"kind": "load_const", "value": "deadbeef"}},
					{"name": "t3", "value": {"kind": "load_param", "name": "x"}}
				],
				"isPublic": true
			}
		]
	}`

	program, err := LoadIRFromBytes([]byte(irJSON))
	if err != nil {
		t.Fatalf("LoadIRFromBytes failed: %v", err)
	}

	body := program.Methods[1].Body

	// t0: load_const 42
	v0 := body[0].Value
	if v0.Kind != "load_const" {
		t.Errorf("t0: expected kind=load_const, got %s", v0.Kind)
	}
	if v0.ConstBigInt == nil || v0.ConstBigInt.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("t0: expected ConstBigInt=42, got %v", v0.ConstBigInt)
	}

	// t1: load_const true
	v1 := body[1].Value
	if v1.ConstBool == nil || *v1.ConstBool != true {
		t.Errorf("t1: expected ConstBool=true, got %v", v1.ConstBool)
	}

	// t2: load_const "deadbeef"
	v2 := body[2].Value
	if v2.ConstString == nil || *v2.ConstString != "deadbeef" {
		t.Errorf("t2: expected ConstString=deadbeef, got %v", v2.ConstString)
	}
}

// ---------------------------------------------------------------------------
// Test: LoadIRFromBytes with unknown kind produces an error
// ---------------------------------------------------------------------------

func TestLoadIRFromBytes_UnknownKind_Error(t *testing.T) {
	irJSON := `{
		"contractName": "Bad",
		"properties": [],
		"methods": [
			{
				"name": "constructor",
				"params": [],
				"body": [],
				"isPublic": false
			},
			{
				"name": "check",
				"params": [],
				"body": [
					{"name": "t0", "value": {"kind": "bogus_kind"}}
				],
				"isPublic": true
			}
		]
	}`

	_, err := LoadIRFromBytes([]byte(irJSON))
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
	t.Logf("got expected error: %v", err)
}

// ---------------------------------------------------------------------------
// Test: ValidateIR rejects empty contractName
// ---------------------------------------------------------------------------

func TestValidateIR_EmptyContractName(t *testing.T) {
	program := &ANFProgram{
		ContractName: "",
		Properties:   []ANFProperty{},
		Methods:      []ANFMethod{},
	}
	err := ValidateIR(program)
	if err == nil {
		t.Fatal("expected error for empty contractName")
	}
}

// ---------------------------------------------------------------------------
// Test: ValidateIR rejects empty method name
// ---------------------------------------------------------------------------

func TestValidateIR_EmptyMethodName(t *testing.T) {
	program := &ANFProgram{
		ContractName: "Test",
		Properties:   []ANFProperty{},
		Methods: []ANFMethod{
			{Name: "", Params: nil, Body: nil, IsPublic: false},
		},
	}
	err := ValidateIR(program)
	if err == nil {
		t.Fatal("expected error for empty method name")
	}
}

// ---------------------------------------------------------------------------
// Test: ValidateIR rejects empty param name
// ---------------------------------------------------------------------------

func TestValidateIR_EmptyParamName(t *testing.T) {
	program := &ANFProgram{
		ContractName: "Test",
		Properties:   []ANFProperty{},
		Methods: []ANFMethod{
			{
				Name:     "check",
				Params:   []ANFParam{{Name: "", Type: "bigint"}},
				Body:     nil,
				IsPublic: true,
			},
		},
	}
	err := ValidateIR(program)
	if err == nil {
		t.Fatal("expected error for empty param name")
	}
}

// ---------------------------------------------------------------------------
// Test: ValidateIR rejects empty property name
// ---------------------------------------------------------------------------

func TestValidateIR_EmptyPropertyName(t *testing.T) {
	program := &ANFProgram{
		ContractName: "Test",
		Properties: []ANFProperty{
			{Name: "", Type: "bigint"},
		},
		Methods: []ANFMethod{},
	}
	err := ValidateIR(program)
	if err == nil {
		t.Fatal("expected error for empty property name")
	}
}

// ---------------------------------------------------------------------------
// Test: ValidateIR rejects loop count exceeding maximum
// ---------------------------------------------------------------------------

func TestValidateIR_LoopCountExceedsMax(t *testing.T) {
	program := &ANFProgram{
		ContractName: "Test",
		Properties:   []ANFProperty{},
		Methods: []ANFMethod{
			{
				Name:   "run",
				Params: nil,
				Body: []ANFBinding{
					{
						Name: "t0",
						Value: ANFValue{
							Kind:    "loop",
							Count:   MaxLoopCount + 1,
							IterVar: "i",
							Body:    []ANFBinding{},
						},
					},
				},
				IsPublic: true,
			},
		},
	}
	err := ValidateIR(program)
	if err == nil {
		t.Fatal("expected error for loop count exceeding maximum")
	}
}

// ---------------------------------------------------------------------------
// Test: Round-trip: construct ANFProgram, marshal to JSON, load back
// ---------------------------------------------------------------------------

func TestLoadIRFromBytes_RoundTrip(t *testing.T) {
	original := &ANFProgram{
		ContractName: "RoundTrip",
		Properties: []ANFProperty{
			{Name: "target", Type: "bigint", Readonly: true},
		},
		Methods: []ANFMethod{
			{
				Name:     "constructor",
				Params:   []ANFParam{{Name: "target", Type: "bigint"}},
				Body:     nil,
				IsPublic: false,
			},
			{
				Name:   "check",
				Params: []ANFParam{{Name: "x", Type: "bigint"}},
				Body: []ANFBinding{
					{Name: "t0", Value: ANFValue{Kind: "load_param", Name: "x"}},
					{Name: "t1", Value: ANFValue{Kind: "load_const", RawValue: json.RawMessage(`42`)}},
					{Name: "t2", Value: ANFValue{Kind: "bin_op", Op: "===", Left: "t0", Right: "t1"}},
					{Name: "t3", Value: ANFValue{Kind: "assert", RawValue: json.RawMessage(`"t2"`)}},
				},
				IsPublic: true,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	loaded, err := LoadIRFromBytes(data)
	if err != nil {
		t.Fatalf("LoadIRFromBytes failed: %v", err)
	}

	if loaded.ContractName != original.ContractName {
		t.Errorf("contractName: expected %s, got %s", original.ContractName, loaded.ContractName)
	}
	if len(loaded.Properties) != len(original.Properties) {
		t.Errorf("properties: expected %d, got %d", len(original.Properties), len(loaded.Properties))
	}
	if len(loaded.Methods) != len(original.Methods) {
		t.Errorf("methods: expected %d, got %d", len(original.Methods), len(loaded.Methods))
	}

	// Verify constant was decoded
	checkMethod := loaded.Methods[1]
	if len(checkMethod.Body) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(checkMethod.Body))
	}
	constBinding := checkMethod.Body[1]
	if constBinding.Value.ConstBigInt == nil || constBinding.Value.ConstBigInt.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("constant not decoded correctly: expected 42, got %v", constBinding.Value.ConstBigInt)
	}

	// Verify assert value ref was decoded
	assertBinding := checkMethod.Body[3]
	if assertBinding.Value.ValueRef != "t2" {
		t.Errorf("assert valueRef: expected t2, got %s", assertBinding.Value.ValueRef)
	}
}

// ---------------------------------------------------------------------------
// Test: LoadIRFromBytes rejects invalid JSON
// ---------------------------------------------------------------------------

func TestLoadIRFromBytes_InvalidJSON(t *testing.T) {
	_, err := LoadIRFromBytes([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// Test: ValidateIR rejects negative loop count
// ---------------------------------------------------------------------------

func TestValidateIR_NegativeLoopCount(t *testing.T) {
	program := &ANFProgram{
		ContractName: "Test",
		Properties:   []ANFProperty{},
		Methods: []ANFMethod{
			{
				Name:   "run",
				Params: nil,
				Body: []ANFBinding{
					{
						Name: "t0",
						Value: ANFValue{
							Kind:    "loop",
							Count:   -1,
							IterVar: "i",
							Body:    []ANFBinding{},
						},
					},
				},
				IsPublic: true,
			},
		},
	}
	err := ValidateIR(program)
	if err == nil {
		t.Fatal("expected error for negative loop count")
	}
}
