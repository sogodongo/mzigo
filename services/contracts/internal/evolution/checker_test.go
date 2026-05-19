package evolution_test

import (
	"testing"

	"github.com/mzigo-io/mzigo/services/contracts/internal/domain"
	"github.com/mzigo-io/mzigo/services/contracts/internal/evolution"
)

var checker = evolution.NewChecker()

// Avro tests

func TestAvro_AddFieldWithDefault_IsCompatible(t *testing.T) {
	current := avroSchema(`{
		"type": "record", "name": "Payment",
		"fields": [
			{"name": "id", "type": "string"}
		]
	}`)
	proposed := avroSchema(`{
		"type": "record", "name": "Payment",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "currency", "type": "string", "default": "USD"}
		]
	}`)

	result, err := checker.Check(current, proposed, domain.SchemaFormatAvro, domain.CompatibilityBackward)
	assertNoError(t, err)
	assertClassification(t, domain.ClassificationCompatible, result.Classification)
}

func TestAvro_RemoveFieldWithNoDefault_IsBreaking(t *testing.T) {
	current := avroSchema(`{
		"type": "record", "name": "Payment",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "amount", "type": "long"}
		]
	}`)
	proposed := avroSchema(`{
		"type": "record", "name": "Payment",
		"fields": [
			{"name": "id", "type": "string"}
		]
	}`)

	result, err := checker.Check(current, proposed, domain.SchemaFormatAvro, domain.CompatibilityBackward)
	assertNoError(t, err)
	assertClassification(t, domain.ClassificationBreaking, result.Classification)
	assertChangeType(t, evolution.ChangeTypeFieldRemoved, result.Changes)
}

func TestAvro_TypeChange_IsBreaking(t *testing.T) {
	current := avroSchema(`{
		"type": "record", "name": "Payment",
		"fields": [{"name": "amount", "type": "int"}]
	}`)
	proposed := avroSchema(`{
		"type": "record", "name": "Payment",
		"fields": [{"name": "amount", "type": "string"}]
	}`)

	result, err := checker.Check(current, proposed, domain.SchemaFormatAvro, domain.CompatibilityBackward)
	assertNoError(t, err)
	assertClassification(t, domain.ClassificationBreaking, result.Classification)
	assertChangeType(t, evolution.ChangeTypeTypeChanged, result.Changes)
}

func TestAvro_NoChanges_IsSafe(t *testing.T) {
	schema := avroSchema(`{
		"type": "record", "name": "Payment",
		"fields": [{"name": "id", "type": "string"}]
	}`)

	result, err := checker.Check(schema, schema, domain.SchemaFormatAvro, domain.CompatibilityBackward)
	assertNoError(t, err)
	assertClassification(t, domain.ClassificationSafe, result.Classification)
}

// JSON Schema tests

func TestJSONSchema_AddOptionalProperty_IsCompatible(t *testing.T) {
	current := jsonSchema(`{
		"type": "object",
		"properties": {"id": {"type": "string"}},
		"required": ["id"]
	}`)
	proposed := jsonSchema(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"metadata": {"type": "object"}
		},
		"required": ["id"]
	}`)

	result, err := checker.Check(current, proposed, domain.SchemaFormatJSONSchema, domain.CompatibilityBackward)
	assertNoError(t, err)
	assertClassification(t, domain.ClassificationCompatible, result.Classification)
}

func TestJSONSchema_AddRequiredProperty_IsBreaking(t *testing.T) {
	current := jsonSchema(`{
		"type": "object",
		"properties": {"id": {"type": "string"}},
		"required": ["id"]
	}`)
	proposed := jsonSchema(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"account_id": {"type": "string"}
		},
		"required": ["id", "account_id"]
	}`)

	result, err := checker.Check(current, proposed, domain.SchemaFormatJSONSchema, domain.CompatibilityBackward)
	assertNoError(t, err)
	assertClassification(t, domain.ClassificationBreaking, result.Classification)
}

// Helpers

func avroSchema(s string) []byte  { return []byte(s) }
func jsonSchema(s string) []byte  { return []byte(s) }

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertClassification(t *testing.T, expected, got domain.ChangeClassification) {
	t.Helper()
	if expected != got {
		t.Errorf("classification: expected %s, got %s", expected, got)
	}
}

func assertChangeType(t *testing.T, expected evolution.ChangeType, changes []evolution.SchemaChange) {
	t.Helper()
	for _, c := range changes {
		if c.ChangeType == expected {
			return
		}
	}
	t.Errorf("expected change type %s not found in %+v", expected, changes)
}
