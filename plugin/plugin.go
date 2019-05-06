package plugin

import (
    "bytes"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "strings"

    validator "github.com/mwitkow/go-proto-validators"
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

    var buf bytes.Buffer
    for _, fname := range req.FileToGenerate {
        f := files[fname]
        for _, m := range f.MessageType {
            for _, fName := range getValidatedFields(m, &[]string{f.GetPackage()}) {
                io.WriteString(&buf, fName)
                io.WriteString(&buf, "\n")
            }
		}
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

func getValidatedFields(m *descriptor.DescriptorProto, parents *[]string) []string {
    var names []string
    current := append(*parents, m.GetName())
    for _, field := range m.Field {
        fName, ok := isValidatedField(field)
        if !ok { continue }
        path := strings.Join(current, "::")
        n := fmt.Sprintf("%s#%s", path, fName)
        names = append(names, n)
    }
    for _, nested := range m.NestedType {
    	names = append(names, getValidatedFields(nested, &current)...)
    }
    return names
}


func isValidatedField(field *descriptor.FieldDescriptorProto) (string, bool) {
    ext, err := proto.GetExtension(field.Options, &proto.ExtensionDesc{
       // FIXME: ExtendedType does not have compatibility
       // ExtendedType: validator.E_Field.ExtendedType,
       ExtensionType: validator.E_Field.ExtensionType,
       Field: validator.E_Field.Field,
       Name: validator.E_Field.Name,
       Tag: validator.E_Field.Tag,
       Filename: validator.E_Field.Filename,
    })
    if err != nil { return "", false }
    _, ok := ext.(*validator.FieldValidator)
    if !ok { return "", false }
    return field.GetName(), true
}

func EmitResponse(resp *plugin.CodeGeneratorResponse) error {
    buf, err := proto.Marshal(resp)
    if err != nil {
        return err
    }
    _, err = os.Stdout.Write(buf)
    return err
}
