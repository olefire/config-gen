package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"
)

type FieldSpec struct {
	Type        string `yaml:"type"`
	Default     any    `yaml:"default"`
	Description string `yaml:"description"`
}

type ConfigTemplate struct {
	PackageName string
	StructName  string
	Fields      []Field
	UsesTime    bool
}

type Field struct {
	OriginalName string
	Name         string
	VarName      string
	Type         string
	Default      string
	Description  string
}

func main() {
	schemaPath := flag.String("schema", "config_schema.yaml", "Path to config schema YAML file")
	flag.Parse()

	yamlData, err := os.ReadFile(*schemaPath)
	if err != nil {
		log.Fatalf("cannot read schema file %q: %v", *schemaPath, err)
	}

	var config map[string]FieldSpec
	if err = yaml.Unmarshal(yamlData, &config); err != nil {
		log.Fatal(err)
	}

	fields, usesTime := parseFields(config)

	tmplData := ConfigTemplate{
		PackageName: "config",
		StructName:  "AppConfig",
		Fields:      fields,
		UsesTime:    usesTime,
	}

	writeConfig(tmplData)
	writeFake(tmplData)

	fmt.Println("✅ Конфигурационный код сгенерирован в", "internal/config")
}

func parseFields(config map[string]FieldSpec) ([]Field, bool) {
	var fields []Field
	usesTime := false

	for name, spec := range config {
		goType := spec.Type
		if goType == "duration" {
			goType = "time.Duration"
			usesTime = true
		}

		field := Field{
			OriginalName: name,
			Name:         toCamel(name),
			VarName:      toCamel(name),
			Type:         goType,
			Default:      formatDefaultValue(spec.Default, spec.Type),
			Description:  spec.Description,
		}
		fields = append(fields, field)
	}
	return fields, usesTime
}

func formatDefaultValue(val any, specType string) string {
	if specType == "duration" {
		if str, ok := val.(string); ok {
			str = normalizeDuration(str)
			if d, err := time.ParseDuration(str); err == nil {
				return fmt.Sprintf("%d * time.Nanosecond", d.Nanoseconds())
			} else {
				log.Fatalf("invalid duration string %q: %v", str, err)
			}
		}
	}

	switch v := val.(type) {
	case string:
		if str, ok := val.(string); ok {
			if looksLikeDuration(str) {
				return fmt.Sprintf(`func() time.Duration {
				d, _ := time.ParseDuration("%s")
				return d
			}()`, str)
			}
		}

		return fmt.Sprintf("%q", v)
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", v)

	case []any:
		if len(v) == 0 {
			return "[]any{}"
		}
		elemType := detectType(v[0])
		elems := make([]string, len(v))
		for i, e := range v {
			elems[i] = formatDefaultValue(e, detectType(e))
		}
		return fmt.Sprintf("[]%s{%s}", elemType, strings.Join(elems, ", "))

	case map[any]any:
		if len(v) == 0 {
			return "map[any]any{}"
		}

		allStructEmpty := true
		allKeysAreStrings := true
		for k, val := range v {
			if _, ok := k.(string); !ok {
				allKeysAreStrings = false
			}
			if _, ok := val.(map[any]any); !ok || len(val.(map[any]any)) > 0 {
				allStructEmpty = false
			}
		}
		if allKeysAreStrings && allStructEmpty {
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, fmt.Sprintf("%q: struct{}{}", k))
			}
			return fmt.Sprintf("map[string]struct{}{%s}", strings.Join(keys, ", "))
		}

		keyType := detectType(getFirstMapKey(v))
		valType := detectType(getFirstMapValue(v))

		entries := make([]string, 0, len(v))
		for k, val := range v {
			entries = append(entries, fmt.Sprintf("%s: %s", formatDefaultValue(k, detectType(k)), formatDefaultValue(val, detectType(val))))
		}
		return fmt.Sprintf("map[%s]%s{%s}", keyType, valType, strings.Join(entries, ", "))

	default:
		return fmt.Sprintf("%#v", val)
	}
}

func detectType(val any) string {
	switch val.(type) {
	case string:
		return "string"
	case int, int64:
		return "int"
	case float64:
		return "float64"
	case bool:
		return "bool"
	case map[any]any:
		return "map[string]any"
	case []any:
		return "[]any"
	default:
		return "any"
	}
}

