package common

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	variable "github.com/fsncps/hyperfile/src/config"
	"github.com/pelletier/go-toml/v2"
)

func LoadHotkeyDisplayGroups() {
	content, err := os.ReadFile(variable.HotkeysFile)
	if err != nil {
		content = []byte(HotkeysTomlString)
	}

	groups, parseErr := ParseHotkeyDisplayGroups(content, Hotkeys)
	if parseErr != nil {
		fallbackGroups, fallbackErr := ParseHotkeyDisplayGroups([]byte(HotkeysTomlString), Hotkeys)
		if fallbackErr != nil {
			HotkeyDisplayGroups = nil
			return
		}
		HotkeyDisplayGroups = fallbackGroups
		return
	}
	HotkeyDisplayGroups = groups
}

func ParseHotkeyDisplayGroups(content []byte, loadedHotkeys HotkeysType) ([]HotkeyDisplayGroup, error) {
	var raw map[string]interface{}
	if err := toml.Unmarshal(content, &raw); err != nil {
		return nil, err
	}

	headers := map[string]string{}
	if headersRaw, ok := raw["headers"]; ok {
		if headersMap, okMap := headersRaw.(map[string]interface{}); okMap {
			for k, v := range headersMap {
				headers[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	contextOrder := extractContextOrder(raw)
	groupsByContext := make(map[string][]HotkeyDisplayEntry)

	keyMap := buildHotkeyFieldMap(loadedHotkeys)

	for key, value := range raw {
		if key == "headers" {
			continue
		}
		table, isTable := value.(map[string]interface{})
		if !isTable {
			continue
		}

		for id, entryValue := range table {
			meta := parseHotkeyMeta(entryValue)
			boundKeys, found := keyMap[id]
			if !found || len(boundKeys) == 0 || boundKeys[0] == "" {
				continue
			}

			item := HotkeyDisplayEntry{
				ID:          id,
				Context:     key,
				Key:         boundKeys[0],
				Name:        meta.name,
				Description: meta.description,
				Icon:        meta.icon,
			}
			if item.Name == "" {
				item.Name = prettifyID(id)
			}
			item.Name = truncateTextByRuneCount(item.Name, maxHotkeyDisplayNameLength)
			if item.Description == "" {
				item.Description = item.Name
			}
			groupsByContext[key] = append(groupsByContext[key], item)
		}
	}

	groups := make([]HotkeyDisplayGroup, 0, len(groupsByContext))
	seen := make(map[string]bool)
	for _, context := range contextOrder {
		entries, ok := groupsByContext[context]
		if !ok {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
		title := headers[context]
		if title == "" {
			title = prettifyID(context)
		}
		groups = append(groups, HotkeyDisplayGroup{Context: context, Title: title, Items: entries})
		seen[context] = true
	}

	for context, entries := range groupsByContext {
		if seen[context] {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
		title := headers[context]
		if title == "" {
			title = prettifyID(context)
		}
		groups = append(groups, HotkeyDisplayGroup{Context: context, Title: title, Items: entries})
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no grouped hotkey metadata found")
	}

	return groups, nil
}

const maxHotkeyDisplayNameLength = 16

type hotkeyMeta struct {
	name        string
	description string
	icon        string
}

func parseHotkeyMeta(v interface{}) hotkeyMeta {
	arr, ok := v.([]interface{})
	if !ok {
		if strArr, okStr := v.([]string); okStr {
			arr = make([]interface{}, 0, len(strArr))
			for _, s := range strArr {
				arr = append(arr, s)
			}
		} else {
			return hotkeyMeta{}
		}
	}

	meta := hotkeyMeta{}
	if len(arr) > 1 {
		meta.name = strings.TrimSpace(fmt.Sprintf("%v", arr[1]))
		meta.name = truncateTextByRuneCount(meta.name, maxHotkeyDisplayNameLength)
	}
	if len(arr) > 2 {
		meta.description = strings.TrimSpace(fmt.Sprintf("%v", arr[2]))
	}
	if len(arr) > 3 {
		meta.icon = strings.TrimSpace(fmt.Sprintf("%v", arr[3]))
	}
	return meta
}

func truncateTextByRuneCount(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

func buildHotkeyFieldMap(h HotkeysType) map[string][]string {
	m := make(map[string][]string)
	v := reflect.ValueOf(h)
	t := reflect.TypeOf(h)
	for i := range v.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("toml")
		if tag == "" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		hotkeys, ok := v.Field(i).Interface().([]string)
		if !ok {
			continue
		}
		m[name] = hotkeys
	}
	return m
}

func extractContextOrder(raw map[string]interface{}) []string {
	order := make([]string, 0, len(raw))
	for key, value := range raw {
		if key == "headers" {
			continue
		}
		if _, ok := value.(map[string]interface{}); ok {
			order = append(order, key)
		}
	}
	return order
}

func prettifyID(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
