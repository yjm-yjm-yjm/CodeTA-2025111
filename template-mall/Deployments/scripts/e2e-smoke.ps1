$ErrorActionPreference = 'Stop'
$Admin = 'http://127.0.0.1:8081'
$CAPI = 'http://127.0.0.1:8080'
$Pay = 'http://127.0.0.1:8082'

function Invoke-Json($Method, $Url, $Body = $null, $Token = $null) {
  $headers = @{}
  if ($Token) { $headers['Authorization'] = "Bearer $Token" }
  if ($null -ne $Body) {
    $json = $Body | ConvertTo-Json -Compress -Depth 8
    return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -ContentType 'application/json' -Body $json
  }
  return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers
}

Write-Host '== health =='
Invoke-RestMethod "$CAPI/healthz" | Out-Null
Invoke-RestMethod "$Admin/healthz" | Out-Null
Invoke-RestMethod "$Pay/healthz" | Out-Null
Write-Host 'health ok'

Write-Host '== admin login (mock API; requires AUTH_MODE=mock or temporarily switch for smoke) =='
try {
  $adminLogin = Invoke-Json POST "$Admin/api/auth/mock-login" @{ admin_id = 'e2e-admin'; nickname = 'E2EAdmin' }
} catch {
  throw 'admin mock-login failed. API smoke needs AUTH_MODE=mock (UI 已只保留 WPS). Set AUTH_MODE=mock, restart Admin, re-run.'
}
$adminToken = $adminLogin.data.token
if (-not $adminToken) { throw 'admin token missing' }

Write-Host '== upload free template =='
$tmp = Join-Path $env:TEMP ("e2e-" + [guid]::NewGuid().ToString('N') + '.pptx')
# minimal zip-based pptx stub bytes
[IO.File]::WriteAllBytes($tmp, [byte[]](0x50,0x4B,0x03,0x04) + ([byte[]](0x00) * 2048))
$fileName = [IO.Path]::GetFileName($tmp)
$credResp = Invoke-Json POST "$Admin/api/upload/credential" @{ filename = $fileName; content_type = 'application/vnd.openxmlformats-officedocument.presentationml.presentation' } $adminToken
$cred = $credResp.data
if (-not $cred.upload_url) { throw 'upload credential missing' }

$bytes = [IO.File]::ReadAllBytes($tmp)
Invoke-RestMethod -Method Put -Uri $cred.upload_url -ContentType $cred.content_type -Body $bytes | Out-Null

$tplResp = Invoke-Json POST "$Admin/api/upload/confirm" @{
  object_key = $cred.object_key
  original_filename = $fileName
  file_type = 'pptx'
  file_size = $bytes.Length
  name = 'E2E Free Template'
  price_fen = 0
  price_type = 'free'
  publish = $true
} $adminToken
$templateId = $tplResp.data.template_id
Write-Host "template_id=$templateId"

Write-Host '== c-end register & download free =='
# 构造合法 11 位号段，避免与已有账号冲突
$phone = '139' + (Get-Random -Minimum 10000000 -Maximum 99999999).ToString()
$reg = Invoke-Json POST "$CAPI/api/auth/register" @{ phone = $phone; password = 'Passw0rd!'; nickname = 'E2E' }
$cToken = $reg.data.token
$dl = Invoke-Json POST "$CAPI/api/templates/$templateId/download" $null $cToken
if ($dl.data.result -ne 'granted') { throw "expected granted, got $($dl.data | ConvertTo-Json -Compress)" }
Write-Host "free download ok order=$($dl.data.order_id)"

Write-Host '== create paid template =='
$tmp2 = Join-Path $env:TEMP ("e2e-paid-" + [guid]::NewGuid().ToString('N') + '.docx')
[IO.File]::WriteAllBytes($tmp2, [byte[]](0x50,0x4B,0x03,0x04) + ([byte[]](0x01) * 1024))
$fileName2 = [IO.Path]::GetFileName($tmp2)
$cred2 = (Invoke-Json POST "$Admin/api/upload/credential" @{ filename = $fileName2; content_type = 'application/msword' } $adminToken).data
$bytes2 = [IO.File]::ReadAllBytes($tmp2)
Invoke-RestMethod -Method Put -Uri $cred2.upload_url -ContentType $cred2.content_type -Body $bytes2 | Out-Null
$paidTpl = (Invoke-Json POST "$Admin/api/upload/confirm" @{
  object_key = $cred2.object_key
  original_filename = $fileName2
  file_type = 'docx'
  file_size = $bytes2.Length
  name = 'E2E Paid Template'
  price_fen = 100
  price_type = 'paid'
  publish = $true
} $adminToken).data

Write-Host '== paid download => payment_required =='
$needPay = Invoke-Json POST "$CAPI/api/templates/$($paidTpl.template_id)/download" $null $cToken
if ($needPay.data.result -ne 'payment_required') { throw 'expected payment_required' }
$orderId = $needPay.data.order.order_id
$payOrderId = $needPay.data.payment.pay_order_id
$amount = $needPay.data.payment.amount_fen
Write-Host "order=$orderId pay=$payOrderId amount=$amount"

Write-Host '== pay callback =='
$cb = Invoke-Json POST "$Pay/api/paycallback" @{
  callback_id = 'e2e-cb-' + [guid]::NewGuid().ToString('N')
  pay_order_id = $payOrderId
  order_id = $orderId
  amount_fen = $amount
}
if (-not $cb.accepted) { throw "callback not accepted: $($cb | ConvertTo-Json -Compress)" }

# Kafka 消费组冷启动/再均衡可能超过 2s
Start-Sleep -Seconds 5
Write-Host '== download after pay =='
$after = Invoke-Json POST "$CAPI/api/templates/$($paidTpl.template_id)/download" $null $cToken
if ($after.data.result -ne 'granted') { throw "expected granted after pay: $($after.data | ConvertTo-Json -Compress)" }
Write-Host "paid download ok order=$($after.data.order_id)"

Write-Host 'E2E SMOKE PASSED'
