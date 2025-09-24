#!/bin/bash

# Development script for conx CMDB
# This script helps set up and run the development environment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if Docker is running
is_docker_running() {
    docker info >/dev/null 2>&1
}

# Function to show help
show_help() {
    echo "conx CMDB Development Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  setup       Set up the development environment"
    echo "  start       Start all services"
    echo "  stop        Stop all services"
    echo "  restart     Restart all services"
    echo "  logs        Show logs for all services"
    echo "  logs [svc]  Show logs for specific service"
    echo "  status      Show status of all services"
    echo "  test        Run all tests"
    echo "  test [unit]  Run unit tests only"
    echo "  test [int]   Run integration tests only"
    echo "  build       Build the application"
    echo "  clean       Clean up containers and volumes"
    echo "  reset       Reset the entire development environment"
    echo "  help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 setup     # Set up development environment"
    echo "  $0 start     # Start all services"
    echo "  $0 test unit # Run unit tests only"
}

# Function to set up development environment
setup_dev() {
    print_status "Setting up development environment..."
    
    # Check prerequisites
    if ! command_exists docker; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command_exists docker-compose; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    if ! is_docker_running; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    # Create .env file if it doesn't exist
    if [ ! -f "$PROJECT_DIR/.env" ]; then
        print_status "Creating .env file..."
        cat > "$PROJECT_DIR/.env" << EOF
# Development environment variables
POSTGRES_PASSWORD=dev_password
NEO4J_PASSWORD=dev_password
REDIS_PASSWORD=dev_password
JWT_SECRET=dev-secret-key-change-in-production
LOG_LEVEL=debug
LOG_FORMAT=json
VITE_API_URL=http://localhost:8080
VITE_NODE_ENV=development
EOF
        print_success ".env file created"
    fi
    
    # Create necessary directories
    mkdir -p "$PROJECT_DIR/logs"
    mkdir -p "$PROJECT_DIR/data/postgres"
    mkdir -p "$PROJECT_DIR/data/neo4j"
    mkdir -p "$PROJECT_DIR/data/redis"
    
    # Initialize databases
    print_status "Initializing databases..."
    cd "$PROJECT_DIR"
    docker-compose --profile init up -d
    
    # Wait for databases to be ready
    print_status "Waiting for databases to be ready..."
    sleep 10
    
    # Run database initialization
    print_status "Running database initialization..."
    docker-compose run --rm db-init
    
    print_success "Development environment setup complete!"
    print_status "You can now start the services with: $0 start"
}

# Function to start all services
start_services() {
    print_status "Starting all services..."
    cd "$PROJECT_DIR"
    docker-compose --profile dev up -d
    
    print_success "Services started!"
    print_status "API is available at: http://localhost:8080"
    print_status "Frontend is available at: http://localhost:3000"
    print_status "Neo4j Browser is available at: http://localhost:7474"
    print_status "PostgreSQL is available at: localhost:5432"
    print_status "Redis is available at: localhost:6379"
}

# Function to stop all services
stop_services() {
    print_status "Stopping all services..."
    cd "$PROJECT_DIR"
    docker-compose --profile dev down
    print_success "Services stopped!"
}

# Function to restart all services
restart_services() {
    print_status "Restarting all services..."
    stop_services
    sleep 2
    start_services
}

# Function to show logs
show_logs() {
    cd "$PROJECT_DIR"
    
    if [ -n "$2" ]; then
        print_status "Showing logs for $2..."
        docker-compose logs -f "$2"
    else
        print_status "Showing logs for all services..."
        docker-compose logs -f
    fi
}

# Function to show status
show_status() {
    cd "$PROJECT_DIR"
    print_status "Service status:"
    docker-compose ps
}

# Function to run tests
run_tests() {
    cd "$PROJECT_DIR"
    
    if [ -n "$2" ]; then
        case "$2" in
            "unit")
                print_status "Running unit tests..."
                go test -v -short ./...
                ;;
            "int")
                print_status "Running integration tests..."
                go test -v -run Integration ./...
                ;;
            *)
                print_error "Unknown test type: $2"
                print_error "Available test types: unit, int"
                exit 1
                ;;
        esac
    else
        print_status "Running all tests..."
        go test -v ./...
    fi
}

# Function to build application
build_app() {
    print_status "Building application..."
    cd "$PROJECT_DIR"
    
    # Build Go application
    print_status "Building API..."
    go build -o bin/api ./cmd/api
    
    # Build frontend (if web directory exists)
    if [ -d "$PROJECT_DIR/web" ]; then
        print_status "Building frontend..."
        cd "$PROJECT_DIR/web"
        npm install
        npm run build
        cd "$PROJECT_DIR"
    fi
    
    print_success "Application built successfully!"
}

# Function to clean up
cleanup() {
    print_status "Cleaning up containers and volumes..."
    cd "$PROJECT_DIR"
    docker-compose --profile dev down -v --remove-orphans
    print_success "Cleanup complete!"
}

# Function to reset environment
reset_env() {
    print_warning "This will remove all containers, volumes, and data!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup
        rm -f "$PROJECT_DIR/.env"
        rm -rf "$PROJECT_DIR/data"
        print_success "Environment reset complete!"
    else
        print_status "Reset cancelled."
    fi
}

# Main script logic
case "${1:-}" in
    "setup")
        setup_dev
        ;;
    "start")
        start_services
        ;;
    "stop")
        stop_services
        ;;
    "restart")
        restart_services
        ;;
    "logs")
        show_logs "$@"
        ;;
    "status")
        show_status
        ;;
    "test")
        run_tests "$@"
        ;;
    "build")
        build_app
        ;;
    "clean")
        cleanup
        ;;
    "reset")
        reset_env
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
