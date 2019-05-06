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

func ProcessRequest(req *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorResponse {
    files := make(map[string]*descriptor.FileDescriptorProto)
    for _, f := range req.ProtoFile {
        files[f.GetName()] = f
    }

    fields := make(map[string]*FieldValidator)
    for _, fname := range req.FileToGenerate {
        f := files[fname]
        for _, m := range f.MessageType {
            fields = merge(&fields, getValidatedFields(m, &[]string{f.GetPackage()}))
		}
    }

    var buf bytes.Buffer
    for path, validator := range fields {
        io.WriteString(&buf, fmt.Sprintf("%s => %s\n", path, validator.String()))
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

func getValidatedFields(m *descriptor.DescriptorProto, parents *[]string) *map[string]*FieldValidator {
    fields := make(map[string]*FieldValidator)
    current := append(*parents, m.GetName())
    for _, field := range m.Field {
        v, ok := getValidator(field)
        if !ok { continue }
        path := strings.Join(current, "::")
        n := fmt.Sprintf("%s#%s", path, field.GetName())
        fields[n] = v
    }
    for _, nested := range m.NestedType {
    	fields = merge(&fields, getValidatedFields(nested, &current))
    }
    return &fields
}

func merge(m1, m2 *map[string]*FieldValidator) map[string]*FieldValidator {
    ans := map[string]*FieldValidator{}
    for k, v := range *m1 {
        ans[k] = v
    }
    for k, v := range *m2 {
        ans[k] = v
    }
    return (ans)
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

func EmitResponse(resp *plugin.CodeGeneratorResponse) error {
    buf, err := proto.Marshal(resp)
    if err != nil {
        return err
    }
    _, err = os.Stdout.Write(buf)
    return err
}
