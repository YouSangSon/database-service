package unit

import (
	"testing"

	"github.com/YouSangSon/database-service/internal/domain/entity"
)

func TestNewDocument(t *testing.T) {
	tests := []struct {
		name       string
		collection string
		data       map[string]interface{}
		wantErr    bool
	}{
		{
			name:       "valid document",
			collection: "users",
			data:       map[string]interface{}{"name": "John", "age": 30},
			wantErr:    false,
		},
		{
			name:       "empty collection",
			collection: "",
			data:       map[string]interface{}{"name": "John"},
			wantErr:    true,
		},
		{
			name:       "nil data",
			collection: "users",
			data:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := entity.NewDocument(tt.collection, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewDocument() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewDocument() unexpected error: %v", err)
				return
			}

			if doc.Collection() != tt.collection {
				t.Errorf("Collection() = %v, want %v", doc.Collection(), tt.collection)
			}

			if doc.Version() != 1 {
				t.Errorf("Version() = %v, want 1", doc.Version())
			}
		})
	}
}

func TestDocument_Update(t *testing.T) {
	doc, _ := entity.NewDocument("users", map[string]interface{}{"name": "John"})
	initialVersion := doc.Version()

	newData := map[string]interface{}{"name": "Jane", "email": "jane@example.com"}
	err := doc.Update(newData)

	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
	}

	if doc.Version() != initialVersion+1 {
		t.Errorf("Version after update = %v, want %v", doc.Version(), initialVersion+1)
	}

	data := doc.Data()
	if data["name"] != "Jane" {
		t.Errorf("Data name = %v, want Jane", data["name"])
	}
}

func TestDocument_Update_WithNilData(t *testing.T) {
	doc, _ := entity.NewDocument("users", map[string]interface{}{"name": "John"})

	err := doc.Update(nil)

	if err == nil {
		t.Error("Update() with nil data should return error")
	}
}
