$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
$Src = Join-Path (Split-Path -Parent $Root) 'TemplateOrderServer'
$DstProto = Join-Path $Root 'api\proto'
$DstGen = Join-Path $Root 'api\gen\templateorder\v1'
New-Item -ItemType Directory -Force -Path $DstProto, $DstGen | Out-Null
Copy-Item (Join-Path $Src 'api\proto\*.proto') $DstProto -Force
Copy-Item (Join-Path $Src 'api\gen\templateorder\v1\*.go') $DstGen -Force
Get-ChildItem $DstProto, $DstGen -File | ForEach-Object {
  $c = [System.IO.File]::ReadAllText($_.FullName)
  $c = $c.Replace('template-mall/TemplateOrderServer/api/gen/templateorder/v1', 'template-mall/TemplateAdminWebServer/api/gen/templateorder/v1')
  [System.IO.File]::WriteAllText($_.FullName, $c)
}
Write-Host 'synced proto client from TemplateOrderServer'
