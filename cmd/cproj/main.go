package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

type Project struct {
	Name       string
	Slug       string
	CMakeName  string
	HeaderName string
	Kind       string
	Modules    []Module
}

type Module struct {
	Name       string
	TypeName   string
	HeaderName string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "cproj: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	switch args[0] {
	case "new":
		return runNew(args[1:])
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() error {
	fmt.Println("usage:")
	fmt.Println("  cproj new <project-name> [--kind basic|lab|modular] [--modules a,b,c] [--out path]")
	return nil
}

func runNew(args []string) error {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	kind := flags.String("kind", "basic", "project kind: basic, lab, modular")
	modulesText := flags.String("modules", "", "comma-separated modules for modular projects")
	out := flags.String("out", ".", "parent output directory")
	force := flags.Bool("force", false, "write into an existing empty directory")

	if err := flags.Parse(NormalizeFlagArgs(args)); err != nil {
		return err
	}

	if flags.NArg() != 1 {
		return errors.New("new requires exactly one project name")
	}

	name := flags.Arg(0)
	project, err := NewProject(name, *kind, *modulesText)
	if err != nil {
		return err
	}

	root := filepath.Join(*out, project.Slug)
	if err := EnsureProjectRoot(root, *force); err != nil {
		return err
	}

	return WriteProject(root, project)
}

func NormalizeFlagArgs(args []string) []string {
	var flags []string
	var positional []string

	for index := 0; index < len(args); index++ {
		arg := args[index]
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		flags = append(flags, arg)
		if arg == "-force" || arg == "--force" {
			continue
		}

		if strings.Contains(arg, "=") {
			continue
		}

		if index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
			flags = append(flags, args[index+1])
			index++
		}
	}

	return append(flags, positional...)
}

func NewProject(name string, kind string, modulesText string) (Project, error) {
	slug := ToSlug(name)
	if slug == "" {
		return Project{}, errors.New("project name must contain letters or numbers")
	}

	if kind != "basic" && kind != "lab" && kind != "modular" {
		return Project{}, fmt.Errorf("unsupported project kind %q", kind)
	}

	modules := []Module{}
	if kind == "modular" {
		names := SplitModules(modulesText)
		if len(names) == 0 {
			return Project{}, errors.New("modular projects require --modules")
		}

		for _, moduleName := range names {
			modules = append(modules, NewModule(moduleName))
		}
	} else {
		modules = append(modules, NewModule(slug))
	}

	return Project{
		Name:       name,
		Slug:       slug,
		CMakeName:  strings.ReplaceAll(slug, "-", "_"),
		HeaderName: ToHeaderGuard(slug + "_h"),
		Kind:       kind,
		Modules:    modules,
	}, nil
}

func NewModule(name string) Module {
	slug := ToSlug(name)
	return Module{
		Name:       slug,
		TypeName:   ToPascalCase(slug),
		HeaderName: ToHeaderGuard(slug + "_h"),
	}
}

func SplitModules(text string) []string {
	seen := map[string]bool{}
	var modules []string

	for _, item := range strings.Split(text, ",") {
		module := ToSlug(strings.TrimSpace(item))
		if module == "" || seen[module] {
			continue
		}

		seen[module] = true
		modules = append(modules, module)
	}

	sort.Strings(modules)
	return modules
}

