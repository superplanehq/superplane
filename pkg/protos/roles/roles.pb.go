// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v3.15.8
// source: roles.proto

package roles

import (
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	authorization "github.com/superplanehq/superplane/pkg/protos/authorization"
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

type AssignRoleRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	RoleName      string                   `protobuf:"bytes,3,opt,name=role_name,json=roleName,proto3" json:"role_name,omitempty"`
	UserId        string                   `protobuf:"bytes,4,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	UserEmail     string                   `protobuf:"bytes,5,opt,name=user_email,json=userEmail,proto3" json:"user_email,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AssignRoleRequest) Reset() {
	*x = AssignRoleRequest{}
	mi := &file_roles_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AssignRoleRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AssignRoleRequest) ProtoMessage() {}

func (x *AssignRoleRequest) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AssignRoleRequest.ProtoReflect.Descriptor instead.
func (*AssignRoleRequest) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{0}
}

func (x *AssignRoleRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *AssignRoleRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *AssignRoleRequest) GetRoleName() string {
	if x != nil {
		return x.RoleName
	}
	return ""
}

func (x *AssignRoleRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *AssignRoleRequest) GetUserEmail() string {
	if x != nil {
		return x.UserEmail
	}
	return ""
}

type AssignRoleResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AssignRoleResponse) Reset() {
	*x = AssignRoleResponse{}
	mi := &file_roles_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AssignRoleResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AssignRoleResponse) ProtoMessage() {}

func (x *AssignRoleResponse) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AssignRoleResponse.ProtoReflect.Descriptor instead.
func (*AssignRoleResponse) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{1}
}

type RemoveRoleRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	RoleName      string                   `protobuf:"bytes,3,opt,name=role_name,json=roleName,proto3" json:"role_name,omitempty"`
	UserId        string                   `protobuf:"bytes,4,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	UserEmail     string                   `protobuf:"bytes,5,opt,name=user_email,json=userEmail,proto3" json:"user_email,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RemoveRoleRequest) Reset() {
	*x = RemoveRoleRequest{}
	mi := &file_roles_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RemoveRoleRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveRoleRequest) ProtoMessage() {}

func (x *RemoveRoleRequest) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveRoleRequest.ProtoReflect.Descriptor instead.
func (*RemoveRoleRequest) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{2}
}

func (x *RemoveRoleRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *RemoveRoleRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *RemoveRoleRequest) GetRoleName() string {
	if x != nil {
		return x.RoleName
	}
	return ""
}

func (x *RemoveRoleRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *RemoveRoleRequest) GetUserEmail() string {
	if x != nil {
		return x.UserEmail
	}
	return ""
}

type RemoveRoleResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RemoveRoleResponse) Reset() {
	*x = RemoveRoleResponse{}
	mi := &file_roles_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RemoveRoleResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveRoleResponse) ProtoMessage() {}

func (x *RemoveRoleResponse) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveRoleResponse.ProtoReflect.Descriptor instead.
func (*RemoveRoleResponse) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{3}
}

type ListRolesRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListRolesRequest) Reset() {
	*x = ListRolesRequest{}
	mi := &file_roles_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListRolesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRolesRequest) ProtoMessage() {}

func (x *ListRolesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListRolesRequest.ProtoReflect.Descriptor instead.
func (*ListRolesRequest) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{4}
}

func (x *ListRolesRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *ListRolesRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

type ListRolesResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Roles         []*Role                `protobuf:"bytes,1,rep,name=roles,proto3" json:"roles,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListRolesResponse) Reset() {
	*x = ListRolesResponse{}
	mi := &file_roles_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListRolesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRolesResponse) ProtoMessage() {}

func (x *ListRolesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListRolesResponse.ProtoReflect.Descriptor instead.
func (*ListRolesResponse) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{5}
}

func (x *ListRolesResponse) GetRoles() []*Role {
	if x != nil {
		return x.Roles
	}
	return nil
}

type DescribeRoleRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	Role          string                   `protobuf:"bytes,3,opt,name=role,proto3" json:"role,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DescribeRoleRequest) Reset() {
	*x = DescribeRoleRequest{}
	mi := &file_roles_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DescribeRoleRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DescribeRoleRequest) ProtoMessage() {}

func (x *DescribeRoleRequest) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DescribeRoleRequest.ProtoReflect.Descriptor instead.
func (*DescribeRoleRequest) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{6}
}

