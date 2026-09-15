# Seed builtin templates (files + covers) via Admin upload APIs.
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

function Upload-One($Token, $FilePath, $ContentType) {
  $fileName = [IO.Path]::GetFileName($FilePath)
  $cred = (Invoke-Json POST "$Admin/api/upload/credential" @{
      filename     = $fileName
      content_type = $ContentType
    } $Token).data
  if (-not $cred.upload_url -or -not $cred.object_key) { throw "credential failed for $fileName" }
  $bytes = [IO.File]::ReadAllBytes($FilePath)
  Invoke-RestMethod -Method Put -Uri $cred.upload_url -ContentType $ContentType -Body $bytes | Out-Null
  return @{ object_key = $cred.object_key; size = $bytes.Length; filename = $fileName }
}

if (-not (Test-Path -LiteralPath $ManifestPath)) { throw "missing manifest: $ManifestPath" }
$items = Get-Content -LiteralPath $ManifestPath -Encoding UTF8 -Raw | ConvertFrom-Json

Write-Host '== mock login =='
$adminToken = (Invoke-Json POST "$Admin/api/auth/mock-login" @{ admin_id = 'seed-admin'; nickname = 'SeedAdmin' }).data.token
if (-not $adminToken) { throw 'admin token missing' }

foreach ($it in $items) {
  $filePath = Join-Path $SeedRoot (($it.file -replace '/', '\'))
  $coverPath = Join-Path $SeedRoot (($it.cover -replace '/', '\'))
  if (-not (Test-Path -LiteralPath $filePath)) { throw "missing $filePath" }
  if (-not (Test-Path -LiteralPath $coverPath)) { throw "missing $coverPath" }

  Write-Host ("== seed {0} ==" -f $it.name)
  $fileUp = Upload-One $adminToken $filePath $it.ctype
  $coverUp = Upload-One $adminToken $coverPath 'image/png'
  $resp = Invoke-Json POST "$Admin/api/upload/confirm" @{
    object_key         = $fileUp.object_key
    cover_object_key   = $coverUp.object_key
    original_filename  = $fileUp.filename
    file_type          = $it.type
    file_size          = [int64]$fileUp.size
    name               = $it.name
    price_fen          = [int64]$it.price
    price_type         = $it.price_type
    publish            = $true
  } $adminToken
  Write-Host ("ok template_id={0} type={1} price={2}" -f $resp.data.template_id, $it.type, $it.price_type)
}

Write-Host 'SEED BUILTIN TEMPLATES DONE'
