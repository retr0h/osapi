// Copyright (c) 2024 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v4.25.1
// source: task/task.proto

package task

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

// The type of action: reboot or shutdown
type ShutdownAction_ActionType int32

const (
	ShutdownAction_REBOOT   ShutdownAction_ActionType = 0
	ShutdownAction_SHUTDOWN ShutdownAction_ActionType = 1
)

// Enum value maps for ShutdownAction_ActionType.
var (
	ShutdownAction_ActionType_name = map[int32]string{
		0: "REBOOT",
		1: "SHUTDOWN",
	}
	ShutdownAction_ActionType_value = map[string]int32{
		"REBOOT":   0,
		"SHUTDOWN": 1,
	}
)

func (x ShutdownAction_ActionType) Enum() *ShutdownAction_ActionType {
	p := new(ShutdownAction_ActionType)
	*p = x
	return p
}

func (x ShutdownAction_ActionType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ShutdownAction_ActionType) Descriptor() protoreflect.EnumDescriptor {
	return file_task_task_proto_enumTypes[0].Descriptor()
}

func (ShutdownAction_ActionType) Type() protoreflect.EnumType {
	return &file_task_task_proto_enumTypes[0]
}

func (x ShutdownAction_ActionType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ShutdownAction_ActionType.Descriptor instead.
func (ShutdownAction_ActionType) EnumDescriptor() ([]byte, []int) {
	return file_task_task_proto_rawDescGZIP(), []int{0, 0}
}

// ShutdownAction represents an action to reboot or shutdown the system.
type ShutdownAction struct {
	state      protoimpl.MessageState    `protogen:"open.v1"`
	ActionType ShutdownAction_ActionType `protobuf:"varint,1,opt,name=action_type,json=actionType,proto3,enum=task.ShutdownAction_ActionType" json:"action_type,omitempty"`
	// Optional field to specify a delay in seconds before reboot/shutdown
	DelaySeconds int32 `protobuf:"varint,2,opt,name=delay_seconds,json=delaySeconds,proto3" json:"delay_seconds,omitempty"`
	// Optional message to log or display before reboot/shutdown
	Message       string `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ShutdownAction) Reset() {
	*x = ShutdownAction{}
	mi := &file_task_task_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ShutdownAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownAction) ProtoMessage() {}

func (x *ShutdownAction) ProtoReflect() protoreflect.Message {
	mi := &file_task_task_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShutdownAction.ProtoReflect.Descriptor instead.
func (*ShutdownAction) Descriptor() ([]byte, []int) {
	return file_task_task_proto_rawDescGZIP(), []int{0}
}

func (x *ShutdownAction) GetActionType() ShutdownAction_ActionType {
	if x != nil {
		return x.ActionType
	}
	return ShutdownAction_REBOOT
}

func (x *ShutdownAction) GetDelaySeconds() int32 {
	if x != nil {
		return x.DelaySeconds
	}
	return 0
}

func (x *ShutdownAction) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

// ChangeDNSAction represents an action to change DNS settings.
type ChangeDNSAction struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// List of DNS server IP addresses (IPv4 or IPv6)
	DnsServers []string `protobuf:"bytes,1,rep,name=dns_servers,json=dnsServers,proto3" json:"dns_servers,omitempty"`
	// List of search domains for DNS resolution
	SearchDomains []string `protobuf:"bytes,2,rep,name=search_domains,json=searchDomains,proto3" json:"search_domains,omitempty"`
	// The name of the network interface to apply DNS settings to
	InterfaceName string `protobuf:"bytes,3,opt,name=interface_name,json=interfaceName,proto3" json:"interface_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChangeDNSAction) Reset() {
	*x = ChangeDNSAction{}
	mi := &file_task_task_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChangeDNSAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChangeDNSAction) ProtoMessage() {}

func (x *ChangeDNSAction) ProtoReflect() protoreflect.Message {
	mi := &file_task_task_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChangeDNSAction.ProtoReflect.Descriptor instead.
func (*ChangeDNSAction) Descriptor() ([]byte, []int) {
	return file_task_task_proto_rawDescGZIP(), []int{1}
}

func (x *ChangeDNSAction) GetDnsServers() []string {
	if x != nil {
		return x.DnsServers
	}
	return nil
}

func (x *ChangeDNSAction) GetSearchDomains() []string {
	if x != nil {
		return x.SearchDomains
	}
	return nil
}

func (x *ChangeDNSAction) GetInterfaceName() string {
	if x != nil {
		return x.InterfaceName
	}
	return ""
}

type Task struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Action:
	//
	//	*Task_ShutdownAction
	//	*Task_ChangeDnsAction
	Action        isTask_Action `protobuf_oneof:"action"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Task) Reset() {
	*x = Task{}
	mi := &file_task_task_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Task) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Task) ProtoMessage() {}

func (x *Task) ProtoReflect() protoreflect.Message {
	mi := &file_task_task_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Task.ProtoReflect.Descriptor instead.
func (*Task) Descriptor() ([]byte, []int) {
	return file_task_task_proto_rawDescGZIP(), []int{2}
}

func (x *Task) GetAction() isTask_Action {
	if x != nil {
		return x.Action
	}
	return nil
}

func (x *Task) GetShutdownAction() *ShutdownAction {
	if x != nil {
		if x, ok := x.Action.(*Task_ShutdownAction); ok {
			return x.ShutdownAction
		}
	}
	return nil
}

func (x *Task) GetChangeDnsAction() *ChangeDNSAction {
	if x != nil {
		if x, ok := x.Action.(*Task_ChangeDnsAction); ok {
			return x.ChangeDnsAction
		}
	}
	return nil
}

type isTask_Action interface {
	isTask_Action()
}

type Task_ShutdownAction struct {
	ShutdownAction *ShutdownAction `protobuf:"bytes,1,opt,name=shutdown_action,json=shutdownAction,proto3,oneof"`
}

type Task_ChangeDnsAction struct {
	ChangeDnsAction *ChangeDNSAction `protobuf:"bytes,2,opt,name=change_dns_action,json=changeDnsAction,proto3,oneof"`
}

func (*Task_ShutdownAction) isTask_Action() {}

func (*Task_ChangeDnsAction) isTask_Action() {}

var File_task_task_proto protoreflect.FileDescriptor

var file_task_task_proto_rawDesc = string([]byte{
	0x0a, 0x0f, 0x74, 0x61, 0x73, 0x6b, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x04, 0x74, 0x61, 0x73, 0x6b, 0x22, 0xb9, 0x01, 0x0a, 0x0e, 0x53, 0x68, 0x75, 0x74,
	0x64, 0x6f, 0x77, 0x6e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x40, 0x0a, 0x0b, 0x61, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x1f, 0x2e, 0x74, 0x61, 0x73, 0x6b, 0x2e, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x41,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65,
	0x52, 0x0a, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x12, 0x23, 0x0a, 0x0d,
	0x64, 0x65, 0x6c, 0x61, 0x79, 0x5f, 0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x73, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x0c, 0x64, 0x65, 0x6c, 0x61, 0x79, 0x53, 0x65, 0x63, 0x6f, 0x6e, 0x64,
	0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x26, 0x0a, 0x0a, 0x41,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x52, 0x45, 0x42,
	0x4f, 0x4f, 0x54, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x53, 0x48, 0x55, 0x54, 0x44, 0x4f, 0x57,
	0x4e, 0x10, 0x01, 0x22, 0x80, 0x01, 0x0a, 0x0f, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x44, 0x4e,
	0x53, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1f, 0x0a, 0x0b, 0x64, 0x6e, 0x73, 0x5f, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x64, 0x6e,
	0x73, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x73, 0x65, 0x61, 0x72,
	0x63, 0x68, 0x5f, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x0d, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x44, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x73, 0x12,
	0x25, 0x0a, 0x0e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61,
	0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x96, 0x01, 0x0a, 0x04, 0x54, 0x61, 0x73, 0x6b, 0x12,
	0x3f, 0x0a, 0x0f, 0x73, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x5f, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x74, 0x61, 0x73, 0x6b, 0x2e,
	0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00,
	0x52, 0x0e, 0x73, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x43, 0x0a, 0x11, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x5f, 0x64, 0x6e, 0x73, 0x5f, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x74, 0x61,
	0x73, 0x6b, 0x2e, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x44, 0x4e, 0x53, 0x41, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x48, 0x00, 0x52, 0x0f, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x44, 0x6e, 0x73, 0x41,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x08, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x42,
	0x36, 0x5a, 0x34, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x65,
	0x74, 0x72, 0x30, 0x68, 0x2f, 0x6f, 0x73, 0x61, 0x70, 0x69, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_task_task_proto_rawDescOnce sync.Once
	file_task_task_proto_rawDescData []byte
)

func file_task_task_proto_rawDescGZIP() []byte {
	file_task_task_proto_rawDescOnce.Do(func() {
		file_task_task_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_task_task_proto_rawDesc), len(file_task_task_proto_rawDesc)))
	})
	return file_task_task_proto_rawDescData
}

var file_task_task_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_task_task_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_task_task_proto_goTypes = []any{
	(ShutdownAction_ActionType)(0), // 0: task.ShutdownAction.ActionType
	(*ShutdownAction)(nil),         // 1: task.ShutdownAction
	(*ChangeDNSAction)(nil),        // 2: task.ChangeDNSAction
	(*Task)(nil),                   // 3: task.Task
}
var file_task_task_proto_depIdxs = []int32{
	0, // 0: task.ShutdownAction.action_type:type_name -> task.ShutdownAction.ActionType
	1, // 1: task.Task.shutdown_action:type_name -> task.ShutdownAction
	2, // 2: task.Task.change_dns_action:type_name -> task.ChangeDNSAction
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_task_task_proto_init() }
func file_task_task_proto_init() {
	if File_task_task_proto != nil {
		return
	}
	file_task_task_proto_msgTypes[2].OneofWrappers = []any{
		(*Task_ShutdownAction)(nil),
		(*Task_ChangeDnsAction)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_task_task_proto_rawDesc), len(file_task_task_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_task_task_proto_goTypes,
		DependencyIndexes: file_task_task_proto_depIdxs,
		EnumInfos:         file_task_task_proto_enumTypes,
		MessageInfos:      file_task_task_proto_msgTypes,
	}.Build()
	File_task_task_proto = out.File
	file_task_task_proto_goTypes = nil
	file_task_task_proto_depIdxs = nil
}
