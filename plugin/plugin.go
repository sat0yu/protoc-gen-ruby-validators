package plugin

import (
    "bytes"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "strings"

    . "github.com/mwitkow/go-proto-validators"
    proto "github.com/golang/protobuf/proto"
    descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
    plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func ParseRequest(r io.Reader) (*plugin.CodeGeneratorRequest, error) {
    buf, err := ioutil.ReadAll(r)
    if err != nil {
        return nil, err
    }

    var req plugin.CodeGeneratorRequest
    if err = proto.Unmarshal(buf, &req); err != nil {
        return nil, err
    }
    return &req, nil
}

type FieldValidation struct {
    FieldValidator
    packages *[]string
    msgClasses *[]string
    field *descriptor.FieldDescriptorProto
}

func (fv *FieldValidation) generateMethod() string {
    var buf bytes.Buffer
    indent := generateIndent(len(*fv.parents()))

    io.WriteString(&buf, fmt.Sprintf("%sdef validate_%s\n", indent, fv.GetFieldName()))

    switch {
    case fv.IsStringField():
        fv.writeStringValidation(&buf)
    case fv.IsIntegerField():
        fv.writeIntegerValidation(&buf)
    case fv.IsFloatField():
        fv.writeFloatValidation(&buf)
    default:
        io.WriteString(&buf, fmt.Sprintf("%s\t# the vallidation (%s) is not supported yet\n", indent, fv.String()))
    }

    io.WriteString(&buf, fmt.Sprintf("%send\n", indent))

    return buf.String()
}

func (fv *FieldValidation) IsStringField() bool {
    return fv.field.GetType() == descriptor.FieldDescriptorProto_TYPE_STRING
}

func (fv *FieldValidation) IsBytesField() bool {
    return fv.field.GetType() == descriptor.FieldDescriptorProto_TYPE_BYTES
}

func (fv *FieldValidation) IsIntegerField() bool {
    switch fv.field.GetType() {
    case descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_TYPE_INT64:
        return true
    case descriptor.FieldDescriptorProto_TYPE_UINT32, descriptor.FieldDescriptorProto_TYPE_UINT64:
        return true
    case descriptor.FieldDescriptorProto_TYPE_SINT32, descriptor.FieldDescriptorProto_TYPE_SINT64:
        return true
    }
    return false
}

func (fv *FieldValidation) IsFloatField() bool {
    switch fv.field.GetType() {
    case descriptor.FieldDescriptorProto_TYPE_FLOAT, descriptor.FieldDescriptorProto_TYPE_DOUBLE:
        return true
    case descriptor.FieldDescriptorProto_TYPE_FIXED32, descriptor.FieldDescriptorProto_TYPE_FIXED64:
        return true
    case descriptor.FieldDescriptorProto_TYPE_SFIXED32, descriptor.FieldDescriptorProto_TYPE_SFIXED64:
        return true
    }
    return false
}

func (fv *FieldValidation) writeStringValidation(buf *bytes.Buffer) {
    indent := generateIndent(len(*fv.parents()))
    io.WriteString(buf, fmt.Sprintf("%s\t# StringField: %s\n", indent, fv.GetFieldName()))
}

func (fv *FieldValidation) writeBytesValidation(buf *bytes.Buffer) {
    indent := generateIndent(len(*fv.parents()))
    io.WriteString(buf, fmt.Sprintf("%s\t# BytesField: %s\n", indent, fv.GetFieldName()))
}

func (fv *FieldValidation) writeIntegerValidation(buf *bytes.Buffer) {
    indent := generateIndent(len(*fv.parents()))
    io.WriteString(buf, fmt.Sprintf("%s\t# IntegerField: %s\n", indent, fv.GetFieldName()))
}

func (fv *FieldValidation) writeFloatValidation(buf *bytes.Buffer) {
    indent := generateIndent(len(*fv.parents()))
    io.WriteString(buf, fmt.Sprintf("%s\t# FloatField: %s\n", indent, fv.GetFieldName()))
}

func (fv *FieldValidation) path() string {
    return strings.Join(append(*fv.packages, *fv.msgClasses...), "::")
}

func (fv *FieldValidation) parents() *[]string {
    parents := append(*fv.packages, *fv.msgClasses...)
    return &parents
}

func (fv *FieldValidation) GetFieldName() string {
    return fv.field.GetName()
}

func generateIndent(width int) string {
    return strings.Repeat("\t", width)
}

func ProcessRequest(req *plugin.CodeGeneratorRequest) []*FieldValidation {
    files := make(map[string]*descriptor.FileDescriptorProto)
    for _, f := range req.ProtoFile {
        files[f.GetName()] = f
    }

    var fields []*FieldValidation
    for _, fname := range req.FileToGenerate {
        f := files[fname]
        for _, m := range f.MessageType {
            pkgs := strings.Split(strings.Title(f.GetPackage()), ".")
            fields = append(fields, getValidatedFields(m, &pkgs, &[]string{})...)
		}
    }
    return fields
}

func getValidatedFields(m *descriptor.DescriptorProto, packages, msgClasses *[]string) []*FieldValidation {
    var fields []*FieldValidation
    classes := append(*msgClasses, strings.Title(m.GetName()))
    for _, field := range m.Field {
        v, ok := getValidator(field)
        if !ok { continue }
        fields = append(fields, &FieldValidation{
            FieldValidator: *v,
        	packages: packages,
        	msgClasses: &classes,
            field: field,
        })
    }
    for _, nested := range m.NestedType {
        fields = append(fields, getValidatedFields(nested, packages, &classes)...)
    }
    return fields
}

func getValidator(field *descriptor.FieldDescriptorProto) (*FieldValidator, bool) {
    ext, err := proto.GetExtension(field.Options, &proto.ExtensionDesc{
       // FIXME: ExtendedType does not have compatibility
       // ExtendedType: E_Field.ExtendedType,
       ExtensionType: E_Field.ExtensionType,
       Field: E_Field.Field,
       Name: E_Field.Name,
       Tag: E_Field.Tag,
       Filename: E_Field.Filename,
    })
    if err != nil { return nil, false }
    v, ok := ext.(*FieldValidator)
    if !ok { return nil, false }
    return v, true
}

func GenerateResponse(fields []*FieldValidation) *plugin.CodeGeneratorResponse {
    var buf bytes.Buffer
    fieldsByPath := make(map[string][]*FieldValidation)
    for _, fv := range fields {
        fieldsByPath[fv.path()] = append(fieldsByPath[fv.path()], fv)
    }
    for _, fvList := range fieldsByPath {
        for width, mod := range *(fvList[0].packages) {
            io.WriteString(&buf, fmt.Sprintf("%smodule %s\n", generateIndent(width), mod))
        }
        for idx, cls := range *(fvList[0].msgClasses) {
            indent := generateIndent(idx + len(*(fvList[0].packages)))
            io.WriteString(&buf, fmt.Sprintf("%sclass %s\n", indent, cls))
        }
        for _, fv := range fvList {
            io.WriteString(&buf, fv.generateMethod())
        }
        parents := *(fvList[0].parents())
        for idx := range parents {
            indent := generateIndent(len(parents) - idx - 1)
            io.WriteString(&buf, fmt.Sprintf("%send\n", indent))
        }
        io.WriteString(&buf, "\n")
    }

    return &plugin.CodeGeneratorResponse{
        File: []*plugin.CodeGeneratorResponse_File{
            {
                Name:    proto.String("validatedFields.rb"),
                Content: proto.String(buf.String()),
            },
        },
    }
}

func EmitResponse(resp *plugin.CodeGeneratorResponse) error {
    buf, err := proto.Marshal(resp)
    if err != nil {
        return err
    }
    _, err = os.Stdout.Write(buf)
    return err
}
