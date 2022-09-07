package fields

import (
	pgs "github.com/lyft/protoc-gen-star"
)

type GleamPrimitive int

type GleamPrimitiveOrValue struct {
	Primitive GleamPrimitive
	Value     string
}

const (
	Unknown GleamPrimitive = iota
	Int
	Float
	String
	List
	Map
	Option
	Bool
	BitString
)

var ProtoTypeToPrimitives = map[pgs.ProtoType]GleamPrimitive{
	pgs.DoubleT:  Float,
	pgs.FloatT:   Float,
	pgs.Int64T:   Int,
	pgs.UInt64T:  Int,
	pgs.Int32T:   Int,
	pgs.Fixed64T: Int,
	pgs.Fixed32T: Int,
	pgs.UInt32T:  Int,
	pgs.SFixed32: Int,
	pgs.SFixed64: Int,
	pgs.SInt64:   Int,
	pgs.SInt32:   Int,
	pgs.StringT:  String,
	pgs.BoolT:    Bool,
	pgs.BytesT:   BitString,
}

var GleamPrimitiveDefaultValues = map[GleamPrimitive]string{
	Int:       "0",
	Float:     "0.0",
	String:    "\"\"",
	List:      "list.new()",
	Map:       "list.new()",
	Option:    "option.None",
	Bool:      "False",
	BitString: "<<>>",
}

// template for generated modules
const GleamTemplate = `
{{ range .imports }}
import {{ . }}
{{ end }}
import gleam/string
import gleam/string_builder


/// {{ .package }} package types generated by gleam_pb
/// DO NOT EDIT

{{ range .messages }}
pub type {{ .type_name }} {
  {{ range .constructors -}}
    {{ . }}
  {{ end }}
}
{{ end }}


// printers
external fn io_lib_format(a : String, b) -> String =
    "io_lib" "format"

fn primitive_show_int(a) -> String {
  //io_lib_format("~B", [a])
  string.inspect(a)
}

fn primitive_show_string(a) -> String {
  string.append("\"", string.append(a , "\""))
}

fn primitive_show_bool(a) -> String {
  string.inspect( a) 
}

fn primitive_show_float(a : Float) -> String {
  string.inspect(a) 
}

fn primitive_show_bit_string(a) -> String {
  string.inspect(a)
}            

{{ range .printers }}
pub fn show_{{ .lowercase_type_name }}(a : {{.type_name}}) -> List(String) {
[
"{{.foreign_type_name}}(",
{{ range .params}}  {{.printer}}, 
{{ end }}
")"
]
}
{{ end }}


{{ range .enums }}
pub fn show_{{ .type_name }}(a) -> String {
  case a {
{{ range .constructors }}    {{.name}} -> "{{.pkg}}.{{.name}}" 
{{ end }}
  }
}
{{ end }}


// generators

{{ range .generators }}
pub fn {{ .func_name }}() {
{{ if .has_fields }}
	{{ .type_name }}({{ .fields }})
{{else}}
	{{ .type_name }}
{{end}}
}
{{ end }}

{{ range .enc_dec }}

// --- {{ .message_name }} ---

{{ if .is_enum }}
pub fn extract_{{ .func_name }}(e: {{ .type_name }}) -> atom.Atom {
	case e {
	  {{ range .extract_patterns -}}
		{{ . }} 
	  {{ end }}
	}
}

pub fn reconstruct_{{ .func_name }}(e: atom.Atom) -> {{ .type_name }} {
	{{ range .reconstruct_vars -}}
		{{ . }}
	{{ end }}

	case e {
	  {{ range .reconstruct_patterns -}}
		{{ . }}
	  {{ end }}
	}
}

{{ else if .is_oneof }}

pub fn extract_{{ .func_name }}(m: {{ .type_name }}) -> dynamic.Dynamic {
	case m {
		{{ range .extract_patterns -}}
			{{ . }} |> dynamic.from
		{{ end }}
	}
}


pub fn reconstruct_{{ .func_name }}(u: gleam_pb.Undefined(#(atom.Atom, x))) -> option.Option({{ .type_name }}) {
	{{ range .reconstruct_vars -}}
		{{ . }}
	{{ end }}

	
	case u {
		gleam_pb.Undefined -> option.None
		gleam_pb.Wrapper(m)	-> 
                 {
                   case m {
			{{ range .reconstruct_patterns -}}
			{{ . }}
			{{ end }}
		} |> option.Some
               }
                m -> {
                case gleam_pb.force_a_to_b(m) {
	   {{ range .reconstruct_patterns -}}
	      {{ . }}
	   {{ end }}
         } |> option.Some
       }
	}
}

{{ else }}
pub fn extract_{{ .func_name }}(reserved__struct_name: atom.Atom, m: {{ .type_name }}) -> dynamic.Dynamic {
	case m {
		{{ range .extract_patterns -}}
			{{ . }} |> dynamic.from
		{{ end }}
	}
}

pub fn reconstruct_{{ .func_name }}(m: #({{ .reconstruct_type }})) -> {{ .type_name }} {
	case m {
		{{ range .reconstruct_patterns -}}
			{{ . }}
		{{ end }}
	}
}

pub fn encode_{{ .func_name }}(m: {{ .type_name }}) -> BitString {
	let name = atom.create_from_string("{{ .message_name }}")

	extract_{{ .func_name }}(name, m) 
	|> gleam_pb.encode(name)
}

external fn decode_msg_{{ .func_name }}(BitString, atom.Atom) -> #({{ .reconstruct_type }}) =
  "gleam_gpb" "decode_msg"

pub fn decode_{{ .func_name }}(b: BitString) -> {{ .type_name }} {
	let name = atom.create_from_string("{{ .message_name }}")
	decode_msg_{{ .func_name }}(b, name)
	|> reconstruct_{{ .func_name }}
}
{{ end }}
{{ end }}
`

// included helper functions and types
const GleamPB = `import gleam/list
import gleam/dynamic
import gleam/option
import gleam/erlang/atom

// helper funcs and types
pub type Undefined(a) {
	Undefined
	Wrapper(a)
}

pub fn encode(m: dynamic.Dynamic, name: atom.Atom) -> BitString {
  encode_msg(m, name, [True])
}

pub fn option_to_gpb(o: option.Option(a)) -> dynamic.Dynamic {
	case o {
		option.Some(v) -> v |> dynamic.from
		option.None -> Undefined |> dynamic.from
	} 
}

pub fn wrapper_to_option(w: Undefined(a)) -> option.Option(a) {
	case w {
		Undefined -> option.None
		Wrapper(v) -> v |> option.Some
	}
}

pub fn force_a_to_b(a: a) -> b {
	a |> dynamic.from |> dynamic.unsafe_coerce 
}

external fn encode_msg(dynamic.Dynamic, atom.Atom, List(Bool)) -> BitString =
  "gleam_gpb" "encode_msg"
`