func EnsureProjectRoot(root string, force bool) error {
	entries, err := os.ReadDir(root)
	if err == nil {
		if len(entries) > 0 || !force {
			return fmt.Errorf("target directory already exists: %s", root)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(root, 0755)
}

func WriteProject(root string, project Project) error {
	files := ProjectFiles(project)
	for path, body := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(fullPath, []byte(body), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("created %s\n", root)
	return nil
}

func ProjectFiles(project Project) map[string]string {
	files := map[string]string{
		".editorconfig":                     Render(EditorConfigTemplate, project),
		".clang-format":                     Render(ClangFormatTemplate, project),
		".clang-tidy":                       Render(ClangTidyTemplate, project),
		".gitignore":                        Render(GitignoreTemplate, project),
		".pre-commit-config.yaml":           Render(PreCommitTemplate, project),
		"CMakeLists.txt":                    Render(CMakeTemplate, project),
		"CMakePresets.json":                 Render(CMakePresetsTemplate, project),
		"CONTRIBUTING.md":                   Render(ContributingTemplate, project),
		"LICENSE":                           Render(LicenseTemplate, project),
		"Makefile":                          Render(MakefileTemplate, project),
		"README.md":                         Render(ReadmeTemplate, project),
		"docs/c-style-guide.md":             Render(StyleGuideTemplate, project),
		"docs/design-notes.md":              Render(DesignNotesTemplate, project),
		"docs/local-dev.md":                 Render(LocalDevTemplate, project),
		"docs/roadmap.md":                   Render(RoadmapTemplate, project),
		"scripts/dev-env.ps1":               Render(DevEnvTemplate, project),
		"scripts/local-ci.ps1":              Render(LocalCiTemplate, project),
		"scripts/format.ps1":                Render(FormatScriptTemplate, project),
		"scripts/analyze.ps1":               Render(AnalyzeScriptTemplate, project),
		"scripts/coverage.ps1":              Render(CoverageScriptTemplate, project),
		"scripts/compile-db.ps1":            Render(CompileDbScriptTemplate, project),
		"tests/test_support/test_support.h": Render(TestSupportHeaderTemplate, project),
		".github/workflows/ci.yml":          Render(CiTemplate, project),
	}

	if project.Kind == "modular" {
		for _, module := range project.Modules {
			files[module.Name+"/include/"+module.Name+".h"] = Render(ModuleHeaderTemplate, module)
			files[module.Name+"/src/"+module.Name+".c"] = Render(ModuleSourceTemplate, module)
			files[module.Name+"/tests/test_"+module.Name+".c"] = Render(ModuleTestTemplate, module)
		}
	} else {
		module := project.Modules[0]
		files["include/"+module.Name+".h"] = Render(ModuleHeaderTemplate, module)
		files["src/"+module.Name+".c"] = Render(ModuleSourceTemplate, module)
		files["src/main.c"] = Render(MainTemplate, project)
		files["tests/test_main.c"] = Render(TestMainTemplate, module)
		files["tests/test_"+module.Name+".c"] = Render(BasicModuleTestTemplate, module)
	}

	if project.Kind == "lab" {
		files["experiments/.gitkeep"] = ""
	}

	files["benchmarks/README.md"] = Render(BenchmarksReadmeTemplate, project)
	return files
}

func Render(text string, data any) string {
	tmpl := template.Must(template.New("template").Parse(text))
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		panic(err)
	}
	return builder.String()
}

func ToSlug(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	text = re.ReplaceAllString(text, "-")
	text = strings.Trim(text, "-")
	return text
}

func ToPascalCase(text string) string {
	parts := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(text, -1)
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}

		builder.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			builder.WriteString(part[1:])
		}
	}

	return builder.String()
}

func ToHeaderGuard(text string) string {
	text = strings.ToUpper(text)
	re := regexp.MustCompile(`[^A-Z0-9]+`)
	text = re.ReplaceAllString(text, "_")
	return strings.Trim(text, "_")
}

const EditorConfigTemplate = `root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 4

[Makefile]
indent_style = tab

[*.{yml,yaml,json,md,ps1}]
indent_size = 2
`

const ClangFormatTemplate = `BasedOnStyle: LLVM
Language: Cpp
IndentWidth: 4
TabWidth: 4
UseTab: Never
ColumnLimit: 120
BreakBeforeBraces: Allman
AllowShortFunctionsOnASingleLine: None
AllowShortIfStatementsOnASingleLine: Never
AllowShortLoopsOnASingleLine: false
AllowShortBlocksOnASingleLine: Never
PointerAlignment: Left
SortIncludes: false
`

