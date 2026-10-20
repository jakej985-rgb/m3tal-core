# M3TAL Platform Build Script (Windows)

# Try to find go
$GoPath = Get-Command go -ErrorAction SilentlyContinue
if (-not $GoPath) {
    Write-Host "[!] Go not found in PATH." -ForegroundColor Red
    $confirm = Read-Host "Would you like to attempt to install Go via winget? (y/n)"
    if ($confirm -eq 'y') {
        Write-Host "[*] Installing Go via winget..." -ForegroundColor Cyan
        winget install GoLang.Go
        Write-Host "[!] Please RESTART your terminal after installation for PATH changes to take effect." -ForegroundColor Yellow
        exit
    } else {
        Write-Host "Please install Go manually: https://go.dev/doc/install" -ForegroundColor Yellow
        exit 1
    }
}

Write-Host "[>] Building M3TAL CLI..." -ForegroundColor Cyan
go build -o m3tal.exe ./cli

Write-Host "[>] Building M3TAL API..." -ForegroundColor Cyan
go build -o m3tal-api.exe ./api/cmd

if ($LASTEXITCODE -eq 0) {
    Write-Host "[+] Build complete." -ForegroundColor Green
} else {
    Write-Host "[-] Build failed. Ensure Go is installed and in your PATH." -ForegroundColor Red
}
