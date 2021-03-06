// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.8
// source: model/relay.proto

package model

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RelayClientContext struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ConnectionId string `protobuf:"bytes,1,opt,name=connectionId,proto3" json:"connectionId,omitempty"`
	SourcePeerId string `protobuf:"bytes,2,opt,name=sourcePeerId,proto3" json:"sourcePeerId,omitempty"`
	TargetPeerId string `protobuf:"bytes,3,opt,name=targetPeerId,proto3" json:"targetPeerId,omitempty"`
}

func (x *RelayClientContext) Reset() {
	*x = RelayClientContext{}
	if protoimpl.UnsafeEnabled {
		mi := &file_model_relay_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RelayClientContext) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RelayClientContext) ProtoMessage() {}

func (x *RelayClientContext) ProtoReflect() protoreflect.Message {
	mi := &file_model_relay_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RelayClientContext.ProtoReflect.Descriptor instead.
func (*RelayClientContext) Descriptor() ([]byte, []int) {
	return file_model_relay_proto_rawDescGZIP(), []int{0}
}

func (x *RelayClientContext) GetConnectionId() string {
	if x != nil {
		return x.ConnectionId
	}
	return ""
}

func (x *RelayClientContext) GetSourcePeerId() string {
	if x != nil {
		return x.SourcePeerId
	}
	return ""
}

func (x *RelayClientContext) GetTargetPeerId() string {
	if x != nil {
		return x.TargetPeerId
	}
	return ""
}

var File_model_relay_proto protoreflect.FileDescriptor

var file_model_relay_proto_rawDesc = []byte{
	0x0a, 0x11, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2f, 0x72, 0x65, 0x6c, 0x61, 0x79, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x05, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x22, 0x80, 0x01, 0x0a, 0x12, 0x52,
	0x65, 0x6c, 0x61, 0x79, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78,
	0x74, 0x12, 0x22, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x22, 0x0a, 0x0c, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50,
	0x65, 0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x50, 0x65, 0x65, 0x72, 0x49, 0x64, 0x12, 0x22, 0x0a, 0x0c, 0x74, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x50, 0x65, 0x65, 0x72, 0x49, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0c, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x50, 0x65, 0x65, 0x72, 0x49, 0x64, 0x42, 0x08, 0x5a,
	0x06, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_model_relay_proto_rawDescOnce sync.Once
	file_model_relay_proto_rawDescData = file_model_relay_proto_rawDesc
)

func file_model_relay_proto_rawDescGZIP() []byte {
	file_model_relay_proto_rawDescOnce.Do(func() {
		file_model_relay_proto_rawDescData = protoimpl.X.CompressGZIP(file_model_relay_proto_rawDescData)
	})
	return file_model_relay_proto_rawDescData
}

var file_model_relay_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_model_relay_proto_goTypes = []interface{}{
	(*RelayClientContext)(nil), // 0: model.RelayClientContext
}
var file_model_relay_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_model_relay_proto_init() }
func file_model_relay_proto_init() {
	if File_model_relay_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_model_relay_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RelayClientContext); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_model_relay_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_model_relay_proto_goTypes,
		DependencyIndexes: file_model_relay_proto_depIdxs,
		MessageInfos:      file_model_relay_proto_msgTypes,
	}.Build()
	File_model_relay_proto = out.File
	file_model_relay_proto_rawDesc = nil
	file_model_relay_proto_goTypes = nil
	file_model_relay_proto_depIdxs = nil
}
