package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	"golang.org/x/tools/imports"
)

var (
	out = flag.String("o", "models", "output directory, default is models in current directory")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("modeling_gin: ")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("set one yaml file name to arguments")
	}

	yaml := flag.Arg(0)
	yamlpath, err := filepath.Abs(yaml)
	if err != nil {
		log.Fatalf("get absolute path of yaml file: %s", err)
	}

	// Load swagger spec from yaml file.
	s, err := loadSwaggerSpec(yamlpath)
	if err != nil {
		log.Fatalf("load swagger spec from yaml file: %s", err)
	}

	var g generator
	// Print the header and package clause.
	g.printf("// Code is auto generated; DO NOT EDIT.\n")
	g.printf("package " + *out + "\n")

	// Generate models code.
	err = g.generateModels(s)
	if err != nil {
		log.Fatalf("generate models: %s", err)
	}

	// Format the output.
	src := g.format()

	// Create output directory.
	err = createOutDir(*out)
	if err != nil {
		log.Fatalf("creating output directory: %s", err)
	}

	outpath, err := filepath.Abs(filepath.Join(*out, "model_gen.go"))
	if err != nil {
		log.Fatalf("get absolute path of output file: %s", err)
	}

	err = ioutil.WriteFile(outpath, src, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}

	// Import package.
	imported, err := imports.Process(outpath, src, nil)
	if err != nil {
		log.Fatalf("import output file: %s", err)
	}

	// Replace output file.
	err = ioutil.WriteFile(outpath, imported, 0644)
	if err != nil {
		log.Fatalf("replacing output: %s", err)
	}
}

