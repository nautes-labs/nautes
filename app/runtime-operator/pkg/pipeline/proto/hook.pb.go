// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.23.2
// source: hook.proto

package proto

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

// // HookBuildData stores information used to build the pre- and post-hook of the pipeline.
type HookBuildData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// UserVars are parameters entered by the user.
	UserVars map[string]string `protobuf:"bytes,1,rep,name=UserVars,proto3" json:"UserVars,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// BuiltinVars is the information about the pipeline in the runtime, such as the code library address and code library type.
	// For details, see  https://github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/pipeline.go
	BuiltinVars map[string]string `protobuf:"bytes,2,rep,name=BuiltinVars,proto3" json:"BuiltinVars,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// EventSourceType is the type of the event source corresponding to the hook.
	EventSourceType string `protobuf:"bytes,3,opt,name=EventSourceType,proto3" json:"EventSourceType,omitempty"`
	// EventType is specific event type in the event source.
	EventType string `protobuf:"bytes,4,opt,name=EventType,proto3" json:"EventType,omitempty"`
}

func (x *HookBuildData) Reset() {
	*x = HookBuildData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HookBuildData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HookBuildData) ProtoMessage() {}

func (x *HookBuildData) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HookBuildData.ProtoReflect.Descriptor instead.
func (*HookBuildData) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{0}
}

func (x *HookBuildData) GetUserVars() map[string]string {
	if x != nil {
		return x.UserVars
	}
	return nil
}

func (x *HookBuildData) GetBuiltinVars() map[string]string {
	if x != nil {
		return x.BuiltinVars
	}
	return nil
}

func (x *HookBuildData) GetEventSourceType() string {
	if x != nil {
		return x.EventSourceType
	}
	return ""
}

func (x *HookBuildData) GetEventType() string {
	if x != nil {
		return x.EventType
	}
	return ""
}

// HookMetadata stores the metadata information of the hook.
type HookMetadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=Name,proto3" json:"Name,omitempty"`
	// IsPreHook indicates whether it can be set as a pre-step.
	IsPreHook bool `protobuf:"varint,2,opt,name=IsPreHook,proto3" json:"IsPreHook,omitempty"`
	// IsPostHook indicates whether it can be set as a post-step.
	IsPostHook bool `protobuf:"varint,3,opt,name=IsPostHook,proto3" json:"IsPostHook,omitempty"`
	// SupportEventSourceTypes is the event source type supported by the hook.If it is empty, it means any type is supported.
	SupportEventSourceTypes []string `protobuf:"bytes,4,rep,name=SupportEventSourceTypes,proto3" json:"SupportEventSourceTypes,omitempty"`
	// VarsDefinition is the description information of the input and output parameters of the hook, in JSONSchema format.
	VarsDefinition []byte `protobuf:"bytes,5,opt,name=VarsDefinition,proto3,oneof" json:"VarsDefinition,omitempty"`
}

func (x *HookMetadata) Reset() {
	*x = HookMetadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HookMetadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HookMetadata) ProtoMessage() {}

func (x *HookMetadata) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HookMetadata.ProtoReflect.Descriptor instead.
func (*HookMetadata) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{1}
}

func (x *HookMetadata) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *HookMetadata) GetIsPreHook() bool {
	if x != nil {
		return x.IsPreHook
	}
	return false
}

func (x *HookMetadata) GetIsPostHook() bool {
	if x != nil {
		return x.IsPostHook
	}
	return false
}

func (x *HookMetadata) GetSupportEventSourceTypes() []string {
	if x != nil {
		return x.SupportEventSourceTypes
	}
	return nil
}

func (x *HookMetadata) GetVarsDefinition() []byte {
	if x != nil {
		return x.VarsDefinition
	}
	return nil
}

// Hook is a task that can be executed before or after the user pipeline.
type Hook struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// RequestVars is the information that needs to be input from the outside for the hook to run.
	RequestInputs []*InputOverWrite `protobuf:"bytes,1,rep,name=RequestInputs,proto3" json:"RequestInputs,omitempty"`
	// RequestResources are the resources that need to be deployed in the environment for the hook to run properly.
	RequestResources [][]byte `protobuf:"bytes,2,rep,name=RequestResources,proto3" json:"RequestResources,omitempty"`
	// Resource is the code fragment that runs the hook, which is used to splice into the pre- and post-steps of the user pipeline.
	Resource []byte `protobuf:"bytes,3,opt,name=Resource,proto3" json:"Resource,omitempty"`
}