const ClangTidyTemplate = `Checks: >
  clang-analyzer-*,
  bugprone-*,
  cert-*,
  misc-*,
  performance-*,
  portability-*,
  readability-*
WarningsAsErrors: ''
HeaderFilterRegex: '.*'
FormatStyle: file
CheckOptions:
  - key: readability-identifier-naming.FunctionIgnoredRegexp
    value: '^([A-Z][A-Za-z0-9]*|[a-z][a-z0-9]*)(_[A-Z][A-Za-z0-9]*)*$'
  - key: readability-identifier-naming.VariableCase
    value: lower_case
  - key: readability-identifier-naming.ParameterCase
    value: lower_case
  - key: readability-identifier-naming.StructCase
    value: CamelCase
  - key: readability-identifier-naming.TypedefCase
    value: CamelCase
  - key: readability-identifier-naming.EnumCase
    value: CamelCase
  - key: readability-identifier-naming.EnumConstantCase
    value: UPPER_CASE
  - key: readability-identifier-naming.MacroDefinitionCase
    value: UPPER_CASE
`

const GitignoreTemplate = `build/
build-coverage/
build-sanitize/
cmake-build-*/
CMakeFiles/
CMakeCache.txt
Testing/
compile_commands.json
*.gcda
*.gcno
*.gcov
coverage.info
*.exe
*.obj
*.o
*.out
`

const LicenseTemplate = `MIT License

Copyright (c) 2026 Kevin Mettias

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`

const ContributingTemplate = `# Contributing

This is a personal systems C project. Use this checklist before committing:

- run the local CI script
- format C sources with clang-format
- keep public names aligned with docs/c-style-guide.md
- add or update tests for behavior changes
- record design decisions in docs/design-notes.md
- keep experiments and benchmarks separate from production code

Do not merge incomplete behavior behind passing placeholder tests. Prefer small, inspectable commits.
`

const PreCommitTemplate = `repos:
  - repo: local
    hooks:
      - id: clang-format
        name: clang-format
        entry: clang-format -i
        language: system
        files: \.(c|h)$

      - id: cppcheck
        name: cppcheck
        entry: cppcheck --enable=warning,style,performance,portability --std=c11 --inline-suppr --quiet .
        language: system
        files: \.(c|h)$
        pass_filenames: false
`

const CMakeTemplate = `cmake_minimum_required(VERSION 3.20)

project({{.CMakeName}} C)

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(CMAKE_C_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

option(ENABLE_COVERAGE "Build with gcov/lcov coverage instrumentation" OFF)
option(ENABLE_ASAN "Build with AddressSanitizer instrumentation" OFF)
option(ENABLE_UBSAN "Build with UndefinedBehaviorSanitizer instrumentation" OFF)

find_package(cmocka CONFIG QUIET)
if(cmocka_FOUND)
    if(TARGET cmocka::cmocka)
        set(CMOCKA_TARGET cmocka::cmocka)
    else()
        set(CMOCKA_TARGET cmocka)
    endif()
else()
    find_library(CMOCKA_TARGET cmocka REQUIRED)
endif()

add_library({{.CMakeName}}_core
{{- if eq .Kind "modular" }}
{{- range .Modules }}
    {{.Name}}/src/{{.Name}}.c
{{- end }}
{{- else }}
{{- range .Modules }}
    src/{{.Name}}.c
{{- end }}
{{- end }}
)

{{- if eq .Kind "modular" }}
target_include_directories({{.CMakeName}}_core PUBLIC
{{- range .Modules }}
    {{.Name}}/include
{{- end }}
)
{{- else }}
target_include_directories({{.CMakeName}}_core PUBLIC include)

add_executable({{.Slug}} src/main.c)
target_link_libraries({{.Slug}} PRIVATE {{.CMakeName}}_core)

add_executable(test_runner
    tests/test_main.c
{{- range .Modules }}
    tests/test_{{.Name}}.c
{{- end }}
)
target_link_libraries(test_runner PRIVATE {{.CMakeName}}_core ${CMOCKA_TARGET})
add_test(NAME test_runner COMMAND test_runner)
{{- end }}

if(MSVC)
    target_compile_options({{.CMakeName}}_core PRIVATE /W4 /WX)
{{- if ne .Kind "modular" }}
    target_compile_options({{.Slug}} PRIVATE /W4)
    target_compile_options(test_runner PRIVATE /W4)
{{- end }}
else()
    target_compile_options({{.CMakeName}}_core PRIVATE -Wall -Wextra -Wpedantic -Werror)
{{- if ne .Kind "modular" }}
    target_compile_options({{.Slug}} PRIVATE -Wall -Wextra -Wpedantic)
    target_compile_options(test_runner PRIVATE -Wall -Wextra -Wpedantic)
{{- end }}
endif()

if(ENABLE_COVERAGE)
    if(CMAKE_C_COMPILER_ID MATCHES "GNU|Clang")
        target_compile_options({{.CMakeName}}_core PRIVATE --coverage -O0 -g)
        target_link_options({{.CMakeName}}_core PUBLIC --coverage)
    else()
        message(FATAL_ERROR "ENABLE_COVERAGE requires GCC or Clang")
    endif()
endif()

if(ENABLE_ASAN)
    if(CMAKE_C_COMPILER_ID MATCHES "GNU|Clang")
        target_compile_options({{.CMakeName}}_core PUBLIC -fsanitize=address -fno-omit-frame-pointer -O1 -g)
        target_link_options({{.CMakeName}}_core PUBLIC -fsanitize=address)
    else()
        message(FATAL_ERROR "ENABLE_ASAN requires GCC or Clang")
    endif()
endif()

if(ENABLE_UBSAN)
    if(CMAKE_C_COMPILER_ID MATCHES "GNU|Clang")
        target_compile_options({{.CMakeName}}_core PUBLIC -fsanitize=undefined -fno-omit-frame-pointer -O1 -g)
        target_link_options({{.CMakeName}}_core PUBLIC -fsanitize=undefined)
    else()
        message(FATAL_ERROR "ENABLE_UBSAN requires GCC or Clang")
    endif()
endif()

enable_testing()

{{- if eq .Kind "modular" }}
{{- range .Modules }}
add_executable(test_{{.Name}} {{.Name}}/tests/test_{{.Name}}.c)
target_link_libraries(test_{{.Name}} PRIVATE {{$.CMakeName}}_core ${CMOCKA_TARGET})
add_test(NAME test_{{.Name}} COMMAND test_{{.Name}})

{{- end }}
{{- end }}
`

