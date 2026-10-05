// Package tokens reads a design system's tokens.json and compiles it to tokens.css.
package tokens

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Doc is a parsed tokens.json.
type Doc struct {
	Name    string
	Themes  []Theme
	Colors  []Token
	Shadows []Token
	Spacing []Token
	Radius  []Token
	// Others are any further {"tokens": [...]} families (opacity, zIndex, ...), sorted by key.
	Others []Family
	Type   Type
}

// Theme is one entry of color.themes.
type Theme struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Token is a named value. Value is a string, a number, or a map of theme id to string.
type Token struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
	Usage string `json:"usage"`
}

// Family is a named list of tokens.
type Family struct {
	Key    string
	Tokens []Token
}

// Type is the type section of tokens.json.
type Type struct {
	Fonts    []Font            `json:"fonts"`
	Families map[string]string `json:"families"`
	Groups   []TypeGroup       `json:"groups"`
	// FamilyOrder is the key order of Families in the source file.
	FamilyOrder []string `json:"-"`
}

// Font is one @font-face source.
type Font struct {
	Family string `json:"family"`
	File   string `json:"file"`
	Weight string `json:"weight"`
	Style  string `json:"style"`
}

// TypeGroup is a named set of type styles sharing a default family key.
type TypeGroup struct {
	Name   string      `json:"name"`
	Family string      `json:"family"`
	Styles []TypeStyle `json:"styles"`
}

// TypeStyle is one text style. Numeric fields accept strings or numbers.
type TypeStyle struct {
	Name          string `json:"name"`
	Family        string `json:"family"`
	FontSize      any    `json:"fontSize"`
	LineHeight    any    `json:"lineHeight"`
	FontWeight    any    `json:"fontWeight"`
	FontStyle     string `json:"fontStyle"`
	LetterSpacing any    `json:"letterSpacing"`
	OpticalSize   any    `json:"opticalSize"`
	Usage         string `json:"usage"`
	Sample        string `json:"sample"`
}

var knownKeys = map[string]bool{
	"name": true, "version": true, "layout": true, "meta": true,
	"color": true, "type": true, "spacing": true, "radius": true, "shadow": true,
}

// Parse reads tokens.json content.
func Parse(data []byte) (*Doc, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("tokens.json: %w", err)
	}
	d := &Doc{}
	if v, ok := raw["name"]; ok {
		_ = json.Unmarshal(v, &d.Name)
	}
	if v, ok := raw["color"]; ok {
		var c struct {
			Themes []Theme `json:"themes"`
			Tokens []Token `json:"tokens"`
		}
		if err := json.Unmarshal(v, &c); err != nil {
			return nil, fmt.Errorf("tokens.json color: %w", err)
		}
		d.Themes, d.Colors = c.Themes, c.Tokens
	}
	var err error
	if d.Shadows, err = family(raw, "shadow"); err != nil {
		return nil, err
	}
	if d.Spacing, err = family(raw, "spacing"); err != nil {
		return nil, err
	}
	if d.Radius, err = family(raw, "radius"); err != nil {
		return nil, err
	}
	var keys []string
	for k := range raw {
		if !knownKeys[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		toks, err := family(raw, k)
		if err != nil || toks == nil {
			continue // not a {"tokens": [...]} family: ignored, as the platform does
		}
		d.Others = append(d.Others, Family{Key: k, Tokens: toks})
	}
	if v, ok := raw["type"]; ok {
		if err := json.Unmarshal(v, &d.Type); err != nil {
			return nil, fmt.Errorf("tokens.json type: %w", err)
		}
		d.Type.FamilyOrder = objectKeys(v, "families")
	}
	return d, nil
}

func family(raw map[string]json.RawMessage, key string) ([]Token, error) {
	v, ok := raw[key]
	if !ok {
		return nil, nil
	}
	var f struct {
		Tokens []Token `json:"tokens"`
	}
	if err := json.Unmarshal(v, &f); err != nil {
		return nil, fmt.Errorf("tokens.json %s: %w", key, err)
	}
	return f.Tokens, nil
}

// objectKeys returns the keys of obj[field] in source order.
func objectKeys(obj json.RawMessage, field string) []string {
	var outer map[string]json.RawMessage
	if json.Unmarshal(obj, &outer) != nil {
		return nil
	}
	dec := json.NewDecoder(bytesReader(outer[field]))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil
	}
	var keys []string
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			break
		}
		keys = append(keys, k.(string))
		var skip json.RawMessage
		if dec.Decode(&skip) != nil {
			break
		}
	}
	return keys
}
