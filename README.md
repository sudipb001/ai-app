# AI Assistant (Go + Fyne)

A desktop AI assistant built with Go, Fyne, and the OpenAI Go SDK.

## Features

- Simple prompt/response GUI
- Uses OpenAI Chat Completions
- Embeds `.env` into the executable at build time via Go `embed`

## Prerequisites

- Go (matching your `go.mod` toolchain)
- Windows C toolchain for Fyne CGO builds (e.g., `gcc` from MinGW/MSYS2)
- Linux: GCC + OpenGL/X11 development libraries
- macOS: Xcode Command Line Tools

## Configuration

1. Copy `.env.example` to `.env`
2. Set your API key:

```env
OPENAI_API_KEY=your_openai_api_key_here
```

> Note: Because `main.go` uses `//go:embed .env`, the `.env` file must exist when you build.
> After building, the key is embedded in the executable and the app can run without a separate `.env` file.

## Run (development)

```bash
go run main.go
```

## Build executable (Windows GUI)

Use this exact command:

```bash
go build -ldflags="-s -w -H=windowsgui" -o AI_Assistant.exe main.go
```

## Build executable (Linux)

Install toolchain/deps (example for Debian/Ubuntu):

```bash
sudo apt update
sudo apt install -y build-essential libgl1-mesa-dev xorg-dev
```

Then build:

```bash
CGO_ENABLED=1 go build -ldflags="-s -w" -o ai-assistant main.go
```

## Build executable (macOS)

Install Xcode Command Line Tools:

```bash
xcode-select --install
```

Then build:

```bash
CGO_ENABLED=1 go build -ldflags="-s -w" -o ai-assistant main.go
```

## Project files

- `main.go` — Fyne UI + OpenAI integration
- `.env.example` — environment template
- `.gitignore` — ignores binaries, local env files, and editor artifacts

## Troubleshooting (Windows)

- **Alternative: install GCC with Chocolatey**
  - If Chocolatey is not installed, run PowerShell as Administrator and install it first:

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
```

- Run PowerShell as Administrator and install MinGW:

```powershell
choco install mingw -y
```

- Restart terminal, then verify:

```powershell
gcc --version
```

- If `gcc` is still not found, add MinGW `bin` (commonly `C:\ProgramData\chocolatey\lib\mingw\tools\install\mingw64\bin`) to your user `PATH`.

- **Install GCC with MSYS2 (PowerShell quick setup)**
  - Install MSYS2 from https://www.msys2.org/
  - Open **MSYS2 UCRT64** shell and run:

```bash
pacman -Syu
pacman -S --needed base-devel mingw-w64-ucrt-x86_64-toolchain
```

- Add UCRT64 `bin` to your Windows user PATH (PowerShell):

```powershell
[Environment]::SetEnvironmentVariable('Path', $env:Path + ';C:\msys64\ucrt64\bin', 'User')
```

- Restart terminal, then verify:

```powershell
gcc --version
go env CGO_ENABLED
```

- **`cgo: C compiler "gcc" not found`**
  - Install a GCC toolchain (MSYS2/MinGW-w64).
  - Ensure `gcc` is available in `PATH`.
  - Verify with:

```powershell
gcc --version
```

- **`build constraints exclude all Go files`** (from `github.com/go-gl/gl/...`)
  - This usually means CGO/native toolchain requirements are missing for Fyne/OpenGL.
  - Build with CGO enabled after GCC is installed:

```powershell
$env:CGO_ENABLED='1'; go build ./...
```

- **App says `OPENAI_API_KEY` not found**
  - Confirm `.env` existed at build time (it is embedded into the executable).
  - Rebuild after updating `.env` values.
