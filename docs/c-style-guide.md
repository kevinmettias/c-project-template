# C Style Guide

Types use PascalCase with no underscores:

```c
typedef struct CacheSim CacheSim;
typedef struct TraceEvent TraceEvent;
```

Functions use the same casing whether public or private. If a type name appears in the function name, keep it exactly as the type name with no added underscores. Use underscores only between other word segments:

```c
CacheSim* CacheSim_Create(CacheConfig config);
void CacheSim_Destroy(CacheSim* cache_sim);
Error CacheSim_Read(CacheSim* cache_sim, uint64_t address);
```

If the type name is intentionally lowercase, keep that lowercase type prefix:

```c
Error tcp_Connect(tcp_Connection* connection);
```

Variables use lower snake case:

```c
uint64_t line_size_bytes;
CacheSim* cache_sim;
TraceEvent trace_event;
```

Macros and enum values use upper snake case:

```c
#define CACHE_LINE_SIZE 64

typedef enum TraceOp
{
    TRACE_OP_READ = 0,
    TRACE_OP_WRITE = 1
} TraceOp;
```

Opening braces go on their own line:

```c
int main(void)
{
    if (ready)
    {
        return 0;
    }

    return 1;
}
```

Defaults:

- C11
- 4 spaces
- no tabs
- 120 column limit
- warnings enabled
- warnings treated as errors
- CMocka for unit tests
- early returns are preferred for error handling
- heap-owned objects use `Create` / `Destroy`
- caller-owned objects use `Init` / `Deinit`
- fallible APIs should prefer `Error Function(...)`
