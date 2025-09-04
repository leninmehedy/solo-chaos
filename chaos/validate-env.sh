#!/bin/bash
# Standalone environment variable validation script
# Usage: ./validate-env.sh [--require-region] [--require-namespace] [--require-node-names]

set -e

# Function to validate REGION
validate_region() {
    if [[ -n "${REGION:-}" ]]; then
        VALID_REGIONS="us eu ap"
        if [[ ! " $VALID_REGIONS " =~ " $REGION " ]]; then
            echo "❌ ERROR: REGION '$REGION' is not valid. Allowed values: $VALID_REGIONS"
            return 1
        fi
        echo "✅ REGION '$REGION' is valid"
    elif [[ "${REQUIRE_REGION:-false}" == "true" ]]; then
        echo "❌ ERROR: REGION variable is required but not set"
        return 1
    fi
    return 0
}

# Function to validate NAMESPACE
validate_namespace() {
    if [[ -n "${NAMESPACE:-}" ]]; then
        if [[ ! "$NAMESPACE" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]] || [[ ${#NAMESPACE} -gt 63 ]]; then
            echo "❌ ERROR: NAMESPACE '$NAMESPACE' does not follow Kubernetes naming conventions"
            echo "   Must be lowercase alphanumeric with hyphens (not at start/end), max 63 chars"
            return 1
        fi
        echo "✅ NAMESPACE '$NAMESPACE' is valid"
    elif [[ "${REQUIRE_NAMESPACE:-false}" == "true" ]]; then
        echo "❌ ERROR: NAMESPACE variable is required but not set"
        return 1
    fi
    return 0
}

# Function to validate NODE_NAMES
validate_node_names() {
    if [[ -n "${NODE_NAMES:-}" ]]; then
        # Split NODE_NAMES by comma and validate each node
        old_ifs="$IFS"
        IFS=','
        set -- $NODE_NAMES
        IFS="$old_ifs"
        
        for node in "$@"; do
            node="${node#"${node%%[![:space:]]*}"}"  # trim leading whitespace
            node="${node%"${node##*[![:space:]]}"}"  # trim trailing whitespace
            if [[ ! "$node" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]] || [[ ${#node} -gt 63 ]]; then
                echo "❌ ERROR: NODE_NAME '$node' does not follow naming conventions"
                echo "   Must be lowercase alphanumeric with hyphens (not at start/end), max 63 chars"
                return 1
            fi
        done
        echo "✅ NODE_NAMES '$NODE_NAMES' are valid"
    elif [[ "${REQUIRE_NODE_NAMES:-false}" == "true" ]]; then
        echo "❌ ERROR: NODE_NAMES variable is required but not set"
        return 1
    fi
    return 0
}

# Function to validate UUID
validate_uuid() {
    if [[ -n "${UUID:-}" ]]; then
        if [[ ! "$UUID" =~ ^[a-zA-Z0-9-]+$ ]] || [[ ${#UUID} -gt 64 ]]; then
            echo "❌ ERROR: UUID '$UUID' contains invalid characters or is too long"
            echo "   Must be alphanumeric with hyphens, max 64 chars"
            return 1
        fi
        echo "✅ UUID '$UUID' is valid"
    fi
    return 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --require-region)
            REQUIRE_REGION="true"
            shift
            ;;
        --require-namespace)
            REQUIRE_NAMESPACE="true"
            shift
            ;;
        --require-node-names)
            REQUIRE_NODE_NAMES="true"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--require-region] [--require-namespace] [--require-node-names]"
            exit 1
            ;;
    esac
done

# Run all validations
validate_region && validate_namespace && validate_node_names && validate_uuid
