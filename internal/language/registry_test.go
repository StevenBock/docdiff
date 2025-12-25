package language

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	t.Run("NewRegistry creates empty registry", func(t *testing.T) {
		r := NewRegistry()
		if len(r.AllExtensions()) != 0 {
			t.Error("New registry should have no extensions")
		}
		if len(r.AllStrategies()) != 0 {
			t.Error("New registry should have no strategies")
		}
	})

	t.Run("DefaultRegistry has all built-in languages", func(t *testing.T) {
		r := DefaultRegistry()

		expectedLangs := []string{"php", "go", "java", "python", "javascript", "ruby", "vue", "shell", "powershell"}
		for _, lang := range expectedLangs {
			if _, ok := r.GetByName(lang); !ok {
				t.Errorf("DefaultRegistry missing language: %s", lang)
			}
		}
	})

	t.Run("GetByExtension returns correct strategy", func(t *testing.T) {
		r := DefaultRegistry()

		tests := []struct {
			ext      string
			wantLang string
		}{
			{".php", "php"},
			{".go", "go"},
			{".java", "java"},
			{".py", "python"},
			{".pyw", "python"},
			{".js", "javascript"},
			{".jsx", "javascript"},
			{".ts", "javascript"},
			{".tsx", "javascript"},
			{".mjs", "javascript"},
			{".cjs", "javascript"},
			{".rb", "ruby"},
			{".rake", "ruby"},
			{".vue", "vue"},
			{".sh", "shell"},
			{".bash", "shell"},
			{".ps1", "powershell"},
			{".psm1", "powershell"},
		}

		for _, tt := range tests {
			t.Run(tt.ext, func(t *testing.T) {
				s, ok := r.GetByExtension(tt.ext)
				if !ok {
					t.Errorf("GetByExtension(%s) not found", tt.ext)
					return
				}
				if s.Name() != tt.wantLang {
					t.Errorf("GetByExtension(%s) = %s, want %s", tt.ext, s.Name(), tt.wantLang)
				}
			})
		}
	})

	t.Run("GetByExtension returns false for unknown extension", func(t *testing.T) {
		r := DefaultRegistry()
		if _, ok := r.GetByExtension(".unknown"); ok {
			t.Error("GetByExtension should return false for unknown extension")
		}
	})

	t.Run("GetByName returns false for unknown language", func(t *testing.T) {
		r := DefaultRegistry()
		if _, ok := r.GetByName("unknown"); ok {
			t.Error("GetByName should return false for unknown language")
		}
	})

	t.Run("Register adds new strategy", func(t *testing.T) {
		r := NewRegistry()

		r.Register(NewGoStrategy())

		s, ok := r.GetByName("go")
		if !ok {
			t.Error("Register did not add strategy")
		}
		if s.Name() != "go" {
			t.Errorf("Registered strategy name = %s, want go", s.Name())
		}

		s2, ok := r.GetByExtension(".go")
		if !ok {
			t.Error("Register did not add extension mapping")
		}
		if s2.Name() != "go" {
			t.Errorf("Extension mapping strategy name = %s, want go", s2.Name())
		}
	})

	t.Run("AllStrategies returns all registered", func(t *testing.T) {
		r := DefaultRegistry()
		strategies := r.AllStrategies()

		if len(strategies) != 9 {
			t.Errorf("AllStrategies() returned %d strategies, want 9", len(strategies))
		}
	})

	t.Run("AllExtensions returns all registered", func(t *testing.T) {
		r := DefaultRegistry()
		exts := r.AllExtensions()

		if len(exts) < 14 {
			t.Errorf("AllExtensions() returned %d extensions, want at least 14", len(exts))
		}
	})
}
