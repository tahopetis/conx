package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SchemaValidator provides validation for CI and relationship data against schemas
type SchemaValidator struct{}

// NewSchemaValidator creates a new SchemaValidator instance
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{}
}

// ValidateCIAgainstSchema validates CI data against a CI type schema
func (v *SchemaValidator) ValidateCIAgainstSchema(ci CI, schema CITypeSchema) ValidationResult {
	result := ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
	}

	// Parse CI attributes into a map for easier validation
	var attributes map[string]interface{}
	if len(ci.Attributes) > 0 {
		if err := json.Unmarshal(ci.Attributes, &attributes); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "attributes",
				Value:   string(ci.Attributes),
				Message: "Invalid JSON in attributes",
			})
			return result
		}
	} else {
		attributes = make(map[string]interface{})
	}

	// Validate each attribute in the schema
	for _, attrDef := range schema.Attributes {
		value, exists := attributes[attrDef.Name]
		
		// Check if required attribute is missing
		if attrDef.Required && !exists {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   attrDef.Name,
				Value:   nil,
				Message: fmt.Sprintf("Required attribute '%s' is missing", attrDef.Name),
			})
			continue
		}

		// Skip validation if attribute doesn't exist and is not required
		if !exists {
			continue
		}

		// Validate attribute type and constraints
		if err := v.validateAttribute(attrDef.Name, value, attrDef.Type, attrDef.Validation); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, *err)
		}
	}

	// Check for extra attributes not defined in schema
	for attrName := range attributes {
		found := false
		for _, attrDef := range schema.Attributes {
			if attrDef.Name == attrName {
				found = true
				break
			}
		}
		if !found {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   attrName,
				Value:   attributes[attrName],
				Message: fmt.Sprintf("Attribute '%s' is not defined in schema", attrName),
			})
		}
	}

	return result
}

// ValidateRelationshipAgainstSchema validates relationship data against a relationship type schema
func (v *SchemaValidator) ValidateRelationshipAgainstSchema(relationship CIRelationship, schema RelationshipTypeSchema) ValidationResult {
	result := ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
	}

	// Parse relationship attributes into a map for easier validation
	var attributes map[string]interface{}
	if len(relationship.Attributes) > 0 {
		if err := json.Unmarshal(relationship.Attributes, &attributes); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "attributes",
				Value:   string(relationship.Attributes),
				Message: "Invalid JSON in attributes",
			})
			return result
		}
	} else {
		attributes = make(map[string]interface{})
	}

	// Validate each attribute in the schema
	for _, attrDef := range schema.Attributes {
		value, exists := attributes[attrDef.Name]
		
		// Check if required attribute is missing
		if attrDef.Required && !exists {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   attrDef.Name,
				Value:   nil,
				Message: fmt.Sprintf("Required attribute '%s' is missing", attrDef.Name),
			})
			continue
		}

		// Skip validation if attribute doesn't exist and is not required
		if !exists {
			continue
		}

		// Validate attribute type and constraints
		if err := v.validateAttribute(attrDef.Name, value, attrDef.Type, attrDef.Validation); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, *err)
		}
	}

	// Check for extra attributes not defined in schema
	for attrName := range attributes {
		found := false
		for _, attrDef := range schema.Attributes {
			if attrDef.Name == attrName {
				found = true
				break
			}
		}
		if !found {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:   attrName,
				Value:   attributes[attrName],
				Message: fmt.Sprintf("Attribute '%s' is not defined in schema", attrName),
			})
		}
	}

	return result
}

// validateAttribute validates a single attribute value against its type and validation rules
func (v *SchemaValidator) validateAttribute(fieldName string, value interface{}, attrType string, validation map[string]interface{}) *ValidationError {
	// Type validation
	if err := v.validateType(fieldName, value, attrType); err != nil {
		return err
	}

	// Custom validation rules
	if validation != nil {
		if err := v.validateCustomRules(fieldName, value, validation); err != nil {
			return err
		}
	}

	return nil
}