func (x *DescribeRoleRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *DescribeRoleRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *DescribeRoleRequest) GetRole() string {
	if x != nil {
		return x.Role
	}
	return ""
}

type DescribeRoleResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Role          *Role                  `protobuf:"bytes,1,opt,name=role,proto3" json:"role,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DescribeRoleResponse) Reset() {
	*x = DescribeRoleResponse{}
	mi := &file_roles_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DescribeRoleResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DescribeRoleResponse) ProtoMessage() {}

func (x *DescribeRoleResponse) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DescribeRoleResponse.ProtoReflect.Descriptor instead.
func (*DescribeRoleResponse) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{7}
}

func (x *DescribeRoleResponse) GetRole() *Role {
	if x != nil {
		return x.Role
	}
	return nil
}

type CreateRoleRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	Role          *Role                    `protobuf:"bytes,3,opt,name=role,proto3" json:"role,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateRoleRequest) Reset() {
	*x = CreateRoleRequest{}
	mi := &file_roles_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateRoleRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateRoleRequest) ProtoMessage() {}

func (x *CreateRoleRequest) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateRoleRequest.ProtoReflect.Descriptor instead.
func (*CreateRoleRequest) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{8}
}

func (x *CreateRoleRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *CreateRoleRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *CreateRoleRequest) GetRole() *Role {
	if x != nil {
		return x.Role
	}
	return nil
}

type CreateRoleResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Role          *Role                  `protobuf:"bytes,1,opt,name=role,proto3" json:"role,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateRoleResponse) Reset() {
	*x = CreateRoleResponse{}
	mi := &file_roles_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateRoleResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateRoleResponse) ProtoMessage() {}

func (x *CreateRoleResponse) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateRoleResponse.ProtoReflect.Descriptor instead.
func (*CreateRoleResponse) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{9}
}

func (x *CreateRoleResponse) GetRole() *Role {
	if x != nil {
		return x.Role
	}
	return nil
}

type UpdateRoleRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	RoleName      string                   `protobuf:"bytes,3,opt,name=role_name,json=roleName,proto3" json:"role_name,omitempty"`
	Role          *Role                    `protobuf:"bytes,4,opt,name=role,proto3" json:"role,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdateRoleRequest) Reset() {
	*x = UpdateRoleRequest{}
	mi := &file_roles_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateRoleRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateRoleRequest) ProtoMessage() {}

func (x *UpdateRoleRequest) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateRoleRequest.ProtoReflect.Descriptor instead.
func (*UpdateRoleRequest) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{10}
}

func (x *UpdateRoleRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *UpdateRoleRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *UpdateRoleRequest) GetRoleName() string {
	if x != nil {
		return x.RoleName
	}
	return ""
}

func (x *UpdateRoleRequest) GetRole() *Role {
	if x != nil {
		return x.Role
	}
	return nil
}

type UpdateRoleResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Role          *Role                  `protobuf:"bytes,1,opt,name=role,proto3" json:"role,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdateRoleResponse) Reset() {
	*x = UpdateRoleResponse{}
	mi := &file_roles_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateRoleResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateRoleResponse) ProtoMessage() {}

func (x *UpdateRoleResponse) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateRoleResponse.ProtoReflect.Descriptor instead.
func (*UpdateRoleResponse) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{11}
}

func (x *UpdateRoleResponse) GetRole() *Role {
	if x != nil {
		return x.Role
	}
	return nil
}

type DeleteRoleRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	RoleName      string                   `protobuf:"bytes,3,opt,name=role_name,json=roleName,proto3" json:"role_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteRoleRequest) Reset() {
	*x = DeleteRoleRequest{}
	mi := &file_roles_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteRoleRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRoleRequest) ProtoMessage() {}

func (x *DeleteRoleRequest) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteRoleRequest.ProtoReflect.Descriptor instead.
func (*DeleteRoleRequest) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{12}
}

func (x *DeleteRoleRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *DeleteRoleRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *DeleteRoleRequest) GetRoleName() string {
	if x != nil {
		return x.RoleName
	}
	return ""
}

type DeleteRoleResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteRoleResponse) Reset() {
	*x = DeleteRoleResponse{}
	mi := &file_roles_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteRoleResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRoleResponse) ProtoMessage() {}

