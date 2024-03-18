$version = Get-Date -Format "yyyy-MM-dd HH:mm"
$BUILD_NAME = "woco"

function build-debug {
    go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$version'" -o "$BUILD_NAME`_debug_bin" main.go
}

function mac {
    $env:GOOS = "darwin"
    go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$version'" -o "$BUILD_NAME-darwin" main.go
    if (Test-Path upx) {
        upx "$BUILD_NAME-darwin"
    }
}

function win {
    $env:GOOS = "windows"
    go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$version'" -o "$BUILD_NAME.exe" main.go
    if (Test-Path upx) {
        upx "$BUILD_NAME.exe"
    }
}

function linux {
    $env:GOOS = "linux"
    go build -ldflags="-s -w" -ldflags="-X 'main.BuildTime=$version'" -o "$BUILD_NAME-linux" main.go
    if (Test-Path upx) {
        upx "$BUILD_NAME-linux"
    }
}