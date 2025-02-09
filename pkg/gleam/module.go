package gleam

import (
	"strings"
	"text/template"

	"github.com/bwireman/gleam_pb/pkg/gleam/fields"
	pgs "github.com/lyft/protoc-gen-star"
)

type GleamModule struct {
	*pgs.ModuleBase
	tpl     *template.Template
	wrapper *gpbWrapper

	gpbHeaderInclude string
	output           string
}

func Gleam() *GleamModule { return &GleamModule{ModuleBase: &pgs.ModuleBase{}} }

func (g *GleamModule) InitContext(c pgs.BuildContext) {
	g.ModuleBase.InitContext(c)
	g.tpl = template.Must(template.New("gleam-package-template").Parse(fields.GleamTemplate))

	if g.output = g.Parameters().Str("output_path"); g.output == "" {
		g.Fail("please specify the `output_path` flag")
	}

	protocErlPath := g.Parameters().StrDefault("protoc_erl_path", "./deps/gpb/bin/protoc-erl")
	if wrapper, err := newGPBWrapper(protocErlPath, g.OutputPath()); err != nil {
		g.Fail(err.Error())
	} else {
		g.wrapper = wrapper
	}

	g.gpbHeaderInclude = g.Parameters().Str("gpb_header_include")
}

func (g *GleamModule) OutputPath() string { return g.output }

func (g *GleamModule) Name() string { return "gleam" }

func (g *GleamModule) Execute(targets map[string]pgs.File, pkgs map[string]pgs.Package) []pgs.Artifact {
	protosFilePaths := []string{}
	for t, f := range targets {
		if f.Package().ProtoName() != "google.protobuf" {
			protosFilePaths = append(protosFilePaths, t)
		}
	}

	if err := g.wrapper.generate(protosFilePaths); err != nil {
		g.Fail(err.Error())
	} else if g.gpbHeaderInclude != "" {
		if err = g.wrapper.updateImport(g.gpbHeaderInclude); err != nil {
			g.Fail(err.Error())
		}
	}

	g.AddCustomFile(g.OutputPath()+"/gleam_pb.gleam", fields.GleamPB, 0644)

	for _, p := range pkgs {
		allMessages := []pgs.Message{}
		allEnums := []pgs.Enum{}
		imports := []string{"gleam/option", "gleam/list", "gleam/pair", "gleam/dynamic", "gleam/erlang/atom", "gleam_pb"}

		for _, file := range p.Files() {
			allMessages = append(allMessages, file.AllMessages()...)
			allEnums = append(allEnums, file.AllEnums()...)

			for _, imp := range file.Imports() {
				if imp.Package().ProtoName() != p.ProtoName() {
					new_import := strings.ReplaceAll(
                                          imp.Package().ProtoName().LowerSnakeCase().String(), ".", "/")
if (new_import != "") {
					imports = append(imports, new_import)
                                      }
				}

			}
		}
		g.generate(allMessages, allEnums, p, imports)
	}

	return g.Artifacts()
}

func (g *GleamModule) generate(all_messages []pgs.Message, all_enums []pgs.Enum, pkg pgs.Package, imports []string) {
	gleam_types_map := []map[string]interface{}{}
	enc_dec := []map[string]interface{}{}
	generators := []map[string]interface{}{}
	enums := []map[string]interface{}{}
	printers := []map[string]interface{}{}
//{{ range .printers }}
//fn show_{{ .type_name }}(a) -> String {
//{{ range .params}}  {{.printer_name}}(a.{{.name}}),  {{ end }}
//}
//{{ end }}


	for _, enum := range all_enums {
		// don't need enum generators
		enum_gleam_type := fields.GleamTypeFromEnum(enum)
		gleam_types_map = append(gleam_types_map, enum_gleam_type.RenderAsMap())
		enc_dec = append(enc_dec, fields.GenEncDecFromEnum(enum, enum_gleam_type).RenderAsMap())

                enums = append(enums, fields.GenPrinterFromEnum(enum, enum_gleam_type));
	}

	for _, msg := range all_messages {
		for _, oo := range msg.OneOfs() {
			// don't need oneof generators
			oo_gleam_type := fields.GleamTypeFromOnoeOf(msg, oo)
			oo_gleam_type_map := oo_gleam_type.RenderAsMap()
			gleam_types_map = append(gleam_types_map, oo_gleam_type_map)
			enc_dec = append(enc_dec, fields.GenEncDecFromOneOf(msg, oo, oo_gleam_type).RenderAsMap())
		}

		msg_gleam_type := fields.GleamTypeFromMessage(msg)


                if generator := fields.GeneratorFnFromGleamType(msg_gleam_type); generator != nil {
			generators = append(generators, generator.RenderAsMap())
		}

		msg_gleam_type_map := msg_gleam_type.RenderAsMap()
		gleam_types_map = append(gleam_types_map, msg_gleam_type_map)
		enc_dec = append(enc_dec, fields.GenEncDecFromMessage(msg, msg_gleam_type).RenderAsMap())
                printers = append(printers, fields.GenPrinterFromMessage(msg, msg_gleam_type))
	}


	g.AddGeneratorTemplateFile(strings.Replace(pkg.ProtoName().LowerSnakeCase().String(), ".", "/", -1)+".gleam", g.tpl, map[string]interface{}{
		"imports":    imports,
		"package":    pkg.ProtoName().LowerSnakeCase().String(),
		"messages":   gleam_types_map,
		"generators": generators,
		"enc_dec":    enc_dec,
                "printers":   printers,
                "enums":      enums,
	})
}
