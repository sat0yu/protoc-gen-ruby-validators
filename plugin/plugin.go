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
    path string
    field *descriptor.FieldDescriptorProto
    validator *FieldValidator
}

func ProcessRequest(req *plugin.CodeGeneratorRequest) *[]*FieldValidation {
    files := make(map[string]*descriptor.FileDescriptorProto)
    for _, f := range req.ProtoFile {
        files[f.GetName()] = f
    }

    var fields []*FieldValidation
    for _, fname := range req.FileToGenerate {
        f := files[fname]
        for _, m := range f.MessageType {
            fields = append(fields, *getValidatedFields(m, &[]string{f.GetPackage()})...)
		}
    }
    return &fields
}

func getValidatedFields(m *descriptor.DescriptorProto, parents *[]string) *[]*FieldValidation {
    var fields []*FieldValidation
    current := append(*parents, m.GetName())
    for _, field := range m.Field {
        v, ok := getValidator(field)
        if !ok { continue }
        path := strings.Join(current, "::")
        fields = append(fields, &FieldValidation{
            path: path,
            field: field,
            validator: v,
        })
    }
    for _, nested := range m.NestedType {
        fields = append(fields, *getValidatedFields(nested, &current)...)
    }
    return &fields
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

func GenerateResponse(fields *[]*FieldValidation) *plugin.CodeGeneratorResponse {
    var buf bytes.Buffer
    for _, fv := range *fields {
        io.WriteString(&buf, fmt.Sprintf("%s#%s => %s\n", fv.path, fv.field.GetName(), fv.validator.String()))
    }

    return &plugin.CodeGeneratorResponse{
        File: []*plugin.CodeGeneratorResponse_File{
            {
                Name:    proto.String("validatedFields.txt"),
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
