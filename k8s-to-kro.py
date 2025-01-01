#!/usr/bin/env python3
import yaml
import sys
import re
from typing import Dict, List, Any

def extract_variables(manifest: Dict) -> Dict[str, Any]:
    """Extract potential variables from manifest."""
    variables = {}
    
    # Common patterns to look for variables
    if 'metadata' in manifest and 'name' in manifest['metadata']:
        variables['name'] = manifest['metadata']['name']
    
    if manifest['kind'] == 'Deployment':
        if 'spec' in manifest and 'template' in manifest['spec']:
            containers = manifest['spec']['template']['spec'].get('containers', [])
            if containers:
                variables['image'] = containers[0]['image']
                if 'resources' in containers[0]:
                    variables['resources'] = containers[0]['resources']
    
    return variables

def create_schema(variables: Dict[str, Any]) -> Dict:
    """Create Kro schema from variables."""
    schema = {
        'apiVersion': 'v1alpha1',
        'kind': 'Application',
        'spec': {}
    }
    
    # Convert variables to schema fields
    for key, value in variables.items():
        if isinstance(value, str):
            if key == 'name':
                schema['spec'][key] = 'string | required=true'
            else:
                schema['spec'][key] = f'string | default="{value}"'
        elif isinstance(value, dict):
            schema['spec'][key] = value
    
    return schema

def convert_manifest(manifest: Dict, variables: Dict[str, Any]) -> Dict:
    """Convert K8s manifest to Kro template with variables."""
    def replace_with_variables(value: str, variables: Dict[str, Any]) -> str:
        for var_name, var_value in variables.items():
            if str(var_value) == value:
                return f'${{schema.spec.{var_name}}}'
        return value

    def process_dict(d: Dict, variables: Dict[str, Any]) -> Dict:
        result = {}
        for k, v in d.items():
            if isinstance(v, dict):
                result[k] = process_dict(v, variables)
            elif isinstance(v, list):
                result[k] = [process_dict(i, variables) if isinstance(i, dict) else i for i in v]
            elif isinstance(v, str):
                result[k] = replace_with_variables(v, variables)
            else:
                result[k] = v
        return result

    return process_dict(manifest, variables)

def create_resource_group(manifests: List[Dict]) -> Dict:
    """Create a Kro ResourceGroup from K8s manifests."""
    # Extract variables from all manifests
    all_variables = {}
    for manifest in manifests:
        all_variables.update(extract_variables(manifest))

    # Create ResourceGroup
    resource_group = {
        'apiVersion': 'kro.run/v1alpha1',
        'kind': 'ResourceGroup',
        'metadata': {
            'name': 'converted-application'
        },
        'spec': {
            'schema': create_schema(all_variables),
            'resources': []
        }
    }

    # Convert each manifest to a resource template
    for manifest in manifests:
        resource = {
            'id': manifest['kind'].lower(),
            'template': convert_manifest(manifest, all_variables)
        }
        resource_group['spec']['resources'].append(resource)

    return resource_group

def main():
    if len(sys.argv) < 2:
        print("Usage: k8s-to-kro.py <kubernetes-manifest.yaml>")
        sys.exit(1)

    # Read input manifests
    manifests = []
    with open(sys.argv[1], 'r') as f:
        for doc in yaml.safe_load_all(f):
            if doc:  # Skip empty documents
                manifests.append(doc)

    # Convert to ResourceGroup
    resource_group = create_resource_group(manifests)

    # Output ResourceGroup
    print(yaml.dump(resource_group, default_flow_style=False)) 