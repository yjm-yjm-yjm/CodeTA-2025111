$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
$env:Path = "$env:LOCALAPPDATA\protoc\bin;C:\Users\77\go\bin;$env:Path"
$Out = "$Root\api\gen\templateorder\v1"
New-Item -ItemType Directory -Force -Path $Out | Out-Null
$protos = Get-ChildItem "$Root\api\proto\*.proto" | ForEach-Object { $_.FullName }
protoc `
  --proto_path="$Root\api\proto" `
  --go_out=$Out --go_opt=paths=source_relative `
  --go-grpc_out=$Out --go-grpc_opt=paths=source_relative `
  @protos
Write-Host "proto generated."
