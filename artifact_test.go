package sdk

import (
	"testing"
)

func TestArtifactBuilder(t *testing.T) {
	t.Run("build code artifact", func(t *testing.T) {
		artifact := NewArtifact("my-code", ArtifactTypeCode).
			WithTitle("Hello World").
			WithContent("print('hello')").
			WithLanguage("python").
			WithDescription("A simple hello world").
			Exportable(true).
			Runnable(true).
			Build()

		if artifact.Name != "my-code" {
			t.Errorf("expected name 'my-code', got '%s'", artifact.Name)
		}

		if artifact.Type != ArtifactTypeCode {
			t.Errorf("expected type code, got %s", artifact.Type)
		}

		if artifact.Language != "python" {
			t.Errorf("expected language python, got '%s'", artifact.Language)
		}

		if !artifact.Exportable {
			t.Error("artifact should be exportable")
		}

		if !artifact.Runnable {
			t.Error("artifact should be runnable")
		}
	})

	t.Run("build document artifact", func(t *testing.T) {
		artifact := NewDocumentArtifact("my-doc", "# Hello\n\nThis is content")

		if artifact.Type != ArtifactTypeMarkdown {
			t.Errorf("expected type markdown, got %s", artifact.Type)
		}

		if artifact.Content != "# Hello\n\nThis is content" {
			t.Error("content mismatch")
		}
	})

	t.Run("build mermaid artifact", func(t *testing.T) {
		diagram := "graph TD\n    A --> B\n    -->"

		artifact := NewMermaidArtifact("my-diagram", diagram)

		if artifact.Type != ArtifactTypeMermaid {
			t.Errorf("expected type mermaid, got %s", artifact.Type)
		}
	})
}

func TestArtifactRegistry(t *testing.T) {
	registry := NewArtifactRegistry(nil, nil)
	registry.Clear() // Ensure clean state

	t.Run("create and get artifact", func(t *testing.T) {
		artifact := NewCodeArtifact("test-code", "go", "package main")

		err := registry.Create(artifact)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}

		retrieved, err := registry.Get(artifact.ID)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if retrieved.Name != "test-code" {
			t.Errorf("expected name 'test-code', got '%s'", retrieved.Name)
		}
	})

	t.Run("get by name", func(t *testing.T) {
		artifact := NewCodeArtifact("unique-name", "python", "print('hi')")
		_ = registry.Create(artifact)

		retrieved, err := registry.GetByName("unique-name")
		if err != nil {
			t.Fatalf("get by name failed: %v", err)
		}

		if retrieved.ID != artifact.ID {
			t.Error("retrieved wrong artifact")
		}
	})

	t.Run("update artifact", func(t *testing.T) {
		artifact := NewCodeArtifact("update-test", "go", "old content")
		_ = registry.Create(artifact)

		err := registry.Update(artifact.ID, "new content")
		if err != nil {
			t.Fatalf("update failed: %v", err)
		}

		updated, _ := registry.Get(artifact.ID)
		if updated.Content != "new content" {
			t.Error("content not updated")
		}

		if updated.Version != 2 {
			t.Errorf("expected version 2, got %d", updated.Version)
		}
	})

	t.Run("delete artifact", func(t *testing.T) {
		artifact := NewCodeArtifact("delete-test", "go", "to delete")
		_ = registry.Create(artifact)

		err := registry.Delete(artifact.ID)
		if err != nil {
			t.Fatalf("delete failed: %v", err)
		}

		_, err = registry.Get(artifact.ID)
		if err == nil {
			t.Error("artifact should not exist after deletion")
		}
	})

	t.Run("list artifacts with filter", func(t *testing.T) {
		// Create a fresh registry for this test
		filterRegistry := NewArtifactRegistry(nil, nil)

		code1 := NewCodeArtifact("code1", "go", "package main")
		if err := filterRegistry.Create(code1); err != nil {
			t.Fatalf("error creating code1: %v", err)
		}

		code2 := NewCodeArtifact("code2", "python", "print('hi')")
		if err := filterRegistry.Create(code2); err != nil {
			t.Fatalf("error creating code2: %v", err)
		}

		doc1 := NewDocumentArtifact("doc1", "# Title")
		if err := filterRegistry.Create(doc1); err != nil {
			t.Fatalf("error creating doc1: %v", err)
		}

		// Verify artifacts were created with correct types
		if code1.Type != ArtifactTypeCode {
			t.Errorf("code1 type mismatch: expected %s, got %s", ArtifactTypeCode, code1.Type)
		}
		if code2.Type != ArtifactTypeCode {
			t.Errorf("code2 type mismatch: expected %s, got %s", ArtifactTypeCode, code2.Type)
		}
		if doc1.Type != ArtifactTypeMarkdown {
			t.Errorf("doc1 type mismatch: expected %s, got %s", ArtifactTypeMarkdown, doc1.Type)
		}

		// List all
		all := filterRegistry.List(nil)
		if len(all) != 3 {
			t.Errorf("expected 3 artifacts, got %d", len(all))
			for i, a := range all {
				t.Logf("  [%d] %s (type: %s)", i, a.Name, a.Type)
			}
		}

		// List only code
		codeOnly := filterRegistry.List(&ArtifactFilter{
			Types: []ArtifactType{ArtifactTypeCode},
		})
		if len(codeOnly) != 2 {
			t.Errorf("expected 2 code artifacts, got %d", len(codeOnly))
			for i, a := range codeOnly {
				t.Logf("  [%d] %s (type: %s)", i, a.Name, a.Type)
			}
			// Show all artifacts for debugging
			t.Logf("All artifacts in registry:")
			for i, a := range all {
				t.Logf("  [%d] %s (type: %s)", i, a.Name, a.Type)
			}
		}
	})

	t.Run("callbacks", func(t *testing.T) {
		registry.Clear()

		created := false
		updated := false
		deleted := false

		registry.
			OnArtifactCreated(func(a *Artifact) { created = true }).
			OnArtifactUpdated(func(a *Artifact) { updated = true }).
			OnArtifactDeleted(func(id string) { deleted = true })

		artifact := NewCodeArtifact("callback-test", "go", "content")
		_ = registry.Create(artifact)

		if !created {
			t.Error("create callback not called")
		}

		_ = registry.Update(artifact.ID, "new content")

		if !updated {
			t.Error("update callback not called")
		}

		_ = registry.Delete(artifact.ID)

		if !deleted {
			t.Error("delete callback not called")
		}
	})
}