func (x *Hook) Reset() {
	*x = Hook{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Hook) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Hook) ProtoMessage() {}

func (x *Hook) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Hook.ProtoReflect.Descriptor instead.
func (*Hook) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{2}
}

func (x *Hook) GetRequestInputs() []*InputOverWrite {
	if x != nil {
		return x.RequestInputs
	}
	return nil
}

func (x *Hook) GetRequestResources() [][]byte {
	if x != nil {
		return x.RequestResources
	}
	return nil
}

func (x *Hook) GetResource() []byte {
	if x != nil {
		return x.Resource
	}
	return nil
}

// InputOverWrite defines the replacement information for the Hook input parameter.
type InputOverWrite struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Source is the source of the data.
	Source *InputSource `protobuf:"bytes,1,opt,name=Source,proto3" json:"Source,omitempty"`
	// The location to be filled in, the format is determined by the pipeline implementation.
	Destination string `protobuf:"bytes,2,opt,name=Destination,proto3" json:"Destination,omitempty"`
}

func (x *InputOverWrite) Reset() {
	*x = InputOverWrite{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InputOverWrite) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InputOverWrite) ProtoMessage() {}

func (x *InputOverWrite) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InputOverWrite.ProtoReflect.Descriptor instead.
func (*InputOverWrite) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{3}
}

func (x *InputOverWrite) GetSource() *InputSource {
	if x != nil {
		return x.Source
	}
	return nil
}

func (x *InputOverWrite) GetDestination() string {
	if x != nil {
		return x.Destination
	}
	return ""
}

// InputSource defines the source of the Hook parameter.
type InputSource struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Src:
	//
	//	*InputSource_FromEventSource
	Src isInputSource_Src `protobuf_oneof:"src"`
}

func (x *InputSource) Reset() {
	*x = InputSource{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InputSource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InputSource) ProtoMessage() {}

func (x *InputSource) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InputSource.ProtoReflect.Descriptor instead.
func (*InputSource) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{4}
}

func (m *InputSource) GetSrc() isInputSource_Src {
	if m != nil {
		return m.Src
	}
	return nil
}

func (x *InputSource) GetFromEventSource() string {
	if x, ok := x.GetSrc().(*InputSource_FromEventSource); ok {
		return x.FromEventSource
	}
	return ""
}

type isInputSource_Src interface {
	isInputSource_Src()
}

type InputSource_FromEventSource struct {
	// FromEventSource records the location where data is obtained from the event source.
	FromEventSource string `protobuf:"bytes,1,opt,name=FromEventSource,proto3,oneof"`
}

func (*InputSource_FromEventSource) isInputSource_Src() {}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{5}
}

type GetPipelineTypeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// PipelineType is the name of the pipeline supported by the plugin.
	PipelineType string `protobuf:"bytes,1,opt,name=PipelineType,proto3" json:"PipelineType,omitempty"`
}

func (x *GetPipelineTypeResponse) Reset() {
	*x = GetPipelineTypeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetPipelineTypeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPipelineTypeResponse) ProtoMessage() {}

func (x *GetPipelineTypeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPipelineTypeResponse.ProtoReflect.Descriptor instead.
func (*GetPipelineTypeResponse) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{6}
}

func (x *GetPipelineTypeResponse) GetPipelineType() string {
	if x != nil {
		return x.PipelineType
	}
	return ""
}

type GetHooksMetaDataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// HooksMetadata is the hook metadata information supported by the plugin.
	HooksMetaData []*HookMetadata `protobuf:"bytes,1,rep,name=HooksMetaData,proto3" json:"HooksMetaData,omitempty"`
}

func (x *GetHooksMetaDataResponse) Reset() {
	*x = GetHooksMetaDataResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetHooksMetaDataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetHooksMetaDataResponse) ProtoMessage() {}

func (x *GetHooksMetaDataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetHooksMetaDataResponse.ProtoReflect.Descriptor instead.
func (*GetHooksMetaDataResponse) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{7}
}

func (x *GetHooksMetaDataResponse) GetHooksMetaData() []*HookMetadata {
	if x != nil {
		return x.HooksMetaData
	}
	return nil
}

type BuildHookRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// HookName is the hook object to be created.
	HookName string `protobuf:"bytes,1,opt,name=HookName,proto3" json:"HookName,omitempty"`
	// HookBuildData is the parameter required for building the hook.
	Info *HookBuildData `protobuf:"bytes,2,opt,name=Info,proto3" json:"Info,omitempty"`
}

func (x *BuildHookRequest) Reset() {
	*x = BuildHookRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BuildHookRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildHookRequest) ProtoMessage() {}

func (x *BuildHookRequest) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildHookRequest.ProtoReflect.Descriptor instead.
func (*BuildHookRequest) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{8}
}

func (x *BuildHookRequest) GetHookName() string {
	if x != nil {
		return x.HookName
	}
	return ""
}

func (x *BuildHookRequest) GetInfo() *HookBuildData {
	if x != nil {
		return x.Info
	}
	return nil
}

type BuildHookResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Hook is a complete Hook information, which includes the code fragment that can be inserted into the Hooks,
	// the external input information required for the fragment,
	// and the environment information required for the fragment to run.
	Hook *Hook `protobuf:"bytes,1,opt,name=Hook,proto3" json:"Hook,omitempty"`
}

func (x *BuildHookResponse) Reset() {
	*x = BuildHookResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hook_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BuildHookResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildHookResponse) ProtoMessage() {}

