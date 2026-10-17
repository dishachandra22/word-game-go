#!/bin/bash
# Kubernetes Multi-Namespace & Secret Setup Script
# Usage: ./k8s/setup-k8s.sh <docker-user> <docker-token>

DOCKER_USER="${1:-dishachandra22}"
DOCKER_TOKEN="${2:-}"

NAMESPACES=("qa" "stage" "prod")

for ns in "${NAMESPACES[@]}"; do
    echo "Creating namespace: $ns ..."
    kubectl create namespace "$ns" --dry-run=client -o yaml | kubectl apply -f -

    if [ -n "$DOCKER_TOKEN" ]; then
        echo "Configuring dockerhub secret in $ns ..."
        kubectl delete secret dockerhub -n "$ns" --ignore-not-found
        kubectl create secret docker-registry dockerhub \
            --docker-username="$DOCKER_USER" \
            --docker-password="$DOCKER_TOKEN" \
            -n "$ns"
    fi

    echo "Applying manifests for $ns ..."
    kubectl apply -f "k8s/$ns.yaml"
done

echo "Kubernetes setup complete!"
echo "Check pods with: kubectl get pods -A"
