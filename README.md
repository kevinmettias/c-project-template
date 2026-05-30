# c-project-template

Reusable C project template and scaffolding tool for personal C projects.

The repo has two purposes:

- document the house style for C projects
- provide `cproj`, a small Go CLI that creates new C project skeletons

## Build `cproj`

```sh
go build -o build/cproj ./cmd/cproj
```

## Create Projects

Basic project:

```sh
cproj new allocator-lab --kind basic
```

Lab-style project:

```sh
cproj new lock-free-lab --kind lab
```

Modular project:

```sh
cproj new netstack --kind modular --modules common,serialize,compress,arp,dhcp,dns,rudp,tcp,tls,http,ws,ssh,tools
```

Database-style modular project:

```sh
cproj new db-lab --kind modular --modules common,page,wal,btree,lsm,kv,tsdb,bench,tools
```

## Defaults

Generated repos include:

- C11
- CMake
- CMake presets for debug, release, coverage, sanitizer, and Windows `debug-vcpkg` builds
- Makefile
- CMocka tests
- a CLion-friendly `test_runner` target for basic/lab projects
- clang-format
- clang-tidy
- cppcheck
- coverage script
- AddressSanitizer and UndefinedBehaviorSanitizer options
- `.editorconfig`
- `LICENSE`
- `CONTRIBUTING.md`
- roadmap and design note docs
- test support and benchmark directories
- local Windows/MSYS2 helper scripts
- GitHub Actions CI

## CLion test debugging workflow

Generated basic and lab projects include a dedicated `test_runner` target so CLion can debug tests directly instead of shelling out through `make test`.

Typical flow:

```powershell
cmake --preset debug
cmake --build --preset debug --target test_runner
ctest --preset debug
```

On Windows with vcpkg-managed `cmocka`:

```powershell
$env:VCPKG_ROOT = "C:\dev\vcpkg"
vcpkg install cmocka:x64-windows
cmake --preset debug-vcpkg
cmake --build --preset debug-vcpkg --target test_runner
ctest --preset debug-vcpkg
```

In CLion, select the matching preset under **Settings → Build, Execution, Deployment → CMake**, reload CMake, then debug the `test_runner` target and set breakpoints in `tests/test_main.c` or module test files.

