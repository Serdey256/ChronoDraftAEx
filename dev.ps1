# ChronoDraftAEx 一键开发启动脚本
# 用法: .\dev.ps1
# 功能: 自动启动前端和后端，管理生命周期，Ctrl+C 退出时自动清理

param(
    [int]$FrontendPort = 5173,
    [int]$MaxWaitSeconds = 30
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$frontendDir = Join-Path $scriptDir "src\frontend"
$srcDir = Join-Path $scriptDir "src"

# 全局进程变量
$script:frontendProcess = $null

# 清理函数
function Cleanup {
    Write-Host "`n正在清理进程..." -ForegroundColor Yellow
    if ($script:frontendProcess -and !$script:frontendProcess.HasExited) {
        Stop-Process -Id $script:frontendProcess.Id -Force -ErrorAction SilentlyContinue
        Write-Host "  前端进程已终止" -ForegroundColor Gray
    }
    # 杀死所有 node 进程（Vite dev server）
    Get-Process -Name "node" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Write-Host "清理完成" -ForegroundColor Green
}

# 注册 Ctrl+C 处理
[Console]::TreatControlCAsInput = $false

try {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  ChronoDraftAEx 开发模式启动" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""

    # 1. 清理可能残留的旧进程
    Write-Host "[1/4] 清理旧进程..." -ForegroundColor Green
    Get-Process -Name "node" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 1

    # 2. 启动前端开发服务器
    Write-Host "[2/4] 启动前端开发服务器 (端口 $FrontendPort)..." -ForegroundColor Green
    $script:frontendProcess = Start-Process -FilePath "cmd.exe" `
        -ArgumentList "/c", "cd /d `"$frontendDir`" && npm run dev -- --port $FrontendPort" `
        -PassThru -WindowStyle Hidden

    # 3. 等待前端就绪
    Write-Host "[3/4] 等待前端服务器就绪..." -ForegroundColor Green
    $ready = $false
    for ($i = 0; $i -lt $MaxWaitSeconds; $i++) {
        Start-Sleep -Milliseconds 1000
        try {
            $resp = Invoke-WebRequest -Uri "http://localhost:$FrontendPort" -UseBasicParsing -TimeoutSec 2
            if ($resp.StatusCode -eq 200) {
                $ready = $true
                break
            }
        } catch {
            # 忽略连接错误，继续等待
        }
        if ($i % 5 -eq 0) {
            Write-Host "  等待中... ($i/$MaxWaitSeconds)" -ForegroundColor Gray
        }
    }

    if (-not $ready) {
        Write-Error "前端服务器启动超时！请检查 npm 是否正确安装。"
        exit 1
    }
    Write-Host "  前端服务器已就绪 ✓" -ForegroundColor Green

    # 4. 启动 Wails 后端（直接在当前终端运行，便于查看日志）
    Write-Host "[4/4] 启动 Wails 后端..." -ForegroundColor Green
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  启动完成！" -ForegroundColor Cyan
    Write-Host "  前端: http://localhost:$FrontendPort" -ForegroundColor White
    Write-Host "  MCP:  http://localhost:8787" -ForegroundColor White
    Write-Host "  按 Ctrl+C 退出" -ForegroundColor Yellow
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""

    # 设置环境变量并启动 Wails（切换到 src 目录运行）
    # FRONTEND_DEVSERVER_URL 让 main.go 知道 Vite 在哪里，从而创建反向代理
    $env:FRONTEND_DEVSERVER_URL = "http://localhost:$FrontendPort"
    $env:GOFLAGS = "-ldflags=-linkmode=internal"
    Push-Location $srcDir
    try {
        & wails3 dev --port $FrontendPort
    } finally {
        Pop-Location
    }

} catch {
    Write-Error "启动失败: $_"
    exit 1
} finally {
    Cleanup
}