func TestArtifactExport(t *testing.T) {
	registry := NewArtifactRegistry(nil, nil)

	artifact := NewArtifact("export-test", ArtifactTypeCode).
		WithTitle("Test Code").
		WithContent("print('hello')").
		WithLanguage("python").
		Exportable(true).
		Build()

	_ = registry.Create(artifact)

	t.Run("export raw", func(t *testing.T) {
		data, err := registry.Export(artifact.ID, ExportFormatRaw)
		if err != nil {
			t.Fatalf("export failed: %v", err)
		}

		if string(data) != "print('hello')" {
			t.Error("raw export content mismatch")
		}
	})

	t.Run("export JSON", func(t *testing.T) {
		data, err := registry.Export(artifact.ID, ExportFormatJSON)
		if err != nil {
			t.Fatalf("export failed: %v", err)
		}

		if len(data) == 0 {
			t.Error("JSON export should not be empty")
		}
	})

	t.Run("export markdown", func(t *testing.T) {
		data, err := registry.Export(artifact.ID, ExportFormatMarkdown)
		if err != nil {
			t.Fatalf("export failed: %v", err)
		}

		content := string(data)
		if len(content) == 0 {
			t.Error("markdown export should not be empty")
		}
	})

	t.Run("export non-exportable fails", func(t *testing.T) {
		// Create directly without using helpers that set defaults
		// Use the _capabilities_set marker to prevent defaults from being applied
		nonExportable := &Artifact{
			ID:         generateArtifactID(),
			Name:       "no-export",
			Type:       ArtifactTypeCode,
			Content:    "secret",
			Exportable: false,
			Metadata:   map[string]any{"_capabilities_set": true},
		}

		_ = registry.Create(nonExportable)

		_, err := registry.Export(nonExportable.ID, ExportFormatRaw)
		if err == nil {
			t.Error("export should fail for non-exportable artifact")
		}
	})
}

func TestArtifactDefaults(t *testing.T) {
	t.Run("code artifact defaults", func(t *testing.T) {
		artifact := NewCodeArtifact("test", "python", "code")

		if !artifact.Exportable {
			t.Error("code artifacts should be exportable by default")
		}

		if !artifact.Editable {
			t.Error("code artifacts should be editable by default")
		}

		if !artifact.Runnable {
			t.Error("python code should be runnable")
		}
	})

	t.Run("document artifact defaults", func(t *testing.T) {
		artifact := NewDocumentArtifact("test", "content")

		if !artifact.Exportable {
			t.Error("documents should be exportable")
		}

		if !artifact.Previewable {
			t.Error("documents should be previewable")
		}
	})
}