const CMakePresetsTemplate = `{
  "version": 6,
  "cmakeMinimumRequired": {
    "major": 3,
    "minor": 20,
    "patch": 0
  },
  "configurePresets": [
    {
      "name": "debug",
      "displayName": "Debug",
      "generator": "Ninja",
      "binaryDir": "${sourceDir}/cmake-build-debug",
      "cacheVariables": {
        "CMAKE_BUILD_TYPE": "Debug",
        "CMAKE_EXPORT_COMPILE_COMMANDS": "ON"
      }
    },
    {
      "name": "debug-vcpkg",
      "displayName": "Debug + vcpkg",
      "description": "Debug build for CLion on Windows using vcpkg-provided cmocka.",
      "inherits": "debug",
      "binaryDir": "${sourceDir}/cmake-build-debug-vcpkg",
      "condition": {
        "type": "equals",
        "lhs": "${hostSystemName}",
        "rhs": "Windows"
      },
      "cacheVariables": {
        "CMAKE_TOOLCHAIN_FILE": "$env{VCPKG_ROOT}/scripts/buildsystems/vcpkg.cmake",
        "VCPKG_TARGET_TRIPLET": "x64-windows"
      }
    },
    {
      "name": "release",
      "displayName": "Release",
      "generator": "Ninja",
      "binaryDir": "${sourceDir}/cmake-build-release",
      "cacheVariables": {
        "CMAKE_BUILD_TYPE": "Release"
      }
    },
    {
      "name": "coverage",
      "displayName": "Coverage",
      "generator": "Ninja",
      "binaryDir": "${sourceDir}/cmake-build-coverage",
      "cacheVariables": {
        "CMAKE_BUILD_TYPE": "Debug",
        "ENABLE_COVERAGE": "ON"
      }
    },
    {
      "name": "sanitize-linux",
      "displayName": "ASan + UBSan (Linux/WSL)",
      "generator": "Ninja",
      "binaryDir": "${sourceDir}/cmake-build-sanitize",
      "condition": {
        "type": "notEquals",
        "lhs": "${hostSystemName}",
        "rhs": "Windows"
      },
      "cacheVariables": {
        "CMAKE_BUILD_TYPE": "Debug",
        "ENABLE_ASAN": "ON",
        "ENABLE_UBSAN": "ON"
      }
    }
  ],
  "buildPresets": [
    {
      "name": "debug",
      "configurePreset": "debug"
    },
    {
      "name": "release",
      "configurePreset": "release"
    },
    {
      "name": "debug-vcpkg",
      "configurePreset": "debug-vcpkg"
    },
    {
      "name": "coverage",
      "configurePreset": "coverage"
    },
    {
      "name": "sanitize-linux",
      "configurePreset": "sanitize-linux"
    }
  ],
  "testPresets": [
    {
      "name": "debug",
      "configurePreset": "debug",
      "output": {
        "outputOnFailure": true
      }
    },
    {
      "name": "debug-vcpkg",
      "configurePreset": "debug-vcpkg",
      "output": {
        "outputOnFailure": true
      }
    },
    {
      "name": "sanitize-linux",
      "configurePreset": "sanitize-linux",
      "output": {
        "outputOnFailure": true
      }
    }
  ]
}
`