// validateType validates that a value matches the expected type
func (v *SchemaValidator) validateType(fieldName string, value interface{}, expectedType string) *ValidationError {
	switch expectedType {
	case AttributeTypeString:
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:   fieldName,
				Value:   value,
				Message: fmt.Sprintf("Expected string, got %T", value),
			}
		}
	case AttributeTypeNumber:
		var num float64
		switch val := value.(type) {
		case float64:
			num = val
		case float32:
			num = float64(val)
		case int:
			num = float64(val)
		case int64:
			num = float64(val)
		case int32:
			num = float64(val)
		default:
			return &ValidationError{
				Field:   fieldName,
				Value:   value,
				Message: fmt.Sprintf("Expected number, got %T", value),
			}
		}
		value = num
	case AttributeTypeBoolean:
		if _, ok := value.(bool); !ok {
			return &ValidationError{
				Field:   fieldName,
				Value:   value,
				Message: fmt.Sprintf("Expected boolean, got %T", value),
			}
		}
	case AttributeTypeDate:
		var dateStr string
		switch val := value.(type) {
		case string:
			dateStr = val
		default:
			return &ValidationError{
				Field:   fieldName,
				Value:   value,
				Message: fmt.Sprintf("Expected date string, got %T", value),
			}
		}
		
		// Try to parse as ISO 8601 date
		_, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			// Try other common date formats
			formats := []string{
				"2006-01-02",
				"2006/01/02",
				"01-02-2006",
				"01/02/2006",
				"January 2, 2006",
			}
			
			parsed := false
			for _, format := range formats {
				if _, err := time.Parse(format, dateStr); err == nil {
					parsed = true
					break
				}
			}
			
			if !parsed {
				return &ValidationError{
					Field:   fieldName,
					Value:   value,
					Message: "Invalid date format, expected ISO 8601 or common date format",
				}
			}
		}
	case AttributeTypeArray:
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return &ValidationError{
				Field:   fieldName,
				Value:   value,
				Message: fmt.Sprintf("Expected array, got %T", value),
			}
		}
	case AttributeTypeObject:
		if reflect.TypeOf(value).Kind() != reflect.Map {
			return &ValidationError{
				Field:   fieldName,
				Value:   value,
				Message: fmt.Sprintf("Expected object, got %T", value),
			}
		}
	default:
		return &ValidationError{
			Field:   fieldName,
			Value:   value,
			Message: fmt.Sprintf("Unknown type: %s", expectedType),
		}
	}

	return nil
}

// validateCustomRules validates custom validation rules for an attribute
func (v *SchemaValidator) validateCustomRules(fieldName string, value interface{}, validation map[string]interface{}) *ValidationError {
	for rule, ruleValue := range validation {
		switch rule {
		case "min":
			if minVal, ok := ruleValue.(float64); ok {
				if numVal, ok := v.toFloat64(value); ok && numVal < minVal {
					return &ValidationError{
						Field:   fieldName,
						Value:   value,
						Message: fmt.Sprintf("Value must be at least %f", minVal),
						Rule:    rule,
					}
				}
			}
		case "max":
			if maxVal, ok := ruleValue.(float64); ok {
				if numVal, ok := v.toFloat64(value); ok && numVal > maxVal {
					return &ValidationError{
						Field:   fieldName,
						Value:   value,
						Message: fmt.Sprintf("Value must be at most %f", maxVal),
						Rule:    rule,
					}
				}
			}
		case "minLength":
			if minLen, ok := ruleValue.(float64); ok {
				if strVal, ok := value.(string); ok && len(strVal) < int(minLen) {
					return &ValidationError{
						Field:   fieldName,
						Value:   value,
						Message: fmt.Sprintf("Value must be at least %d characters long", int(minLen)),
						Rule:    rule,
					}
				}
			}
		case "maxLength":
			if maxLen, ok := ruleValue.(float64); ok {
				if strVal, ok := value.(string); ok && len(strVal) > int(maxLen) {
					return &ValidationError{
						Field:   fieldName,
						Value:   value,
						Message: fmt.Sprintf("Value must be at most %d characters long", int(maxLen)),
						Rule:    rule,
					}
				}
			}
		case "pattern":
			if pattern, ok := ruleValue.(string); ok {
				if strVal, ok := value.(string); ok {
					re, err := regexp.Compile(pattern)
					if err != nil {
						return &ValidationError{
							Field:   fieldName,
							Value:   value,
							Message: fmt.Sprintf("Invalid pattern in validation rule: %s", pattern),
							Rule:    rule,
						}
					}
					if !re.MatchString(strVal) {
						return &ValidationError{
							Field:   fieldName,
							Value:   value,
							Message: fmt.Sprintf("Value does not match pattern: %s", pattern),
							Rule:    rule,
						}
					}
				}
			}
		case "format":
			if format, ok := ruleValue.(string); ok {
				if strVal, ok := value.(string); ok {
					switch format {
					case "email":
						emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
						if !emailRegex.MatchString(strVal) {
							return &ValidationError{
								Field:   fieldName,
								Value:   value,
								Message: "Value must be a valid email address",
								Rule:    rule,
							}
						}
					case "ipv4":
						ipv4Regex := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
						if !ipv4Regex.MatchString(strVal) {
							return &ValidationError{
								Field:   fieldName,
								Value:   value,
								Message: "Value must be a valid IPv4 address",
								Rule:    rule,
							}
						}
					case "url":
						urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
						if !urlRegex.MatchString(strVal) {
							return &ValidationError{
								Field:   fieldName,
								Value:   value,
								Message: "Value must be a valid URL",
								Rule:    rule,
							}
						}
					}
				}
			}
		case "enum":
			if enumValues, ok := ruleValue.([]interface{}); ok {
				valueFound := false
				for _, enumVal := range enumValues {
					if reflect.DeepEqual(value, enumVal) {
						valueFound = true
						break
					}
				}
				if !valueFound {
					return &ValidationError{
						Field:   fieldName,
						Value:   value,
						Message: fmt.Sprintf("Value must be one of: %v", enumValues),
						Rule:    rule,
					}
				}
			}
		}
	}

	return nil
}

