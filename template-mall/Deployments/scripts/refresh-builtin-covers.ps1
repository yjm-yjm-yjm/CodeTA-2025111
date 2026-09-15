# 用 covers/ 中的真实首页预览图，更新已上架内置模板的 cover_object_key。
$ErrorActionPreference = 'Stop'
$Admin = 'http://127.0.0.1:8081'
$SeedRoot = Join-Path (Split-Path -Parent $PSScriptRoot) 'seed\builtin'
$ManifestPath = Join-Path $SeedRoot 'manifest.json'

function Invoke-Json($Method, $Url, $Body = $null, $Token = $null) {
  $headers = @{}
  if ($Token) { $headers['Authorization'] = "Bearer $Token" }
  if ($null -ne $Body) {
    $json = $Body | ConvertTo-Json -Compress -Depth 8
    return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -ContentType 'application/json; charset=utf-8' -Body ([Text.Encoding]::UTF8.GetBytes($json))
  }
  return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers
}

function Upload-Cover($Token, $FilePath) {
  $fileName = [IO.Path]::GetFileName($FilePath)
  $cred = (Invoke-Json POST "$Admin/api/upload/credential" @{
      filename     = $fileName
      content_type = 'image/png'
    } $Token).data
  $bytes = [IO.File]::ReadAllBytes($FilePath)
  Invoke-RestMethod -Method Put -Uri $cred.upload_url -ContentType 'image/png' -Body $bytes | Out-Null
  return $cred.object_key
}

$items = Get-Content -LiteralPath $ManifestPath -Encoding UTF8 -Raw | ConvertFrom-Json
Write-Host '== mock login =='
$adminToken = (Invoke-Json POST "$Admin/api/auth/mock-login" @{ admin_id = 'seed-admin'; nickname = 'SeedAdmin' }).data.token

Write-Host '== list templates =='
$tpls = (Invoke-Json GET "$Admin/api/templates?page=1&page_size=100" $null $adminToken).data.items
if (-not $tpls) { throw 'no templates' }

foreach ($it in $items) {
  $coverPath = Join-Path $SeedRoot (($it.cover -replace '/', '\'))
  if (-not (Test-Path -LiteralPath $coverPath)) { throw "missing $coverPath" }
  $match = $tpls | Where-Object { $_.name -eq $it.name } | Select-Object -First 1
  if (-not $match) {
    Write-Host ("skip (not found): {0}" -f $it.name)
    continue
  }
  Write-Host ("== refresh cover {0} ({1}) ==" -f $it.name, $match.template_id)
  $key = Upload-Cover $adminToken $coverPath
  # UpdateTemplate 尚无 cover 字段：直连 MySQL 更新
  $sql = "UPDATE templates SET cover_object_key='$key', updated_at=NOW() WHERE template_id='$($match.template_id)' AND deleted_at IS NULL;"
  $mysql = (docker ps --format '{{.Names}}' | Select-String -Pattern 'template-mall-mysql' | Select-Object -First 1)
  if (-not $mysql) { $mysql = (docker ps --format '{{.Names}}' | Select-String -Pattern 'mysql' | Select-Object -First 1) }
  if (-not $mysql) { throw 'mysql container not found' }
  docker exec "$mysql" mysql -uapp -papp template_order_db -e $sql
  Write-Host ("ok cover_object_key={0}" -f $key)
}

Write-Host 'REFRESH BUILTIN COVERS DONE'