func (x *DeleteRoleResponse) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteRoleResponse.ProtoReflect.Descriptor instead.
func (*DeleteRoleResponse) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{13}
}

type Role struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Metadata      *Role_Metadata         `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Spec          *Role_Spec             `protobuf:"bytes,2,opt,name=spec,proto3" json:"spec,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role) Reset() {
	*x = Role{}
	mi := &file_roles_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role) ProtoMessage() {}

func (x *Role) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role.ProtoReflect.Descriptor instead.
func (*Role) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{14}
}

func (x *Role) GetMetadata() *Role_Metadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Role) GetSpec() *Role_Spec {
	if x != nil {
		return x.Spec
	}
	return nil
}

type Role_Metadata struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	Name          string                   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	DomainType    authorization.DomainType `protobuf:"varint,2,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,3,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	CreatedAt     *timestamp.Timestamp     `protobuf:"bytes,4,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt     *timestamp.Timestamp     `protobuf:"bytes,5,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Metadata) Reset() {
	*x = Role_Metadata{}
	mi := &file_roles_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Metadata) ProtoMessage() {}

func (x *Role_Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Metadata.ProtoReflect.Descriptor instead.
func (*Role_Metadata) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{14, 0}
}

func (x *Role_Metadata) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Role_Metadata) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *Role_Metadata) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *Role_Metadata) GetCreatedAt() *timestamp.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

func (x *Role_Metadata) GetUpdatedAt() *timestamp.Timestamp {
	if x != nil {
		return x.UpdatedAt
	}
	return nil
}