func loadSwaggerSpec(path string) (*spec.Swagger, error) {
	j, err := swag.YAMLDoc(path)
	if err != nil {
		return nil, err
	}

	var s spec.Swagger
	err = json.Unmarshal(j, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func createOutDir(dir string) error {
	err := os.Mkdir(dir, 0755)
	if !os.IsExist(err) {
		return err
	}
	return nil
}

type generator struct {
	buf           bytes.Buffer
	embededStruct []byte
}

// copyright: https://github.com/golang/tools/blob/master/cmd/stringer/stringer.go#L170
func (g *generator) printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// format returns the gofmt-ed contents of the Generator's buffer.
// copyright: https://github.com/golang/tools/blob/master/cmd/stringer/stringer.go#L342
func (g *generator) format() []byte {
	b := append(g.buf.Bytes(), g.embededStruct...)
	src, err := format.Source(b)
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		return b
	}
	return src
}

func (g *generator) generateModels(s *spec.Swagger) error {
	exist := make(map[string]bool)

	// Iterate definitions and print models.
	for name, s := range s.Definitions {
		if exist[name] {
			continue
		}
		exist[name] = true

		// Print comments.
		g.printComments(s)
		// Print struct.
		name = toUpperCaseFirstChar(name)
		g.printf("type %s struct {\n", name)
		err := g.generateCodes(name, s)
		if err != nil {
			return err
		}
		// Print struct close brace
		g.printf("}\n")
	}

	for name, s := range s.Responses {
		schema := s.Schema
		if schema == nil {
			continue
		}
		// Print comments.
		g.printf(formatComment(s.Description) + "\n")
		// Print struct.
		name = toUpperCaseFirstChar(name)
		g.printf("type %s struct {\n", name)
		if &schema.Ref != nil {
			g.printf("%s %s `json:\"response,omitempty\"`\n", name, "*"+g.extractReferenceName(schema.Ref))
		} else if schema != nil {
			err := g.generateProperties(name, *schema)
			if err != nil {
				return err
			}
		}
		g.printf("}\n")
	}

	// Interate paths and print both request and response.
	for _, s := range s.Paths.Paths {
		err := g.generateReqAndRes(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) printComments(s spec.Schema) {
	var comments []string
	if s.Title != "" {
		comment := formatComment(s.Title)
		comments = append(comments, comment)
	}
	if s.Description != "" {
		comment := formatComment(s.Description)
		comments = append(comments, comment)
	}
	g.printf("%s\n", strings.Join(comments, "\n//\n"))
}

func formatComment(comment string) string {
	descs := strings.Split(strings.TrimRight(comment, "\n"), "\n")
	for i, d := range descs {
		descs[i] = "// " + d
	}
	return strings.Join(descs, "\n")
}

func (g *generator) generateCodes(name string, s spec.Schema) error {
	allOf := s.AllOf
	if len(allOf) > 0 {
		for _, a := range allOf {
			// In all-of field, object type and reference are supported.
			if !a.Type.Contains("object") && a.Ref.String() == "" {
				return fmt.Errorf("object is only supported in all of properties, at %s definition", name)
			}
			refn := g.extractReferenceName(a.Ref)
			g.printf("%s\n", refn)

			// Generate properties in all-of field.
			err := g.generateProperties(name, a)
			if err != nil {
				return err
			}
		}
	}

	// If schema object type is not `object`, skip
	if !s.Type.Contains("object") {
		return nil
	}

	// Generate properties in definitions.
	err := g.generateProperties(name, s)
	if err != nil {
		return err
	}

	return nil
}

func (g *generator) generateProperties(name string, s spec.Schema) error {
	reqm := make(map[string]bool)
	for _, req := range s.Required {
		if !reqm[req] {
			reqm[req] = true
		}
	}

	// Iterate properties and print them.
	for pname, p := range s.Properties {
		var tags []string
		if reqm[pname] {
			tags = append(tags, "required")
		}

		var (
			typ     string
			newtags []string
			err     error
		)
		// If propetie is additional, generate from additonal properties field.
		// Otherwise, generate from properties field.
		if p.AdditionalProperties != nil && p.AdditionalProperties.Schema != nil {
			typ, newtags, err = g.extractTypeAndTagsFromPropertie(name, pname, *p.AdditionalProperties.Schema)
			if err != nil {
				return err
			}
			typ = "map[string]" + typ
		} else {
			typ, newtags, err = g.extractTypeAndTagsFromPropertie(name, pname, p)
			if err != nil {
				return err
			}
		}
		tags = append(tags, newtags...)
		u := toUpperCaseFirstChar(pname)
		g.printComments(p)
		format := "%s %s `json:\"%s,omitempty\" binding:\"%s\"`\n"
		if pname == "id" {
			format = "%s %s `json:\"%s,omitempty\" binding:\"%s\" goon:\"id\"`\n"
		}
		g.printf(format, u, typ, pname, strings.Join(tags, ","))
	}

	return nil
}

func toUpperCaseFirstChar(name string) string {
	if name == "" {
		return name
	}
	first := strings.ToUpper(name[0:1])
	return first + name[1:]
}

func (g *generator) extractReferenceName(ref spec.Ref) string {
	rs := ref.String()
	if rs == "" {
		return rs
	}
	li := strings.LastIndex(rs, "/")
	return toUpperCaseFirstChar(rs[li+1:])
}

func (g *generator) extractTypeAndTagsFromPropertie(parent, name string, p spec.Schema) (typ string, tags []string, err error) {
	if p.Ref.String() != "" {
		refn := g.extractReferenceName(p.Ref)
		return "*" + refn, nil, nil
	}

	if len(p.Type) != 1 {
		return "", nil, fmt.Errorf("propertie length must be one")
	}

	if len(p.Enum) > 0 {
		enumt := getEnumTag(p)
		tags = append(tags, enumt)
	}

	var newtags []string
	switch p.Type[0] {
	case "integer":
		typ, newtags = g.extractTypeAndTagsFromInteger(p)
	case "number":
		typ, newtags = g.extractTypeAndTagsFromNumber(p)
	case "string":
		typ, newtags = g.extractTypeAndTagsString(p)
	case "boolean":
		typ = "bool"
	case "array":
		typ, newtags, err = g.extractTypeAndTagsFromArray(p)
	case "object":
		// print embeded struct
		var eg generator
		name = parent + toUpperCaseFirstChar(name)
		eg.printf("type %s struct {\n", name)
		err = eg.generateCodes(name, p)
		if err != nil {
			return
		}
		eg.printf("}\n")
		g.embededStruct = append(g.embededStruct, eg.buf.Bytes()...)
		typ = "*" + name
	default:
		err = fmt.Errorf("propertie type [%s] is not supported", p.Type[0])
		return
	}

	tags = append(tags, newtags...)

	return
}

func getEnumTag(p spec.Schema) string {
	var enums []string
	for _, e := range p.Enum {
		enums = append(enums, fmt.Sprintf("%v", e))
	}

	return strings.Join(enums, "|")
}

func (g *generator) extractTypeAndTagsFromInteger(p spec.Schema) (typ string, tags []string) {
	switch p.Format {
	case "int64":
		typ = "int64"
	default:
		typ = "int"
	}

	tags = getMaxAndMinTags(p)
	return
}

func (g *generator) extractTypeAndTagsFromNumber(p spec.Schema) (typ string, tags []string) {
	typ = "float64"
	tags = getMaxAndMinTags(p)
	return
}

func (g *generator) extractTypeAndTagsString(p spec.Schema) (typ string, tags []string) {
	switch p.Format {
	case "byte":
		typ = "string"
		tags = append(tags, "base64")
	case "date", "date-time":
		typ = "time.Time"
	default:
		typ = "string"
	}

	if p.MaxLength != nil {
		tags = append(tags, "max="+fmt.Sprint(*p.MaxLength))
	}
	if p.MinLength != nil {
		tags = append(tags, "min="+fmt.Sprint(*p.MinLength))
	}
	return
}

func (g *generator) extractTypeAndTagsFromArray(p spec.Schema) (typ string, tags []string, err error) {
	if p.Items == nil || p.Items.Schema == nil {
		err = fmt.Errorf("array propertie must have items field and items schema")
		return
	}
	if p.MaxItems != nil {
		tags = append(tags, "max="+fmt.Sprint(*p.MaxItems))
	}
	if p.MinItems != nil {
		tags = append(tags, "min="+fmt.Sprint(*p.MaxItems))
	}

	itemp := p.Items.Schema
	itemtyp, itemtags, err := g.extractTypeAndTagsFromPropertie("", "", *itemp)
	if err != nil {
		return
	}

	typ = "[]" + itemtyp
	if len(itemtags) > 0 {
		tags = append(tags, "dive")
		tags = append(tags, itemtags...)
	}
	return
}

func getMaxAndMinTags(p spec.Schema) (tags []string) {
	if p.Maximum != nil {
		if p.ExclusiveMaximum {
			tags = append(tags, "lt="+fmt.Sprint(*p.Maximum))
		} else {
			tags = append(tags, "max="+fmt.Sprint(*p.Maximum))
		}
	}
	if p.Minimum != nil {
		if p.ExclusiveMinimum {
			tags = append(tags, "gt="+fmt.Sprint(*p.Minimum))
		} else {
			tags = append(tags, "min="+fmt.Sprint(*p.Minimum))
		}
	}
	return
}

func (g *generator) generateReqAndRes(item spec.PathItem) (err error) {
	get := item.Get
	if get != nil {
		err = g.printReqAndRes(get)
		if err != nil {
			return err
		}
	}
	post := item.Post
	if post != nil {
		err = g.printReqAndRes(post)
		if err != nil {
			return err
		}
	}
	put := item.Put
	if put != nil {
		err = g.printReqAndRes(put)
		if err != nil {
			return err
		}
	}
	delete := item.Delete
	if delete != nil {
		err = g.printReqAndRes(delete)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) printReqAndRes(op *spec.Operation) (err error) {
	id := op.ID
	if id == "" {
		return errors.New("operation id must be not empty")
	}
	err = g.printRequest(id, op.OperationProps)
	if err != nil {
		return err
	}
	err = g.printResponse(id, op.OperationProps)
	if err != nil {
		return err
	}
	return nil
}

func (g *generator) printRequest(id string, op spec.OperationProps) (err error) {
	for _, p := range op.Parameters {
		if p.Name != "body" || p.Schema == nil {
			continue
		}
		name := toUpperCaseFirstChar(id + "Request")
		g.printf(formatComment(p.Description) + "\n")
		g.printf("type %s struct {\n", name)
		if p.Schema != nil {
			ref := p.Schema.Ref
			if &ref != nil && ref.String() != "" {
				g.printf("Request %s `json:\"request,omitempty\"`\n", "*"+g.extractReferenceName(p.Schema.Ref))
			} else {
				err = g.generateProperties(name, *p.Schema)
				if err != nil {
					return err
				}
			}
		}
		g.printf("}\n")
	}
	return nil
}

func (g *generator) printResponse(id string, op spec.OperationProps) error {
	res := op.Responses
	if res == nil {
		return nil
	}
	def := res.Default
	if def != nil {
		ref := def.Ref
		schema := def.Schema
		if &ref == nil && schema == nil {
			return nil
		}
		name := toUpperCaseFirstChar(id + "DefaultResponse")
		g.printf(formatComment(def.Description) + "\n")
		g.printf("type %s struct {\n", name)
		if &ref != nil && ref.String() != "" {
			g.printf("Response %s `json:\"response,omitempty\"`\n", "*"+g.extractReferenceName(ref))
		} else if schema != nil {
			err := g.generateProperties(name, *schema)
			if err != nil {
				return err
			}
		}
		g.printf("}\n")
	}
	m := res.StatusCodeResponses
	for code, res := range m {
		schema := res.Schema
		if schema == nil {
			return nil
		}
		name := toUpperCaseFirstChar(id + strconv.Itoa(code) + "Response")
		g.printf(formatComment(res.Description) + "\n")
		g.printf("type %s struct {\n", name)
		ref := schema.Ref
		if &ref != nil && ref.String() != "" {
			g.printf("Response %s `json:\"response,omitempty\"`\n", "*"+g.extractReferenceName(schema.Ref))
		} else if schema != nil {
			format := "Response %s `json:\"%s,omitempty\" binding:\"%s\"`\n"
			typ, tags, err := g.extractTypeAndTagsFromPropertie("", name, *schema)
			if err != nil {
				return err
			}
			g.printf(format, typ, "response", strings.Join(tags, ","))
		}
		g.printf("}\n")
	}
	return nil
}