// toFloat64 converts various numeric types to float64
func (v *SchemaValidator) toFloat64(value interface{}) (float64, bool) {
	switch val := value.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case string:
		if num, err := strconv.ParseFloat(val, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

// ApplyDefaults applies default values to CI attributes based on schema
func (v *SchemaValidator) ApplyDefaults(attributes map[string]interface{}, schema CITypeSchema) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy existing attributes
	for k, v := range attributes {
		result[k] = v
	}
	
	// Apply defaults for missing attributes
	for _, attrDef := range schema.Attributes {
		if _, exists := result[attrDef.Name]; !exists && attrDef.Default != nil {
			result[attrDef.Name] = attrDef.Default
		}
	}
	
	return result
}

// ValidateSchemaDefinition validates that a schema definition is valid
func (v *SchemaValidator) ValidateSchemaDefinition(schema CITypeSchema) ValidationResult {
	result := ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
	}

	// Validate schema name
	if strings.TrimSpace(schema.Name) == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Value:   schema.Name,
			Message: "Schema name cannot be empty",
		})
	}

	// Validate attributes
	attrNames := make(map[string]bool)
	for i, attr := range schema.Attributes {
		// Validate attribute name
		if strings.TrimSpace(attr.Name) == "" {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("attributes[%d].name", i),
				Value:   attr.Name,
				Message: "Attribute name cannot be empty",
			})
		}

		// Check for duplicate attribute names
		if attrNames[attr.Name] {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("attributes[%d].name", i),
				Value:   attr.Name,
				Message: fmt.Sprintf("Duplicate attribute name: %s", attr.Name),
			})
		}
		attrNames[attr.Name] = true

		// Validate attribute type
		validTypes := map[string]bool{
			AttributeTypeString:  true,
			AttributeTypeNumber:  true,
			AttributeTypeBoolean: true,
			AttributeTypeDate:    true,
			AttributeTypeArray:   true,
			AttributeTypeObject:  true,
		}

		if !validTypes[attr.Type] {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("attributes[%d].type", i),
				Value:   attr.Type,
				Message: fmt.Sprintf("Invalid attribute type: %s", attr.Type),
			})
		}

		// Validate default value type matches attribute type
		if attr.Default != nil {
			if err := v.validateType("default", attr.Default, attr.Type); err != nil {
				result.IsValid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("attributes[%d].default", i),
					Value:   attr.Default,
					Message: fmt.Sprintf("Default value type mismatch: %s", err.Message),
				})
			}
		}
	}

	return result
}
