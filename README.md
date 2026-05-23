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
- Makefile
- CMocka tests
- clang-format
- clang-tidy
- cppcheck
- coverage script
- local Windows/MSYS2 helper scripts
- GitHub Actions CI