type Role_Spec struct {
	state         protoimpl.MessageState      `protogen:"open.v1"`
	DisplayName   string                      `protobuf:"bytes,1,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	Description   string                      `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	Permissions   []*authorization.Permission `protobuf:"bytes,3,rep,name=permissions,proto3" json:"permissions,omitempty"`
	InheritedRole *Role                       `protobuf:"bytes,4,opt,name=inherited_role,json=inheritedRole,proto3" json:"inherited_role,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Spec) Reset() {
	*x = Role_Spec{}
	mi := &file_roles_proto_msgTypes[16]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Spec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Spec) ProtoMessage() {}

func (x *Role_Spec) ProtoReflect() protoreflect.Message {
	mi := &file_roles_proto_msgTypes[16]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Spec.ProtoReflect.Descriptor instead.
func (*Role_Spec) Descriptor() ([]byte, []int) {
	return file_roles_proto_rawDescGZIP(), []int{14, 1}
}

func (x *Role_Spec) GetDisplayName() string {
	if x != nil {
		return x.DisplayName
	}
	return ""
}

func (x *Role_Spec) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *Role_Spec) GetPermissions() []*authorization.Permission {
	if x != nil {
		return x.Permissions
	}
	return nil
}

func (x *Role_Spec) GetInheritedRole() *Role {
	if x != nil {
		return x.InheritedRole
	}
	return nil
}

var File_roles_proto protoreflect.FileDescriptor

const file_roles_proto_rawDesc = "" +
	"\n" +
	"\vroles.proto\x12\x10Superplane.Roles\x1a\x1cgoogle/api/annotations.proto\x1a\x1fgoogle/protobuf/timestamp.proto\x1a.protoc-gen-openapiv2/options/annotations.proto\x1a\x13authorization.proto\"\xcc\x01\n" +
	"\x11AssignRoleRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\x12\x1b\n" +
	"\trole_name\x18\x03 \x01(\tR\broleName\x12\x17\n" +
	"\auser_id\x18\x04 \x01(\tR\x06userId\x12\x1d\n" +
	"\n" +
	"user_email\x18\x05 \x01(\tR\tuserEmail\"\x14\n" +
	"\x12AssignRoleResponse\"\xcc\x01\n" +
	"\x11RemoveRoleRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\x12\x1b\n" +
	"\trole_name\x18\x03 \x01(\tR\broleName\x12\x17\n" +
	"\auser_id\x18\x04 \x01(\tR\x06userId\x12\x1d\n" +
	"\n" +
	"user_email\x18\x05 \x01(\tR\tuserEmail\"\x14\n" +
	"\x12RemoveRoleResponse\"v\n" +
	"\x10ListRolesRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\"A\n" +
	"\x11ListRolesResponse\x12,\n" +
	"\x05roles\x18\x01 \x03(\v2\x16.Superplane.Roles.RoleR\x05roles\"\x8d\x01\n" +
	"\x13DescribeRoleRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\x12\x12\n" +
	"\x04role\x18\x03 \x01(\tR\x04role\"B\n" +
	"\x14DescribeRoleResponse\x12*\n" +
	"\x04role\x18\x01 \x01(\v2\x16.Superplane.Roles.RoleR\x04role\"\xa3\x01\n" +
	"\x11CreateRoleRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\x12*\n" +
	"\x04role\x18\x03 \x01(\v2\x16.Superplane.Roles.RoleR\x04role\"@\n" +
	"\x12CreateRoleResponse\x12*\n" +
	"\x04role\x18\x01 \x01(\v2\x16.Superplane.Roles.RoleR\x04role\"\xc0\x01\n" +
	"\x11UpdateRoleRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\x12\x1b\n" +
	"\trole_name\x18\x03 \x01(\tR\broleName\x12*\n" +
	"\x04role\x18\x04 \x01(\v2\x16.Superplane.Roles.RoleR\x04role\"@\n" +
	"\x12UpdateRoleResponse\x12*\n" +
	"\x04role\x18\x01 \x01(\v2\x16.Superplane.Roles.RoleR\x04role\"\x94\x01\n" +
	"\x11DeleteRoleRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\x12\x1b\n" +
	"\trole_name\x18\x03 \x01(\tR\broleName\"\x14\n" +
	"\x12DeleteRoleResponse\"\xc4\x04\n" +
	"\x04Role\x12;\n" +
	"\bmetadata\x18\x01 \x01(\v2\x1f.Superplane.Roles.Role.MetadataR\bmetadata\x12/\n" +
	"\x04spec\x18\x02 \x01(\v2\x1b.Superplane.Roles.Role.SpecR\x04spec\x1a\xf8\x01\n" +
	"\bMetadata\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12E\n" +
	"\vdomain_type\x18\x02 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x03 \x01(\tR\bdomainId\x129\n" +
	"\n" +
	"created_at\x18\x04 \x01(\v2\x1a.google.protobuf.TimestampR\tcreatedAt\x129\n" +
	"\n" +
	"updated_at\x18\x05 \x01(\v2\x1a.google.protobuf.TimestampR\tupdatedAt\x1a\xd2\x01\n" +
	"\x04Spec\x12!\n" +
	"\fdisplay_name\x18\x01 \x01(\tR\vdisplayName\x12 \n" +
	"\vdescription\x18\x02 \x01(\tR\vdescription\x12F\n" +
	"\vpermissions\x18\x03 \x03(\v2$.Superplane.Authorization.PermissionR\vpermissions\x12=\n" +
	"\x0einherited_role\x18\x04 \x01(\v2\x16.Superplane.Roles.RoleR\rinheritedRole2\x98\v\n" +
	"\x05Roles\x12\xb9\x01\n" +
	"\n" +
	"AssignRole\x12#.Superplane.Roles.AssignRoleRequest\x1a$.Superplane.Roles.AssignRoleResponse\"`\x92A>\n" +
	"\x05Roles\x12\vAssign role\x1a(Assigns a role to a user within a domain\x82\xd3\xe4\x93\x02\x19:\x01*2\x14/api/v1/roles/assign\x12\xbb\x01\n" +
	"\n" +
	"RemoveRole\x12#.Superplane.Roles.RemoveRoleRequest\x1a$.Superplane.Roles.RemoveRoleResponse\"b\x92A@\n" +
	"\x05Roles\x12\vRemove role\x1a*Removes a role from a user within a domain\x82\xd3\xe4\x93\x02\x19:\x01*2\x14/api/v1/roles/remove\x12\xdd\x01\n" +
	"\tListRoles\x12\".Superplane.Roles.ListRolesRequest\x1a#.Superplane.Roles.ListRolesResponse\"\x86\x01\x92An\n" +
	"\x05Roles\x12\n" +
	"List roles\x1aYReturns available roles for a specific domain type with their permissions and inheritance\x82\xd3\xe4\x93\x02\x0f\x12\r/api/v1/roles\x12\xf1\x01\n" +
	"\fDescribeRole\x12%.Superplane.Roles.DescribeRoleRequest\x1a&.Superplane.Roles.DescribeRoleResponse\"\x91\x01\x92Ap\n" +
	"\x05Roles\x12\rDescribe role\x1aXReturns detailed information about a specific role including permissions and inheritance\x82\xd3\xe4\x93\x02\x18\x12\x16/api/v1/roles/describe\x12\xbe\x01\n" +
	"\n" +
	"CreateRole\x12#.Superplane.Roles.CreateRoleRequest\x1a$.Superplane.Roles.CreateRoleResponse\"e\x92AJ\n" +
	"\x05Roles\x12\vCreate role\x1a4Creates a new custom role with specified permissions\x82\xd3\xe4\x93\x02\x12:\x01*\"\r/api/v1/roles\x12\xca\x01\n" +
	"\n" +
	"UpdateRole\x12#.Superplane.Roles.UpdateRoleRequest\x1a$.Superplane.Roles.UpdateRoleResponse\"q\x92AJ\n" +
	"\x05Roles\x12\vUpdate role\x1a4Updates an existing custom role with new permissions\x82\xd3\xe4\x93\x02\x1e:\x01*\x1a\x19/api/v1/roles/{role_name}\x12\xb2\x01\n" +
	"\n" +
	"DeleteRole\x12#.Superplane.Roles.DeleteRoleRequest\x1a$.Superplane.Roles.DeleteRoleResponse\"Y\x92A5\n" +
	"\x05Roles\x12\vDelete role\x1a\x1fDeletes an existing custom role\x82\xd3\xe4\x93\x02\x1b*\x19/api/v1/roles/{role_name}B\xbf\x01\x92A\x86\x01\x12\\\n" +
	"\x14Superplane Roles API\x12\x18API for Superplane Roles\"%\n" +
	"\vAPI Support\x1a\x16support@superplane.com2\x031.0*\x02\x01\x022\x10application/json:\x10application/jsonZ3github.com/superplanehq/superplane/pkg/protos/rolesb\x06proto3"

var (
	file_roles_proto_rawDescOnce sync.Once
	file_roles_proto_rawDescData []byte
)

func file_roles_proto_rawDescGZIP() []byte {
	file_roles_proto_rawDescOnce.Do(func() {
		file_roles_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_roles_proto_rawDesc), len(file_roles_proto_rawDesc)))
	})
	return file_roles_proto_rawDescData
}

