# Local Development

Generated projects assume MSYS2 UCRT64 on Windows.

Expected root:

```text
C:\msys64
```

Verify tools:

```powershell
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\dev-env.ps1
```

Run local CI:

```powershell
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\local-ci.ps1
```

Run formatting:

```powershell
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\format.ps1
```

Run analysis:

```powershell
C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe -ExecutionPolicy Bypass -File scripts\analyze.ps1
```

