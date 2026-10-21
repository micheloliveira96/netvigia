# NetVigia Windows Agent (PowerShell 5.1+)
# Cadastre primeiro o PC como tipo windows; depois preencha os três valores abaixo.
$ServerUrl = "http://IP-DO-SERVIDOR:8787"
$DeviceId = "dev-ID-DO-CADASTRO"
$ApiKey = "COLOQUE-A-CHAVE-DO-NETVIGIA"
$ErrorActionPreference = "Stop"
try {
  $cpu = (Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average
  $os = Get-CimInstance Win32_OperatingSystem
  $total = [double]$os.TotalVisibleMemorySize * 1KB
  $free = [double]$os.FreePhysicalMemory * 1KB
  $disk = Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | Measure-Object -Property FreeSpace -Sum
  $payload = @{ device_id=$DeviceId; name=$env:COMPUTERNAME; metrics=@{ cpu_percent=[math]::Round([double]$cpu,2); memory_total_bytes=[math]::Round($total,0); memory_used_percent=[math]::Round((1-($free/$total))*100,2); memory_available_bytes=[math]::Round($free,0); disk_free_bytes=[math]::Round([double]$disk.Sum,0); uptime_seconds=[math]::Round(((Get-Date)-$os.LastBootUpTime).TotalSeconds,0) } } | ConvertTo-Json -Depth 5
  Invoke-RestMethod -Method Post -Uri "$ServerUrl/api/agent/metrics" -Headers @{ "X-API-Key"=$ApiKey } -ContentType "application/json" -Body $payload | Out-Null
} catch { Write-Error "Falha ao enviar métricas NetVigia: $_"; exit 1 }