func (x *BuildHookResponse) ProtoReflect() protoreflect.Message {
	mi := &file_hook_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildHookResponse.ProtoReflect.Descriptor instead.
func (*BuildHookResponse) Descriptor() ([]byte, []int) {
	return file_hook_proto_rawDescGZIP(), []int{9}
}

func (x *BuildHookResponse) GetHook() *Hook {
	if x != nil {
		return x.Hook
	}
	return nil
}

var File_hook_proto protoreflect.FileDescriptor

var file_hook_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x68, 0x6f, 0x6f, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0xdd, 0x02, 0x0a, 0x0d, 0x48, 0x6f, 0x6f, 0x6b, 0x42, 0x75, 0x69, 0x6c,
	0x64, 0x44, 0x61, 0x74, 0x61, 0x12, 0x3e, 0x0a, 0x08, 0x55, 0x73, 0x65, 0x72, 0x56, 0x61, 0x72,
	0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x48, 0x6f, 0x6f, 0x6b, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x61, 0x74, 0x61, 0x2e, 0x55, 0x73,
	0x65, 0x72, 0x56, 0x61, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x55, 0x73, 0x65,
	0x72, 0x56, 0x61, 0x72, 0x73, 0x12, 0x47, 0x0a, 0x0b, 0x42, 0x75, 0x69, 0x6c, 0x74, 0x69, 0x6e,
	0x56, 0x61, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x48, 0x6f, 0x6f, 0x6b, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x61, 0x74, 0x61,
	0x2e, 0x42, 0x75, 0x69, 0x6c, 0x74, 0x69, 0x6e, 0x56, 0x61, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x0b, 0x42, 0x75, 0x69, 0x6c, 0x74, 0x69, 0x6e, 0x56, 0x61, 0x72, 0x73, 0x12, 0x28,
	0x0a, 0x0f, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x54, 0x79, 0x70,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x54, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x1a, 0x3b, 0x0a, 0x0d, 0x55, 0x73, 0x65, 0x72, 0x56, 0x61,
	0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x1a, 0x3e, 0x0a, 0x10, 0x42, 0x75, 0x69, 0x6c, 0x74, 0x69, 0x6e, 0x56, 0x61,
	0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x22, 0xda, 0x01, 0x0a, 0x0c, 0x48, 0x6f, 0x6f, 0x6b, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x49, 0x73, 0x50, 0x72,
	0x65, 0x48, 0x6f, 0x6f, 0x6b, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x49, 0x73, 0x50,
	0x72, 0x65, 0x48, 0x6f, 0x6f, 0x6b, 0x12, 0x1e, 0x0a, 0x0a, 0x49, 0x73, 0x50, 0x6f, 0x73, 0x74,
	0x48, 0x6f, 0x6f, 0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x49, 0x73, 0x50, 0x6f,
	0x73, 0x74, 0x48, 0x6f, 0x6f, 0x6b, 0x12, 0x38, 0x0a, 0x17, 0x53, 0x75, 0x70, 0x70, 0x6f, 0x72,
	0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x54, 0x79, 0x70, 0x65,
	0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x17, 0x53, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x54, 0x79, 0x70, 0x65, 0x73,
	0x12, 0x2b, 0x0a, 0x0e, 0x56, 0x61, 0x72, 0x73, 0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x48, 0x00, 0x52, 0x0e, 0x56, 0x61, 0x72, 0x73,
	0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x88, 0x01, 0x01, 0x42, 0x11, 0x0a,
	0x0f, 0x5f, 0x56, 0x61, 0x72, 0x73, 0x44, 0x65, 0x66, 0x69, 0x6e, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x22, 0x8b, 0x01, 0x0a, 0x04, 0x48, 0x6f, 0x6f, 0x6b, 0x12, 0x3b, 0x0a, 0x0d, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x15, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x4f, 0x76,
	0x65, 0x72, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52, 0x0d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x49, 0x6e, 0x70, 0x75, 0x74, 0x73, 0x12, 0x2a, 0x0a, 0x10, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c,
	0x52, 0x10, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x22, 0x5e,
	0x0a, 0x0e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x4f, 0x76, 0x65, 0x72, 0x57, 0x72, 0x69, 0x74, 0x65,
	0x12, 0x2a, 0x0a, 0x06, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x53, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x52, 0x06, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x20, 0x0a, 0x0b,
	0x44, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x44, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x40,
	0x0a, 0x0b, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x2a, 0x0a,
	0x0f, 0x46, 0x72, 0x6f, 0x6d, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0f, 0x46, 0x72, 0x6f, 0x6d, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x05, 0x0a, 0x03, 0x73, 0x72, 0x63,
	0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x3d, 0x0a, 0x17, 0x47, 0x65, 0x74,
	0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x22, 0x0a, 0x0c, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65,
	0x54, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x50, 0x69, 0x70, 0x65,
	0x6c, 0x69, 0x6e, 0x65, 0x54, 0x79, 0x70, 0x65, 0x22, 0x55, 0x0a, 0x18, 0x47, 0x65, 0x74, 0x48,
	0x6f, 0x6f, 0x6b, 0x73, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x39, 0x0a, 0x0d, 0x48, 0x6f, 0x6f, 0x6b, 0x73, 0x4d, 0x65, 0x74,
	0x61, 0x44, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x48, 0x6f, 0x6f, 0x6b, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x52, 0x0d, 0x48, 0x6f, 0x6f, 0x6b, 0x73, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x22,
	0x58, 0x0a, 0x10, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x48, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x48, 0x6f, 0x6f, 0x6b, 0x4e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x48, 0x6f, 0x6f, 0x6b, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x28, 0x0a, 0x04, 0x49, 0x6e, 0x66, 0x6f, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x48, 0x6f, 0x6f, 0x6b, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x44,
	0x61, 0x74, 0x61, 0x52, 0x04, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0x34, 0x0a, 0x11, 0x42, 0x75, 0x69,
	0x6c, 0x64, 0x48, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1f,
	0x0a, 0x04, 0x48, 0x6f, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x48, 0x6f, 0x6f, 0x6b, 0x52, 0x04, 0x48, 0x6f, 0x6f, 0x6b, 0x32,
	0xd1, 0x01, 0x0a, 0x0b, 0x48, 0x6f, 0x6f, 0x6b, 0x46, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x12,
	0x3f, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x50, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x0c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x1a, 0x1e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x50, 0x69, 0x70, 0x65,
	0x6c, 0x69, 0x6e, 0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x41, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x48, 0x6f, 0x6f, 0x6b, 0x73, 0x4d, 0x65, 0x74, 0x61,
	0x44, 0x61, 0x74, 0x61, 0x12, 0x0c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x1a, 0x1f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x48, 0x6f,
	0x6f, 0x6b, 0x73, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x3e, 0x0a, 0x09, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x48, 0x6f, 0x6f, 0x6b,
	0x12, 0x17, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x48, 0x6f,
	0x6f, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x48, 0x6f, 0x6f, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x42, 0x09, 0x5a, 0x07, 0x2e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_hook_proto_rawDescOnce sync.Once
	file_hook_proto_rawDescData = file_hook_proto_rawDesc
)

