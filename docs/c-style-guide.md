# C Style Guide

Types use PascalCase with no underscores:

```c
typedef struct CacheSim CacheSim;
typedef struct TraceEvent TraceEvent;
```

Functions start with a capital letter. If a type name appears in the function name, keep it exactly as the type name with no added underscores. Use underscores only between other word segments:

```c
CacheSim *CacheSim_Create(CacheConfig config);
void CacheSim_Destroy(CacheSim *cache_sim);
bool CacheSim_Read(CacheSim *cache_sim, uint64_t address);
```

Variables use lower snake case:

```c
uint64_t line_size_bytes;
CacheSim *cache_sim;
TraceEvent trace_event;
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
- warnings enabled
- warnings treated as errors
- CMocka for unit tests

