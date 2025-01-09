#!/bin/bash

# Function to create directories
create_directories() {
    local namespace=$1
    echo "Creating directories for namespace: $namespace"
    
    mkdir -p "$namespace/rg"
    mkdir -p "$namespace/instance"
    mkdir -p "$namespace/templates"
    
    echo "✓ Created directory structure"
}

# Function to create resource group files
create_rg_files() {
    local namespace=$1
    echo "Creating resource group files..."
    
    # List of resource group files to create
    declare -a rg_files=("gateway" "git-gate" "gitrepo" "orig")
    
    for base in "${rg_files[@]}"; do
        touch "$namespace/rg/$base-rg.yaml"
        echo "✓ Created $base-rg.yaml"
    done
}

# Function to create instance files
create_instance_files() {
    local namespace=$1
    echo "Creating instance files..."
    
    # List of instance files to create
    declare -a instance_files=("gateway" "git-gate" "gitrepo" "orig")
    
    for base in "${instance_files[@]}"; do
        touch "$namespace/instance/$base-instance.yaml"
        echo "✓ Created $base-instance.yaml"
    done
}

# Function to create template files
create_template_files() {
    local namespace=$1
    echo "Creating template files..."
    
    # List of template files to create
    declare -a template_files=("gateway" "git-gate" "gitrepo" "orig")
    
    for base in "${template_files[@]}"; do
        touch "$namespace/templates/$base.yaml"
        echo "✓ Created $base.yaml"
    done
}

# Main function
main() {
    # Check if namespace argument is provided
    if [ $# -eq 0 ]; then
        echo "Error: Please provide a namespace name"
        echo "Usage: $0 <namespace-name>"
        exit 1
    fi

    local namespace=$1
    
    echo "Starting namespace creation process..."
    
    # Create the directory structure
    create_directories "$namespace"
    
    # Create all the files
    create_rg_files "$namespace"
    create_instance_files "$namespace"
    create_template_files "$namespace"
    
    echo "✓ Namespace structure created successfully!"
    echo "
Directory structure created:
$namespace/
├── rg/
│   ├── gateway-rg.yaml
│   ├── git-gate-rg.yaml
│   ├── gitrepo-rg.yaml
│   └── orig-rg.yaml
├── instance/
│   ├── gateway-instance.yaml
│   ├── git-gate-instance.yaml
│   ├── gitrepo-instance.yaml
│   └── orig-instance.yaml
└── templates/
    ├── gateway.yaml
    ├── git-gate.yaml
    ├── gitrepo.yaml
    └── orig.yaml"
}

# Execute main function with all arguments
main "$@" 