func file_hook_proto_rawDescGZIP() []byte {
	file_hook_proto_rawDescOnce.Do(func() {
		file_hook_proto_rawDescData = protoimpl.X.CompressGZIP(file_hook_proto_rawDescData)
	})
	return file_hook_proto_rawDescData
}

var file_hook_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_hook_proto_goTypes = []interface{}{
	(*HookBuildData)(nil),            // 0: proto.HookBuildData
	(*HookMetadata)(nil),             // 1: proto.HookMetadata
	(*Hook)(nil),                     // 2: proto.Hook
	(*InputOverWrite)(nil),           // 3: proto.InputOverWrite
	(*InputSource)(nil),              // 4: proto.InputSource
	(*Empty)(nil),                    // 5: proto.Empty
	(*GetPipelineTypeResponse)(nil),  // 6: proto.GetPipelineTypeResponse
	(*GetHooksMetaDataResponse)(nil), // 7: proto.GetHooksMetaDataResponse
	(*BuildHookRequest)(nil),         // 8: proto.BuildHookRequest
	(*BuildHookResponse)(nil),        // 9: proto.BuildHookResponse
	nil,                              // 10: proto.HookBuildData.UserVarsEntry
	nil,                              // 11: proto.HookBuildData.BuiltinVarsEntry
}
var file_hook_proto_depIdxs = []int32{
	10, // 0: proto.HookBuildData.UserVars:type_name -> proto.HookBuildData.UserVarsEntry
	11, // 1: proto.HookBuildData.BuiltinVars:type_name -> proto.HookBuildData.BuiltinVarsEntry
	3,  // 2: proto.Hook.RequestInputs:type_name -> proto.InputOverWrite
	4,  // 3: proto.InputOverWrite.Source:type_name -> proto.InputSource
	1,  // 4: proto.GetHooksMetaDataResponse.HooksMetaData:type_name -> proto.HookMetadata
	0,  // 5: proto.BuildHookRequest.Info:type_name -> proto.HookBuildData
	2,  // 6: proto.BuildHookResponse.Hook:type_name -> proto.Hook
	5,  // 7: proto.HookFactory.GetPipelineType:input_type -> proto.Empty
	5,  // 8: proto.HookFactory.GetHooksMetaData:input_type -> proto.Empty
	8,  // 9: proto.HookFactory.BuildHook:input_type -> proto.BuildHookRequest
	6,  // 10: proto.HookFactory.GetPipelineType:output_type -> proto.GetPipelineTypeResponse
	7,  // 11: proto.HookFactory.GetHooksMetaData:output_type -> proto.GetHooksMetaDataResponse
	9,  // 12: proto.HookFactory.BuildHook:output_type -> proto.BuildHookResponse
	10, // [10:13] is the sub-list for method output_type
	7,  // [7:10] is the sub-list for method input_type
	7,  // [7:7] is the sub-list for extension type_name
	7,  // [7:7] is the sub-list for extension extendee
	0,  // [0:7] is the sub-list for field type_name
}

func init() { file_hook_proto_init() }
func file_hook_proto_init() {
	if File_hook_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_hook_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HookBuildData); i {
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
		file_hook_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HookMetadata); i {
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
		file_hook_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Hook); i {
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
		file_hook_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InputOverWrite); i {
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
		file_hook_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InputSource); i {
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
		file_hook_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
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
		file_hook_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetPipelineTypeResponse); i {
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
		file_hook_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetHooksMetaDataResponse); i {
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
		file_hook_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BuildHookRequest); i {
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
		file_hook_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BuildHookResponse); i {
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
	file_hook_proto_msgTypes[1].OneofWrappers = []interface{}{}
	file_hook_proto_msgTypes[4].OneofWrappers = []interface{}{
		(*InputSource_FromEventSource)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_hook_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_hook_proto_goTypes,
		DependencyIndexes: file_hook_proto_depIdxs,
		MessageInfos:      file_hook_proto_msgTypes,
	}.Build()
	File_hook_proto = out.File
	file_hook_proto_rawDesc = nil
	file_hook_proto_goTypes = nil
	file_hook_proto_depIdxs = nil
}