const MakefileTemplate = `CC := cc
CMAKE := cmake
CFLAGS := -std=c11 -Wall -Wextra -Wpedantic -Werror{{if eq .Kind "modular"}}{{range .Modules}} -I{{.Name}}/include{{end}}{{else}} -Iinclude{{end}}
LDFLAGS :=
LDLIBS := -lcmocka

BUILD_DIR := build
CORE_SRC :={{if eq .Kind "modular"}}{{range .Modules}} {{.Name}}/src/{{.Name}}.c{{end}}{{else}}{{range .Modules}} src/{{.Name}}.c{{end}}{{end}}
{{- if ne .Kind "modular" }}
APP_SRC := src/main.c
APP := $(BUILD_DIR)/{{.Slug}}
TEST_RUNNER := $(BUILD_DIR)/test_runner
{{- end }}
TESTS :={{if eq .Kind "modular"}}{{range .Modules}} $(BUILD_DIR)/test_{{.Name}}{{end}}{{else}} $(TEST_RUNNER){{end}}

.PHONY: all test clean dirs

all: dirs{{if ne .Kind "modular"}} $(APP){{end}} $(TESTS)

dirs:
	$(CMAKE) -E make_directory $(BUILD_DIR)

{{- if ne .Kind "modular" }}
$(APP): $(CORE_SRC) $(APP_SRC) | dirs
	$(CC) $(CFLAGS) $(CORE_SRC) $(APP_SRC) -o $@ $(LDFLAGS)
{{- end }}

{{- range .Modules }}
{{- if eq $.Kind "modular" }}
$(BUILD_DIR)/test_{{.Name}}: $(CORE_SRC) {{.Name}}/tests/test_{{.Name}}.c | dirs
	$(CC) $(CFLAGS) $(CORE_SRC) {{.Name}}/tests/test_{{.Name}}.c -o $@ $(LDFLAGS) $(LDLIBS)
{{- else }}
$(TEST_RUNNER): $(CORE_SRC) tests/test_main.c tests/test_{{.Name}}.c | dirs
  $(CC) $(CFLAGS) $(CORE_SRC) tests/test_main.c tests/test_{{.Name}}.c -o $@ $(LDFLAGS) $(LDLIBS)
{{- end }}

{{- end }}
test: $(TESTS)
{{- if eq .Kind "modular" }}
{{- range .Modules }}
  $(BUILD_DIR)/test_{{.Name}}
{{- end }}
{{- else }}
  $(TEST_RUNNER)
{{- end }}

clean:
	rm -rf $(BUILD_DIR)
`