var file_roles_proto_msgTypes = make([]protoimpl.MessageInfo, 17)
var file_roles_proto_goTypes = []any{
	(*AssignRoleRequest)(nil),        // 0: Superplane.Roles.AssignRoleRequest
	(*AssignRoleResponse)(nil),       // 1: Superplane.Roles.AssignRoleResponse
	(*RemoveRoleRequest)(nil),        // 2: Superplane.Roles.RemoveRoleRequest
	(*RemoveRoleResponse)(nil),       // 3: Superplane.Roles.RemoveRoleResponse
	(*ListRolesRequest)(nil),         // 4: Superplane.Roles.ListRolesRequest
	(*ListRolesResponse)(nil),        // 5: Superplane.Roles.ListRolesResponse
	(*DescribeRoleRequest)(nil),      // 6: Superplane.Roles.DescribeRoleRequest
	(*DescribeRoleResponse)(nil),     // 7: Superplane.Roles.DescribeRoleResponse
	(*CreateRoleRequest)(nil),        // 8: Superplane.Roles.CreateRoleRequest
	(*CreateRoleResponse)(nil),       // 9: Superplane.Roles.CreateRoleResponse
	(*UpdateRoleRequest)(nil),        // 10: Superplane.Roles.UpdateRoleRequest
	(*UpdateRoleResponse)(nil),       // 11: Superplane.Roles.UpdateRoleResponse
	(*DeleteRoleRequest)(nil),        // 12: Superplane.Roles.DeleteRoleRequest
	(*DeleteRoleResponse)(nil),       // 13: Superplane.Roles.DeleteRoleResponse
	(*Role)(nil),                     // 14: Superplane.Roles.Role
	(*Role_Metadata)(nil),            // 15: Superplane.Roles.Role.Metadata
	(*Role_Spec)(nil),                // 16: Superplane.Roles.Role.Spec
	(authorization.DomainType)(0),    // 17: Superplane.Authorization.DomainType
	(*timestamp.Timestamp)(nil),      // 18: google.protobuf.Timestamp
	(*authorization.Permission)(nil), // 19: Superplane.Authorization.Permission
}
var file_roles_proto_depIdxs = []int32{
	17, // 0: Superplane.Roles.AssignRoleRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	17, // 1: Superplane.Roles.RemoveRoleRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	17, // 2: Superplane.Roles.ListRolesRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	14, // 3: Superplane.Roles.ListRolesResponse.roles:type_name -> Superplane.Roles.Role
	17, // 4: Superplane.Roles.DescribeRoleRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	14, // 5: Superplane.Roles.DescribeRoleResponse.role:type_name -> Superplane.Roles.Role
	17, // 6: Superplane.Roles.CreateRoleRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	14, // 7: Superplane.Roles.CreateRoleRequest.role:type_name -> Superplane.Roles.Role
	14, // 8: Superplane.Roles.CreateRoleResponse.role:type_name -> Superplane.Roles.Role
	17, // 9: Superplane.Roles.UpdateRoleRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	14, // 10: Superplane.Roles.UpdateRoleRequest.role:type_name -> Superplane.Roles.Role
	14, // 11: Superplane.Roles.UpdateRoleResponse.role:type_name -> Superplane.Roles.Role
	17, // 12: Superplane.Roles.DeleteRoleRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	15, // 13: Superplane.Roles.Role.metadata:type_name -> Superplane.Roles.Role.Metadata
	16, // 14: Superplane.Roles.Role.spec:type_name -> Superplane.Roles.Role.Spec
	17, // 15: Superplane.Roles.Role.Metadata.domain_type:type_name -> Superplane.Authorization.DomainType
	18, // 16: Superplane.Roles.Role.Metadata.created_at:type_name -> google.protobuf.Timestamp
	18, // 17: Superplane.Roles.Role.Metadata.updated_at:type_name -> google.protobuf.Timestamp
	19, // 18: Superplane.Roles.Role.Spec.permissions:type_name -> Superplane.Authorization.Permission
	14, // 19: Superplane.Roles.Role.Spec.inherited_role:type_name -> Superplane.Roles.Role
	0,  // 20: Superplane.Roles.Roles.AssignRole:input_type -> Superplane.Roles.AssignRoleRequest
	2,  // 21: Superplane.Roles.Roles.RemoveRole:input_type -> Superplane.Roles.RemoveRoleRequest
	4,  // 22: Superplane.Roles.Roles.ListRoles:input_type -> Superplane.Roles.ListRolesRequest
	6,  // 23: Superplane.Roles.Roles.DescribeRole:input_type -> Superplane.Roles.DescribeRoleRequest
	8,  // 24: Superplane.Roles.Roles.CreateRole:input_type -> Superplane.Roles.CreateRoleRequest
	10, // 25: Superplane.Roles.Roles.UpdateRole:input_type -> Superplane.Roles.UpdateRoleRequest
	12, // 26: Superplane.Roles.Roles.DeleteRole:input_type -> Superplane.Roles.DeleteRoleRequest
	1,  // 27: Superplane.Roles.Roles.AssignRole:output_type -> Superplane.Roles.AssignRoleResponse
	3,  // 28: Superplane.Roles.Roles.RemoveRole:output_type -> Superplane.Roles.RemoveRoleResponse
	5,  // 29: Superplane.Roles.Roles.ListRoles:output_type -> Superplane.Roles.ListRolesResponse
	7,  // 30: Superplane.Roles.Roles.DescribeRole:output_type -> Superplane.Roles.DescribeRoleResponse
	9,  // 31: Superplane.Roles.Roles.CreateRole:output_type -> Superplane.Roles.CreateRoleResponse
	11, // 32: Superplane.Roles.Roles.UpdateRole:output_type -> Superplane.Roles.UpdateRoleResponse
	13, // 33: Superplane.Roles.Roles.DeleteRole:output_type -> Superplane.Roles.DeleteRoleResponse
	27, // [27:34] is the sub-list for method output_type
	20, // [20:27] is the sub-list for method input_type
	20, // [20:20] is the sub-list for extension type_name
	20, // [20:20] is the sub-list for extension extendee
	0,  // [0:20] is the sub-list for field type_name
}

func init() { file_roles_proto_init() }
func file_roles_proto_init() {
	if File_roles_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_roles_proto_rawDesc), len(file_roles_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   17,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_roles_proto_goTypes,
		DependencyIndexes: file_roles_proto_depIdxs,
		MessageInfos:      file_roles_proto_msgTypes,
	}.Build()
	File_roles_proto = out.File
	file_roles_proto_goTypes = nil
	file_roles_proto_depIdxs = nil
}
