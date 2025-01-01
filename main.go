package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResourceGroup represents a Kro ResourceGroup
type ResourceGroup struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Spec       ResourceGroupSpec `yaml:"spec"`
}

// ResourceGroupSpec represents the spec of a ResourceGroup
type ResourceGroupSpec struct {
	Schema    Schema     `yaml:"schema"`
	Resources []Resource `yaml:"resources"`
}

// Schema represents the schema section of a ResourceGroup
type Schema struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Spec       map[string]interface{} `yaml:"spec"`
	Status     map[string]string      `yaml:"status,omitempty"`
}

// Resource represents a resource in a ResourceGroup
type Resource struct {
	ID          string                 `yaml:"id"`
	Template    map[string]interface{} `yaml:"template"`
	IncludeWhen []string               `yaml:"includeWhen,omitempty"`
	DependsOn   []string               `yaml:"dependsOn,omitempty"`
}

// ResourceRelation represents a relationship between resources
type ResourceRelation struct {
	Source       string
	Target       string
	RelationType string
}

// Variable represents a schema variable with validation
type Variable struct {
	Type        string
	Required    bool
	Default     interface{}
	Description string
	Validation  map[string]interface{}
}

func detectResourceRelations(manifests []map[string]interface{}) []ResourceRelation {
	relations := []ResourceRelation{}

	for _, source := range manifests {
		sourceKind := source["kind"].(string)

		for _, target := range manifests {
			targetKind := target["kind"].(string)
			targetName := target["metadata"].(map[string]interface{})["name"].(string)

			// Service -> Deployment relationship (via selector)
			if sourceKind == "Service" && targetKind == "Deployment" {
				if svcSpec, ok := source["spec"].(map[string]interface{}); ok {
					if selector, ok := svcSpec["selector"].(map[string]interface{}); ok {
						if deploySpec, ok := target["spec"].(map[string]interface{}); ok {
							if template, ok := deploySpec["template"].(map[string]interface{}); ok {
								if metadata, ok := template["metadata"].(map[string]interface{}); ok {
									if labels, ok := metadata["labels"].(map[string]interface{}); ok {
										if matchLabels(selector, labels) {
											relations = append(relations, ResourceRelation{
												Source:       strings.ToLower(sourceKind),
												Target:       strings.ToLower(targetKind),
												RelationType: "selector",
											})
										}
									}
								}
							}
						}
					}
				}
			}

			// Ingress -> Service relationship
			if sourceKind == "Ingress" && targetKind == "Service" {
				if spec, ok := source["spec"].(map[string]interface{}); ok {
					if rules, ok := spec["rules"].([]interface{}); ok {
						for _, rule := range rules {
							if ruleMap, ok := rule.(map[string]interface{}); ok {
								if http, ok := ruleMap["http"].(map[string]interface{}); ok {
									if paths, ok := http["paths"].([]interface{}); ok {
										for _, path := range paths {
											if pathMap, ok := path.(map[string]interface{}); ok {
												if backend, ok := pathMap["backend"].(map[string]interface{}); ok {
													if service, ok := backend["service"].(map[string]interface{}); ok {
														if name, ok := service["name"].(string); ok && name == targetName {
															relations = append(relations, ResourceRelation{
																Source:       strings.ToLower(sourceKind),
																Target:       strings.ToLower(targetKind),
																RelationType: "backend",
															})
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return relations
}

func matchLabels(selector, labels map[string]interface{}) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func extractVariables(manifest map[string]interface{}) map[string]Variable {
	variables := make(map[string]Variable)

	// Extract name
	if metadata, ok := manifest["metadata"].(map[string]interface{}); ok {
		if _, ok := metadata["name"].(string); ok {
			variables["name"] = Variable{
				Type:        "string",
				Required:    true,
				Description: "Name of the application",
			}
		}
	}

	// Extract variables from Deployment
	if kind, ok := manifest["kind"].(string); ok && kind == "Deployment" {
		if spec, ok := manifest["spec"].(map[string]interface{}); ok {
			// Extract replicas
			if replicas, ok := spec["replicas"].(int); ok {
				variables["replicas"] = Variable{
					Type:        "integer",
					Default:     replicas,
					Description: "Number of replicas",
					Validation: map[string]interface{}{
						"minimum": 1,
						"maximum": 100,
					},
				}
			}

			if template, ok := spec["template"].(map[string]interface{}); ok {
				if podSpec, ok := template["spec"].(map[string]interface{}); ok {
					if containers, ok := podSpec["containers"].([]interface{}); ok && len(containers) > 0 {
						if container, ok := containers[0].(map[string]interface{}); ok {
							// Extract image
							if image, ok := container["image"].(string); ok {
								variables["image"] = Variable{
									Type:        "string",
									Default:     image,
									Description: "Container image to deploy",
								}
							}

							// Extract ports
							if ports, ok := container["ports"].([]interface{}); ok {
								for _, port := range ports {
									if portMap, ok := port.(map[string]interface{}); ok {
										if containerPort, ok := portMap["containerPort"].(int); ok {
											variables["port"] = Variable{
												Type:        "integer",
												Default:     containerPort,
												Description: "Container port",
												Validation: map[string]interface{}{
													"minimum": 1,
													"maximum": 65535,
												},
											}
										}
									}
								}
							}

							// Extract resources
							if resources, ok := container["resources"].(map[string]interface{}); ok {
								variables["resources"] = Variable{
									Type:        "object",
									Default:     resources,
									Description: "Container resource requirements",
								}
							}
						}
					}
				}
			}
		}
	}

	// Extract variables from Service
	if kind, ok := manifest["kind"].(string); ok && kind == "Service" {
		if spec, ok := manifest["spec"].(map[string]interface{}); ok {
			// Extract service type
			if serviceType, ok := spec["type"].(string); ok {
				variables["serviceType"] = Variable{
					Type:        "string",
					Default:     serviceType,
					Description: "Service type",
					Validation: map[string]interface{}{
						"enum": []string{"ClusterIP", "NodePort", "LoadBalancer"},
					},
				}
			}
		}
	}

	// Extract variables from Ingress
	if kind, ok := manifest["kind"].(string); ok && kind == "Ingress" {
		variables["ingress"] = Variable{
			Type: "object",
			Default: map[string]interface{}{
				"enabled": false,
				"host":    "example.com",
				"path":    "/",
			},
			Description: "Ingress configuration",
		}
	}

	return variables
}

func createSchema(variables map[string]Variable) Schema {
	schema := Schema{
		APIVersion: "v1alpha1",
		Kind:       "Application",
		Spec:       make(map[string]interface{}),
		Status: map[string]string{
			"phase":             "${deployment.status.phase}",
			"availableReplicas": "${deployment.status.availableReplicas}",
			"message":           "${deployment.status.conditions[0].message}",
			"serviceEndpoint":   "${service.spec.clusterIP}",
		},
	}

	for key, variable := range variables {
		schemaField := fmt.Sprintf("%s", variable.Type)

		if variable.Required {
			schemaField += " | required=true"
		}

		if variable.Default != nil {
			schemaField += fmt.Sprintf(" | default=\"%v\"", variable.Default)
		}

		if variable.Description != "" {
			schemaField += fmt.Sprintf(" | description=\"%s\"", variable.Description)
		}

		if validation := variable.Validation; validation != nil {
			for k, v := range validation {
				schemaField += fmt.Sprintf(" | %s=%v", k, v)
			}
		}

		schema.Spec[key] = schemaField
	}

	return schema
}

func replaceWithVariables(manifest map[string]interface{}, variables map[string]Variable) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range manifest {
		switch v := value.(type) {
		case map[string]interface{}:
			result[key] = replaceWithVariables(v, variables)
		case []interface{}:
			newArray := make([]interface{}, len(v))
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					newArray[i] = replaceWithVariables(itemMap, variables)
				} else {
					newArray[i] = item
				}
			}
			result[key] = newArray
		case string:
			replaced := false
			for varName, variable := range variables {
				if variable.Default != nil && fmt.Sprintf("%v", variable.Default) == v {
					result[key] = fmt.Sprintf("${schema.spec.%s}", varName)
					replaced = true
					break
				}
			}
			if !replaced {
				result[key] = v
			}
		default:
			result[key] = v
		}
	}

	return result
}

func createResourceGroup(manifests []map[string]interface{}) ResourceGroup {
	// Extract variables from all manifests
	allVariables := make(map[string]Variable)
	for _, manifest := range manifests {
		vars := extractVariables(manifest)
		for k, v := range vars {
			allVariables[k] = v
		}
	}

	// Detect resource relations
	relations := detectResourceRelations(manifests)

	// Create ResourceGroup
	rg := ResourceGroup{
		APIVersion: "kro.run/v1alpha1",
		Kind:       "ResourceGroup",
		Metadata: map[string]string{
			"name": "converted-application",
		},
		Spec: ResourceGroupSpec{
			Schema:    createSchema(allVariables),
			Resources: make([]Resource, 0),
		},
	}

	// Build dependency graph
	dependencies := make(map[string][]string)
	for _, relation := range relations {
		dependencies[relation.Source] = append(dependencies[relation.Source], relation.Target)
	}

	// Convert each manifest to a resource
	for _, manifest := range manifests {
		kind := manifest["kind"].(string)
		resourceID := strings.ToLower(kind)

		resource := Resource{
			ID:       resourceID,
			Template: replaceWithVariables(manifest, allVariables),
		}

		// Add dependencies
		if deps, ok := dependencies[resourceID]; ok {
			resource.DependsOn = deps
		}

		// Add conditional inclusion for Ingress
		if kind == "Ingress" {
			resource.IncludeWhen = []string{"${schema.spec.ingress.enabled}"}
		}

		rg.Spec.Resources = append(rg.Spec.Resources, resource)
	}

	return rg
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: k8s-to-kro <kubernetes-manifest.yaml>")
		os.Exit(1)
	}

	// Read input file
	filename := os.Args[1]
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Split YAML documents
	var manifests []map[string]interface{}
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	for {
		var doc map[string]interface{}
		if err := decoder.Decode(&doc); err != nil {
			break
		}
		if doc != nil {
			manifests = append(manifests, doc)
		}
	}

	// Convert to ResourceGroup
	rg := createResourceGroup(manifests)

	// Output ResourceGroup
	output, err := yaml.Marshal(rg)
	if err != nil {
		log.Fatalf("Error marshaling output: %v", err)
	}

	// Write to file
	outputFile := strings.TrimSuffix(filename, filepath.Ext(filename)) + "-kro.yaml"
	if err := ioutil.WriteFile(outputFile, output, 0644); err != nil {
		log.Fatalf("Error writing output: %v", err)
	}

	fmt.Printf("Successfully converted to Kro ResourceGroup: %s\n", outputFile)
}
