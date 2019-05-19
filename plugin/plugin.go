package plugin

import (
    "bytes"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "strings"

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
    packages *[]string
    msgClasses *[]string
    field *descriptor.FieldDescriptorProto
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
        fields = append(fields, &FieldValidation{
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
        var fNames []string
        for _, fv := range fvList {
            fNames = append(fNames, fv.GetFieldName())
        }
        parents := *(fvList[0].parents())
        indent := generateIndent(len(parents))
        lines := []string{
            "def self.build(**kwargs)",
            fmt.Sprintf("\tattributes = kwargs.select{|k, _| %%i(%s).include? k}", strings.Join(fNames, " ")),
            "\tself.new(attributes)",
            "end",
        }
        for _, line := range lines {
            io.WriteString(&buf, fmt.Sprintf("%s%s\n", indent, line))
        }
        println()
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
