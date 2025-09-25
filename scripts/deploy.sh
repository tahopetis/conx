#!/bin/bash

# CMDB Connect Deployment Script
# This script handles the deployment of the CMDB Connect application

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ENVIRONMENT=${1:-staging}
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yml"
ENV_FILE="${PROJECT_ROOT}/.env"

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

# Check if required files exist
check_requirements() {
    log "Checking deployment requirements..."
    
    if [[ ! -f "$DOCKER_COMPOSE_FILE" ]]; then
        error "Docker Compose file not found: $DOCKER_COMPOSE_FILE"
    fi
    
    if [[ ! -f "$ENV_FILE" ]]; then
        error "Environment file not found: $ENV_FILE"
    fi
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed"
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose is not installed"
    fi
    
    log "All requirements satisfied"
}

# Load environment variables
load_env() {
    log "Loading environment variables..."
    set -a
    source "$ENV_FILE"
    set +a
    log "Environment variables loaded"
}

# Backup existing deployment
backup_deployment() {
    log "Creating backup of existing deployment..."
    
    BACKUP_DIR="${PROJECT_ROOT}/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # Backup Docker volumes
    docker-compose -f "$DOCKER_COMPOSE_FILE" down
    docker run --rm -v cmdb_postgres_data:/data -v "$BACKUP_DIR:/backup" alpine tar czf "/backup/postgres_data.tar.gz" -C /data .
    docker run --rm -v cmdb_neo4j_data:/data -v "$BACKUP_DIR:/backup" alpine tar czf "/backup/neo4j_data.tar.gz" -C /data .
    docker run --rm -v cmdb_redis_data:/data -v "$BACKUP_DIR:/backup" alpine tar czf "/backup/redis_data.tar.gz" -C /data .
    
    log "Backup created at: $BACKUP_DIR"
}

# Build and deploy services
deploy_services() {
    log "Building and deploying services..."
    
    # Pull latest images
    docker-compose -f "$DOCKER_COMPOSE_FILE" pull
    
    # Build services
    docker-compose -f "$DOCKER_COMPOSE_FILE" build --no-cache
    
    # Start services
    docker-compose -f "$DOCKER_COMPOSE_FILE" up -d
    
    log "Services deployed successfully"
}

# Health check
health_check() {
    log "Performing health check..."
    
    # Wait for services to be healthy
    local services=("postgres" "neo4j" "redis" "backend" "frontend")
    local max_attempts=30
    local attempt=1
    
    for service in "${services[@]}"; do
        log "Checking health of $service..."
        
        while [[ $attempt -le $max_attempts ]]; do
            if docker-compose -f "$DOCKER_COMPOSE_FILE" ps -q "$service" | xargs docker inspect --format='{{.State.Health.Status}}' 2>/dev/null | grep -q "healthy"; then
                log "$service is healthy"
                break
            fi
            
            if [[ $attempt -eq $max_attempts ]]; then
                error "$service failed health check"
            fi
            
            warn "$service health check attempt $attempt/$max_attempts failed. Retrying in 10 seconds..."
            sleep 10
            ((attempt++))
        done
        
        attempt=1
    done
    
    log "All services are healthy"
}

# Run tests
run_tests() {
    log "Running deployment tests..."
    
    # Test backend health
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        log "Backend health check passed"
    else
        error "Backend health check failed"
    fi
    
    # Test frontend health
    if curl -f http://localhost/health > /dev/null 2>&1; then
        log "Frontend health check passed"
    else
        error "Frontend health check failed"
    fi
    
    # Test API endpoints
    if curl -f http://localhost:8080/api/v1/health > /dev/null 2>&1; then
        log "API health check passed"
    else
        error "API health check failed"
    fi
    
    log "All tests passed"
}

# Cleanup
cleanup() {
    log "Cleaning up old resources..."
    
    # Remove unused Docker images
    docker image prune -f
    
    # Remove unused Docker networks
    docker network prune -f
    
    # Remove unused Docker volumes
    docker volume prune -f
    
    log "Cleanup completed"
}

# Main deployment function
main() {
    log "Starting CMDB Connect deployment for $ENVIRONMENT environment..."
    
    check_requirements
    load_env
    
    if [[ "$ENVIRONMENT" == "production" ]]; then
        backup_deployment
    fi
    
    deploy_services
    health_check
    run_tests
    cleanup
    
    log "Deployment completed successfully!"
    log "Application is available at: http://localhost"
    log "API is available at: http://localhost:8080"
    log "Grafana dashboard is available at: http://localhost:3000"
    log "Prometheus metrics are available at: http://localhost:9090"
}

# Handle script interruption
trap 'error "Deployment interrupted"' INT TERM

# Run main function
main "$@"