const ReadmeTemplate = `# {{.Slug}}

C project generated by ` + "`cproj`" + `.

## Build

` + "```sh" + `
cmake --preset debug
cmake --build --preset debug
` + "```" + `

## Test

` + "```sh" + `
ctest --preset debug --output-on-failure
` + "```" + `

Fallback Makefile workflow:

` + "```sh" + `
make
make test
` + "```" + `

## Debug Tests In CLion

Basic and lab projects generate a dedicated ` + "`test_runner`" + ` target so CLion can debug tests directly. Open the project as a CMake project, pick the ` + "`debug`" + ` preset, build ` + "`test_runner`" + `, and debug that target with breakpoints in ` + "`tests/test_main.c`" + `.

## Windows + vcpkg

On Windows, install ` + "`cmocka`" + ` through vcpkg and use the ` + "`debug-vcpkg`" + ` preset:

` + "```powershell" + `
$env:VCPKG_ROOT = "C:\dev\vcpkg"
vcpkg install cmocka:x64-windows
cmake --preset debug-vcpkg
cmake --build --preset debug-vcpkg --target test_runner
ctest --preset debug-vcpkg
` + "```" + `

## Local CI

` + "```powershell" + `
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\local-ci.ps1
` + "```" + `

## Design Notes

- docs/roadmap.md tracks milestones and scope.
- docs/design-notes.md records decisions, invariants, and open questions.
- benchmarks/ stores benchmark plans, inputs, and reports.
`

const StyleGuideTemplate = `# C Style Guide

Types use PascalCase:

` + "```c" + `
typedef struct CacheSim CacheSim;
typedef enum TraceOp TraceOp;
` + "```" + `

Functions use the same casing whether public or private. If a type name appears in the function name, keep it exactly as the type name with no added underscores. Use underscores only between other word segments:

` + "```c" + `
CacheSim* CacheSim_Create(CacheConfig config);
void CacheSim_Destroy(CacheSim* cache_sim);
` + "```" + `

If the type name is intentionally lowercase, keep that lowercase type prefix:

` + "```c" + `
Error tcp_Connect(tcp_Connection* connection);
` + "```" + `

Variables use lower snake case.

Pointers bind to the type:

` + "```c" + `
CacheSim* cache_sim;
` + "```" + `

Macros and enum values use upper snake case:

` + "```c" + `
#define CACHE_LINE_SIZE 64

typedef enum TraceOp
{
    TRACE_OP_READ = 0,
    TRACE_OP_WRITE = 1
} TraceOp;
` + "```" + `

Opening braces go on their own line:

` + "```c" + `
int main(void)
{
    return 0;
}
` + "```" + `

Use early returns for error handling. Heap-owned objects use ` + "`Create`" + ` / ` + "`Destroy`" + `, caller-owned objects use ` + "`Init`" + ` / ` + "`Deinit`" + `, and fallible APIs should prefer:

` + "```c" + `
Error Function(...);
` + "```" + `
`

const RoadmapTemplate = `# Roadmap

## Milestones

1. Define the public surface and project invariants.
2. Add focused tests for the smallest useful behavior.
3. Implement the first narrow feature.
4. Add benchmarks only after correctness tests exist.
5. Record design decisions as the implementation changes.

## In Scope

- core implementation
- tests
- benchmarks
- design notes

## Out Of Scope For Now

- broad rewrites before the first working path
- performance tuning before correctness
- large abstractions without repeated concrete use
`

const DesignNotesTemplate = `# Design Notes

Use this file to capture decisions while they are still fresh.

## Invariants

- TBD

## Data Ownership

- TBD

## Error Handling

- TBD

## Open Questions

- TBD
`

const LocalDevTemplate = `# Local Development

Use MSYS2 UCRT64 on Windows.

` + "```powershell" + `
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\dev-env.ps1
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\local-ci.ps1
` + "```" + `

Useful CMake presets:

` + "```powershell" + `
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\dev-env.ps1 cmake --preset debug
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\dev-env.ps1 cmake --build --preset debug
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\dev-env.ps1 ctest --preset debug
` + "```" + `

CLion + vcpkg on Windows:

` + "```powershell" + `
$env:VCPKG_ROOT = "C:\dev\vcpkg"
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\dev-env.ps1 cmake --preset debug-vcpkg
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\dev-env.ps1 cmake --build --preset debug-vcpkg --target test_runner
` + "```" + `

In CLion, open **Settings → Build, Execution, Deployment → CMake**, choose the ` + "`debug-vcpkg`" + ` preset, reload CMake, and debug ` + "`test_runner`" + ` directly so breakpoints work in ` + "`tests/test_main.c`" + `.

The ` + "`sanitize-linux`" + ` preset is intended for Linux, WSL, and CI. Native Windows/MSYS2 GCC builds may not provide ASan/UBSan runtime libraries.
`