func getFirstMapKey(m map[any]any) any {
	for k := range m {
		return k
	}
	return nil
}

func getFirstMapValue(m map[any]any) any {
	for _, v := range m {
		return v
	}
	return nil
}

func toCamel(input string) string {
	parts := strings.Split(input, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

func toLowerCamel(input string) string {
	camel := toCamel(input)
	return strings.ToLower(camel[:1]) + camel[1:]
}

func looksLikeDuration(s string) bool {
	_, err := time.ParseDuration(s)
	return err == nil
}

func normalizeDuration(s string) string {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasSuffix(s, "d"):
		num := strings.TrimSuffix(s, "d")
		n, err := strconv.Atoi(num)
		if err != nil {
			log.Fatalf("invalid number in duration %q: %v", s, err)
		}
		return fmt.Sprintf("%dh", n*24)

	case strings.HasSuffix(s, "w"):
		num := strings.TrimSuffix(s, "w")
		n, err := strconv.Atoi(num)
		if err != nil {
			log.Fatalf("invalid number in duration %q: %v", s, err)
		}
		return fmt.Sprintf("%dh", n*24*7)

	case strings.HasSuffix(s, "ms"), strings.HasSuffix(s, "s"), strings.HasSuffix(s, "m"), strings.HasSuffix(s, "h"):
		return s

	default:
		log.Fatalf("unsupported duration suffix in %q — allowed: ms, s, m, h, d, w", s)
		return ""
	}
}

func writeConfig(tmplData ConfigTemplate) {
	tmpl := template.Must(template.New("config").Funcs(template.FuncMap{
		"toCamel":      toCamel,
		"toLowerCamel": toLowerCamel,
	}).Parse(configTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplData); err != nil {
		log.Fatal(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("internal/config", 0755); err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("internal/config/config.go", formatted, 0644); err != nil {
		log.Fatal(err)
	}
}

func writeFake(tmplData ConfigTemplate) {
	fakeBuf := bytes.Buffer{}
	fakeTmpl := template.Must(template.New("fake").Funcs(template.FuncMap{
		"toCamel": toCamel,
	}).Parse(fakeTemplate))

	if err := fakeTmpl.Execute(&fakeBuf, tmplData); err != nil {
		log.Fatal(err)
	}

	formattedFake, err := format.Source(fakeBuf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("internal/config/fake.go", formattedFake, 0644); err != nil {
		log.Fatal(err)
	}
}

const configTemplate = `// Code generated by config-gen; DO NOT EDIT.
package {{.PackageName}}

import (
    "context"
    {{- if .UsesTime }}
    "time"
    {{- end }}

    clientv3 "go.etcd.io/etcd/client/v3"
    konfig "github.com/olefire/realtime-config-go"
)

type AppConfig interface {
{{- range .Fields}}
    // Get{{.Name}} возвращает значение {{.OriginalName}}. {{.Description}}
    Get{{.Name}}() {{.Type}}
{{- end}}
}

type appConfig struct {
{{- range .Fields}}
    {{.Name}} {{.Type}} ` + "`etcd:\"{{.OriginalName}}\"`" + `
{{- end}}
}

func NewAppConfig(ctx context.Context, cli *clientv3.Client, prefix string) (*konfig.RealTimeConfig, AppConfig, error) {
    cfg := &appConfig{
{{- range .Fields}}
        {{.Name}}: {{.Default}},
{{- end}}
    }
    rtc, err := konfig.NewRealTimeConfig(ctx, cli, prefix, cfg)
    return rtc, cfg, err
}

{{range .Fields}}
// Get{{.Name}} возвращает значение {{.OriginalName}}. {{.Description}}
func (c *appConfig) Get{{.Name}}() {{.Type}} {
    return c.{{.Name}}
}
{{end}}
`

const fakeTemplate = `// Code generated by config-gen; DO NOT EDIT.
package {{.PackageName}}

	{{- if .UsesTime }}
	import (
	"time"
	)
	{{- end }}

type FakeAppConfig struct {
	{{- range .Fields}}
	{{.VarName}} {{.Type}}
	{{- end}}
}

{{range .Fields}}
// Get{{.Name}} возвращает значение {{.OriginalName}}. {{.Description}}
func (f *FakeAppConfig) Get{{.Name}}() {{.Type}} {
	return f.{{.VarName}}
}
{{end}}
`
