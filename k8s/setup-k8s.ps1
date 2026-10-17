# Kubernetes Multi-Namespace & Secret Setup Script
# Usage: .\k8s\setup-k8s.ps1 -DockerUser <user> -DockerToken <token>

param (
    [Parameter(Mandatory=$false)]
    [string]$DockerUser = "dishachandra22",
    [Parameter(Mandatory=$false)]
    [string]$DockerToken = ""
)

$namespaces = @("qa", "stage", "prod")

foreach ($ns in $namespaces) {
    Write-Host "Creating namespace: $ns ..." -ForegroundColor Cyan
    kubectl create namespace $ns --dry-run=client -o yaml | kubectl apply -f -

    if ($DockerToken -ne "") {
        Write-Host "Configuring dockerhub secret in $ns ..." -ForegroundColor Green
        kubectl delete secret dockerhub -n $ns --ignore-not-found
        kubectl create secret docker-registry dockerhub `
            --docker-username=$DockerUser `
            --docker-password=$DockerToken `
            -n $ns
    } else {
        Write-Host "Note: Provide -DockerToken to automatically create dockerhub pull secret in $ns" -ForegroundColor Yellow
    }

    Write-Host "Applying manifests for $ns ..." -ForegroundColor Cyan
    kubectl apply -f "k8s/$ns.yaml"
}

Write-Host "Kubernetes deployments created successfully!" -ForegroundColor Green
Write-Host "Verify pods status with: kubectl get pods -A" -ForegroundColor White