const DevEnvTemplate = `param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]] $Command
)

$ErrorActionPreference = "Stop"

$msys_root = "C:\msys64"
$ucrt_bin = Join-Path $msys_root "ucrt64\bin"
$usr_bin = Join-Path $msys_root "usr\bin"
$pipx_bin = Join-Path $HOME ".local\bin"
$env:GOROOT = Join-Path $msys_root "ucrt64\lib\go"

$env:Path = "$pipx_bin;$ucrt_bin;$usr_bin;$env:Path"

if ($Command.Count -eq 0) {
    Write-Host "MSYS2 UCRT64 development environment"
    gcc --version
    cmake --version
    make --version
    pkg-config --modversion cmocka
    clang-format --version
    clang-tidy --version
    cppcheck --version
    gdb --version
    pre-commit --version
    exit $LASTEXITCODE
}

$program = $Command[0]
$arguments = @()
if ($Command.Count -gt 1) {
    $arguments = $Command[1..($Command.Count - 1)]
}

& $program @arguments
exit $LASTEXITCODE
`

const LocalCiTemplate = `$ErrorActionPreference = "Stop"

$repo_root = Split-Path -Parent $PSScriptRoot
$dev_env = Join-Path $PSScriptRoot "dev-env.ps1"

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)]
        [string[]] $Command
    )

    & $dev_env @Command
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $($Command -join ' ')"
    }
}

Push-Location $repo_root
try {
    Invoke-Checked -Command @("cmake", "-S", ".", "-B", "build", "-G", "Ninja")
    Invoke-Checked -Command @("cmake", "--build", "build", "--parallel")
    Invoke-Checked -Command @("ctest", "--test-dir", "build", "--output-on-failure")
    Invoke-Checked -Command @("make", "clean")
    Invoke-Checked -Command @("make", "test")
} finally {
    Pop-Location
}
`

const FormatScriptTemplate = `$ErrorActionPreference = "Stop"

$repo_root = Split-Path -Parent $PSScriptRoot
$dev_env = Join-Path $PSScriptRoot "dev-env.ps1"

Push-Location $repo_root
try {
    $files = Get-ChildItem -Path . -Recurse -File -Include *.c, *.h |
        Where-Object { $_.FullName -notmatch "\\build\\" } |
        ForEach-Object { $_.FullName }

    if ($files.Count -eq 0) {
        exit 0
    }

    & $dev_env clang-format -i @files
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
`

const AnalyzeScriptTemplate = `$ErrorActionPreference = "Stop"

$repo_root = Split-Path -Parent $PSScriptRoot
$dev_env = Join-Path $PSScriptRoot "dev-env.ps1"

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)]
        [string[]] $Command
    )

    & $dev_env @Command
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $($Command -join ' ')"
    }
}

Push-Location $repo_root
try {
    Invoke-Checked -Command @("cmake", "-S", ".", "-B", "build", "-G", "Ninja", "-DCMAKE_EXPORT_COMPILE_COMMANDS=ON")
    $source_files = Get-ChildItem -Path . -Recurse -File -Include *.c |
        Where-Object { $_.FullName -notmatch "\\build\\" } |
        ForEach-Object { $_.FullName }

    foreach ($source_file in $source_files) {
        Invoke-Checked -Command @("clang-tidy", $source_file, "-p", "build")
    }

    Invoke-Checked -Command @("cppcheck", "--enable=warning,style,performance,portability", "--std=c11", "--inline-suppr", "--quiet", "--project=build/compile_commands.json")
} finally {
    Pop-Location
}
`

