package main

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// OrderedMap preserves the key order from YAML unmarshaling
type OrderedMap struct {
	Keys   []string
	Values map[string]interface{}
}

// UnmarshalYAML preserves key order during unmarshaling, recursively for nested maps
func (o *OrderedMap) UnmarshalYAML(value *yaml.Node) error {
	o.Values = make(map[string]interface{})
	o.Keys = []string{}
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		var val interface{}
		node := value.Content[i+1]

		// If this is a nested map, decode it as OrderedMap to preserve order
		if node.Kind == yaml.MappingNode {
			var orderedVal OrderedMap
			if err := node.Decode(&orderedVal); err != nil {
				return err
			}
			val = orderedVal
		} else if node.Kind == yaml.SequenceNode {
			// Handle sequences (arrays) - check if they contain maps
			val = decodeSequence(node)
		} else {
			// For scalar values, decode normally
			if err := node.Decode(&val); err != nil {
				return err
			}
		}
		o.Keys = append(o.Keys, key)
		o.Values[key] = val
	}
	return nil
}

// decodeSequence recursively decodes sequences, preserving order in nested maps
func decodeSequence(node *yaml.Node) []interface{} {
	result := make([]interface{}, 0, len(node.Content))
	for _, item := range node.Content {
		if item.Kind == yaml.MappingNode {
			var orderedMap OrderedMap
			if err := item.Decode(&orderedMap); err == nil {
				result = append(result, orderedMap)
			}
		} else if item.Kind == yaml.SequenceNode {
			result = append(result, decodeSequence(item))
		} else {
			var val interface{}
			if err := item.Decode(&val); err == nil {
				result = append(result, val)
			}
		}
	}
	return result
}

// MarshalYAML preserves key order during marshaling
func (o OrderedMap) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, key := range o.Keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
		var valueNode yaml.Node
		if err := valueNode.Encode(o.Values[key]); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, keyNode, &valueNode)
	}
	return node, nil
}

// sortKeysInOriginalOrder sorts keys based on their original order
func sortKeysInOriginalOrder(keys []string, originalOrder []string) []string {
	orderMap := make(map[string]int)
	for i, key := range originalOrder {
		orderMap[key] = i
	}
	sort.Slice(keys, func(i, j int) bool {
		return orderMap[keys[i]] < orderMap[keys[j]]
	})
	return keys
}
