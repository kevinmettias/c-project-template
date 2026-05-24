package main

import (
	"reflect"
	"testing"
)

func TestToSlug(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "spaces", text: "memory allocator lab", want: "memory-allocator-lab"},
		{name: "underscores", text: "cache_sim", want: "cache-sim"},
		{name: "trim punctuation", text: "  db-lab!! ", want: "db-lab"},
		{name: "empty", text: " !!! ", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ToSlug(test.text)
			if got != test.want {
				t.Fatalf("ToSlug(%q) = %q, want %q", test.text, got, test.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{text: "rudp", want: "Rudp"},
		{text: "cache-sim", want: "CacheSim"},
		{text: "job_dependency_graph", want: "JobDependencyGraph"},
	}

	for _, test := range tests {
		got := ToPascalCase(test.text)
		if got != test.want {
			t.Fatalf("ToPascalCase(%q) = %q, want %q", test.text, got, test.want)
		}
	}
}

func TestToHeaderGuard(t *testing.T) {
	got := ToHeaderGuard("db-lab_h")
	if got != "DB_LAB_H" {
		t.Fatalf("ToHeaderGuard returned %q", got)
	}
}

func TestNormalizeFlagArgs(t *testing.T) {
	got := NormalizeFlagArgs([]string{
		"netstack",
		"--kind",
		"modular",
		"--modules",
		"common,dns",
		"--out=build/smoke",
	})
	want := []string{
		"--kind",
		"modular",
		"--modules",
		"common,dns",
		"--out=build/smoke",
		"netstack",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeFlagArgs() = %#v, want %#v", got, want)
	}
}

func TestSplitModules(t *testing.T) {
	got := SplitModules(" common, dns, tcp, dns, http ")
	want := []string{"common", "dns", "http", "tcp"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitModules() = %#v, want %#v", got, want)
	}
}

func TestNewProjectBasic(t *testing.T) {
	project, err := NewProject("memory allocator lab", "basic", "")
	if err != nil {
		t.Fatalf("NewProject returned error: %v", err)
	}

	if project.Slug != "memory-allocator-lab" {
		t.Fatalf("Slug = %q", project.Slug)
	}

	if project.CMakeName != "memory_allocator_lab" {
		t.Fatalf("CMakeName = %q", project.CMakeName)
	}

	if len(project.Modules) != 1 || project.Modules[0].Name != "memory-allocator-lab" {
		t.Fatalf("Modules = %#v", project.Modules)
	}
}

func TestNewProjectModularRequiresModules(t *testing.T) {
	_, err := NewProject("netstack", "modular", "")
	if err == nil {
		t.Fatal("NewProject modular without modules succeeded")
	}
}

func TestNewProjectModular(t *testing.T) {
	project, err := NewProject("netstack", "modular", "common,dns,tcp,http")
	if err != nil {
		t.Fatalf("NewProject returned error: %v", err)
	}

	got := make([]string, 0, len(project.Modules))
	for _, module := range project.Modules {
		got = append(got, module.Name)
	}

	want := []string{"common", "dns", "http", "tcp"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Modules = %#v, want %#v", got, want)
	}
}

func TestProjectFilesBasic(t *testing.T) {
	project, err := NewProject("allocator-lab", "basic", "")
	if err != nil {
		t.Fatalf("NewProject returned error: %v", err)
	}

	files := ProjectFiles(project)
	required := []string{
		".editorconfig",
		"CMakeLists.txt",
		"CMakePresets.json",
		"LICENSE",
		"CONTRIBUTING.md",
		"include/allocator-lab.h",
		"src/allocator-lab.c",
		"src/main.c",
		"tests/test_allocator-lab.c",
		"tests/test_support/test_support.h",
		"docs/roadmap.md",
		"docs/design-notes.md",
		"benchmarks/README.md",
	}

	for _, path := range required {
		if _, ok := files[path]; !ok {
			t.Fatalf("ProjectFiles missing %q", path)
		}
	}
}

func TestProjectFilesModular(t *testing.T) {
	project, err := NewProject("netstack", "modular", "common,dns")
	if err != nil {
		t.Fatalf("NewProject returned error: %v", err)
	}

	files := ProjectFiles(project)
	required := []string{
		"common/include/common.h",
		"common/src/common.c",
		"common/tests/test_common.c",
		"dns/include/dns.h",
		"dns/src/dns.c",
		"dns/tests/test_dns.c",
		".github/workflows/ci.yml",
	}

	for _, path := range required {
		if _, ok := files[path]; !ok {
			t.Fatalf("ProjectFiles missing %q", path)
		}
	}

	if _, ok := files["src/main.c"]; ok {
		t.Fatal("modular ProjectFiles unexpectedly emitted src/main.c")
	}
}