const CoverageScriptTemplate = `$ErrorActionPreference = "Stop"

$repo_root = Split-Path -Parent $PSScriptRoot
$dev_env = Join-Path $PSScriptRoot "dev-env.ps1"
$coverage_dir = "build-coverage"

Push-Location $repo_root
try {
    & $dev_env cmake -S . -B $coverage_dir -G Ninja -DENABLE_COVERAGE=ON
    & $dev_env cmake --build $coverage_dir --parallel
    & $dev_env ctest --test-dir $coverage_dir --output-on-failure
    & $dev_env lcov --capture --directory $coverage_dir --output-file "$coverage_dir/coverage.info"
    & $dev_env lcov --summary "$coverage_dir/coverage.info"
} finally {
    Pop-Location
}
`

const CompileDbScriptTemplate = `$ErrorActionPreference = "Stop"

$repo_root = Split-Path -Parent $PSScriptRoot
$dev_env = Join-Path $PSScriptRoot "dev-env.ps1"

Push-Location $repo_root
try {
    & $dev_env cmake -S . -B build -G Ninja -DCMAKE_EXPORT_COMPILE_COMMANDS=ON
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
`

const CiTemplate = `name: CI

on:
  push:
    branches:
      - main
  pull_request:
    branches:
      - main

jobs:
  build-and-test:
    name: Build and test
    runs-on: ubuntu-latest

    steps:
      - name: Check out repository
        uses: actions/checkout@v4

      - name: Install C dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y \
            bear \
            build-essential \
            clang-tidy \
            cmake \
            cppcheck \
            lcov \
            libcmocka-dev \
            pre-commit \
            valgrind

      - name: Configure CMake
        run: cmake -S . -B build

      - name: Build CMake project
        run: cmake --build build --parallel

      - name: Run CMake tests
        run: ctest --test-dir build --output-on-failure

      - name: Clean Make build directory
        run: make clean

      - name: Build with Make
        run: make

      - name: Run Make tests
        run: make test

      - name: Configure sanitizer build
        run: cmake -S . -B build-sanitize -DENABLE_ASAN=ON -DENABLE_UBSAN=ON

      - name: Build sanitizer target
        run: cmake --build build-sanitize --parallel

      - name: Run sanitizer tests
        run: ctest --test-dir build-sanitize --output-on-failure
`

const ModuleHeaderTemplate = `#ifndef {{.HeaderName}}
#define {{.HeaderName}}

typedef struct {{.TypeName}}
{
    int placeholder;
} {{.TypeName}};

#endif
`

const ModuleSourceTemplate = `#include "{{.Name}}.h"
`

const ModuleTestTemplate = `#include <stdarg.h>
#include <stddef.h>
#include <setjmp.h>
#include <cmocka.h>

static void {{.TypeName}}_Placeholder_Test(void **state)
{
    (void)state;
}

int main(void)
{
    const struct CMUnitTest tests[] = {
        cmocka_unit_test({{.TypeName}}_Placeholder_Test),
    };

    return cmocka_run_group_tests(tests, NULL, NULL);
}
`

const BasicModuleTestTemplate = `#include <stdarg.h>
#include <stddef.h>
#include <setjmp.h>
#include <cmocka.h>

void {{.TypeName}}_Placeholder_Test(void **state)
{
    (void)state;
}
`

const TestMainTemplate = `#include <stdarg.h>
#include <stddef.h>
#include <setjmp.h>
#include <cmocka.h>

void {{.TypeName}}_Placeholder_Test(void **state);

int main(void)
{
    const struct CMUnitTest tests[] = {
        cmocka_unit_test({{.TypeName}}_Placeholder_Test),
    };

    return cmocka_run_group_tests(tests, NULL, NULL);
}
`

const TestSupportHeaderTemplate = `#ifndef TEST_SUPPORT_H
#define TEST_SUPPORT_H

#include <stdint.h>

static inline uint64_t TestSupport_Kib(uint64_t value)
{
    return value * 1024;
}

static inline uint64_t TestSupport_Mib(uint64_t value)
{
    return TestSupport_Kib(value) * 1024;
}

#endif
`

const BenchmarksReadmeTemplate = `# Benchmarks

Keep benchmark inputs, commands, and results here.

Suggested layout:

- inputs/
- results/
- reports/

Start with correctness tests. Add benchmarks once the first useful behavior exists.
`

const MainTemplate = `#include <stdio.h>

int main(void)
{
    printf("{{.Slug}}\n");
    return 0;
}
`
