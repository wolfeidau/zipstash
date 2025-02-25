// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        (unknown)
// source: provider/v1/provider.proto

package providerv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Provider represents supported CI/CD providers
type Provider int32

const (
	Provider_PROVIDER_UNSPECIFIED    Provider = 0
	Provider_PROVIDER_GITHUB_ACTIONS Provider = 1
	Provider_PROVIDER_GITLAB         Provider = 2
	Provider_PROVIDER_BUILDKITE      Provider = 3
)

// Enum value maps for Provider.
var (
	Provider_name = map[int32]string{
		0: "PROVIDER_UNSPECIFIED",
		1: "PROVIDER_GITHUB_ACTIONS",
		2: "PROVIDER_GITLAB",
		3: "PROVIDER_BUILDKITE",
	}
	Provider_value = map[string]int32{
		"PROVIDER_UNSPECIFIED":    0,
		"PROVIDER_GITHUB_ACTIONS": 1,
		"PROVIDER_GITLAB":         2,
		"PROVIDER_BUILDKITE":      3,
	}
)

func (x Provider) Enum() *Provider {
	p := new(Provider)
	*p = x
	return p
}

func (x Provider) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Provider) Descriptor() protoreflect.EnumDescriptor {
	return file_provider_v1_provider_proto_enumTypes[0].Descriptor()
}

func (Provider) Type() protoreflect.EnumType {
	return &file_provider_v1_provider_proto_enumTypes[0]
}

func (x Provider) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Provider.Descriptor instead.
func (Provider) EnumDescriptor() ([]byte, []int) {
	return file_provider_v1_provider_proto_rawDescGZIP(), []int{0}
}

var File_provider_v1_provider_proto protoreflect.FileDescriptor

var file_provider_v1_provider_proto_rawDesc = string([]byte{
	0x0a, 0x1a, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2a, 0x6e, 0x0a, 0x08, 0x50, 0x72, 0x6f,
	0x76, 0x69, 0x64, 0x65, 0x72, 0x12, 0x18, 0x0a, 0x14, 0x50, 0x52, 0x4f, 0x56, 0x49, 0x44, 0x45,
	0x52, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12,
	0x1b, 0x0a, 0x17, 0x50, 0x52, 0x4f, 0x56, 0x49, 0x44, 0x45, 0x52, 0x5f, 0x47, 0x49, 0x54, 0x48,
	0x55, 0x42, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x53, 0x10, 0x01, 0x12, 0x13, 0x0a, 0x0f,
	0x50, 0x52, 0x4f, 0x56, 0x49, 0x44, 0x45, 0x52, 0x5f, 0x47, 0x49, 0x54, 0x4c, 0x41, 0x42, 0x10,
	0x02, 0x12, 0x16, 0x0a, 0x12, 0x50, 0x52, 0x4f, 0x56, 0x49, 0x44, 0x45, 0x52, 0x5f, 0x42, 0x55,
	0x49, 0x4c, 0x44, 0x4b, 0x49, 0x54, 0x45, 0x10, 0x03, 0x42, 0xb4, 0x01, 0x0a, 0x0f, 0x63, 0x6f,
	0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x42, 0x0d, 0x50,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x45,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x6f, 0x6c, 0x66, 0x65,
	0x69, 0x64, 0x61, 0x75, 0x2f, 0x7a, 0x69, 0x70, 0x73, 0x74, 0x61, 0x73, 0x68, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x3b, 0x70, 0x72, 0x6f, 0x76, 0x69,
	0x64, 0x65, 0x72, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x50, 0x58, 0x58, 0xaa, 0x02, 0x0b, 0x50, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x0b, 0x50, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x17, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64,
	0x65, 0x72, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0xea, 0x02, 0x0c, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x3a, 0x3a, 0x56, 0x31,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_provider_v1_provider_proto_rawDescOnce sync.Once
	file_provider_v1_provider_proto_rawDescData []byte
)

func file_provider_v1_provider_proto_rawDescGZIP() []byte {
	file_provider_v1_provider_proto_rawDescOnce.Do(func() {
		file_provider_v1_provider_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_provider_v1_provider_proto_rawDesc), len(file_provider_v1_provider_proto_rawDesc)))
	})
	return file_provider_v1_provider_proto_rawDescData
}

var file_provider_v1_provider_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_provider_v1_provider_proto_goTypes = []any{
	(Provider)(0), // 0: provider.v1.Provider
}
var file_provider_v1_provider_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_provider_v1_provider_proto_init() }
func file_provider_v1_provider_proto_init() {
	if File_provider_v1_provider_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_provider_v1_provider_proto_rawDesc), len(file_provider_v1_provider_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_provider_v1_provider_proto_goTypes,
		DependencyIndexes: file_provider_v1_provider_proto_depIdxs,
		EnumInfos:         file_provider_v1_provider_proto_enumTypes,
	}.Build()
	File_provider_v1_provider_proto = out.File
	file_provider_v1_provider_proto_goTypes = nil
	file_provider_v1_provider_proto_depIdxs = nil
}
