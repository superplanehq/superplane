// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v3.15.8
// source: users.proto

package users

import (
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	authorization "github.com/superplanehq/superplane/pkg/protos/authorization"
	roles "github.com/superplanehq/superplane/pkg/protos/roles"
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

type ListUserPermissionsRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	UserId        string                   `protobuf:"bytes,3,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListUserPermissionsRequest) Reset() {
	*x = ListUserPermissionsRequest{}
	mi := &file_users_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListUserPermissionsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUserPermissionsRequest) ProtoMessage() {}

func (x *ListUserPermissionsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUserPermissionsRequest.ProtoReflect.Descriptor instead.
func (*ListUserPermissionsRequest) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{0}
}

func (x *ListUserPermissionsRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *ListUserPermissionsRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *ListUserPermissionsRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

type ListUserPermissionsResponse struct {
	state         protoimpl.MessageState      `protogen:"open.v1"`
	UserId        string                      `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	DomainType    authorization.DomainType    `protobuf:"varint,2,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                      `protobuf:"bytes,3,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	Permissions   []*authorization.Permission `protobuf:"bytes,4,rep,name=permissions,proto3" json:"permissions,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListUserPermissionsResponse) Reset() {
	*x = ListUserPermissionsResponse{}
	mi := &file_users_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListUserPermissionsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUserPermissionsResponse) ProtoMessage() {}

func (x *ListUserPermissionsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUserPermissionsResponse.ProtoReflect.Descriptor instead.
func (*ListUserPermissionsResponse) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{1}
}

func (x *ListUserPermissionsResponse) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *ListUserPermissionsResponse) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *ListUserPermissionsResponse) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *ListUserPermissionsResponse) GetPermissions() []*authorization.Permission {
	if x != nil {
		return x.Permissions
	}
	return nil
}

type ListUserRolesRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	UserId        string                   `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	DomainType    authorization.DomainType `protobuf:"varint,2,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,3,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListUserRolesRequest) Reset() {
	*x = ListUserRolesRequest{}
	mi := &file_users_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListUserRolesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUserRolesRequest) ProtoMessage() {}

func (x *ListUserRolesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUserRolesRequest.ProtoReflect.Descriptor instead.
func (*ListUserRolesRequest) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{2}
}

func (x *ListUserRolesRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *ListUserRolesRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *ListUserRolesRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

type ListUserRolesResponse struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	UserId        string                   `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	DomainType    authorization.DomainType `protobuf:"varint,2,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,3,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	Roles         []*roles.Role            `protobuf:"bytes,4,rep,name=roles,proto3" json:"roles,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListUserRolesResponse) Reset() {
	*x = ListUserRolesResponse{}
	mi := &file_users_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListUserRolesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUserRolesResponse) ProtoMessage() {}

func (x *ListUserRolesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUserRolesResponse.ProtoReflect.Descriptor instead.
func (*ListUserRolesResponse) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{3}
}

func (x *ListUserRolesResponse) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *ListUserRolesResponse) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *ListUserRolesResponse) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *ListUserRolesResponse) GetRoles() []*roles.Role {
	if x != nil {
		return x.Roles
	}
	return nil
}

type ListUsersRequest struct {
	state         protoimpl.MessageState   `protogen:"open.v1"`
	DomainType    authorization.DomainType `protobuf:"varint,1,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId      string                   `protobuf:"bytes,2,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListUsersRequest) Reset() {
	*x = ListUsersRequest{}
	mi := &file_users_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListUsersRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUsersRequest) ProtoMessage() {}

func (x *ListUsersRequest) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUsersRequest.ProtoReflect.Descriptor instead.
func (*ListUsersRequest) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{4}
}

func (x *ListUsersRequest) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *ListUsersRequest) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

type ListUsersResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Users         []*User                `protobuf:"bytes,1,rep,name=users,proto3" json:"users,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListUsersResponse) Reset() {
	*x = ListUsersResponse{}
	mi := &file_users_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListUsersResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListUsersResponse) ProtoMessage() {}

func (x *ListUsersResponse) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListUsersResponse.ProtoReflect.Descriptor instead.
func (*ListUsersResponse) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{5}
}

func (x *ListUsersResponse) GetUsers() []*User {
	if x != nil {
		return x.Users
	}
	return nil
}

// User data structure
type User struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Metadata      *User_Metadata         `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Spec          *User_Spec             `protobuf:"bytes,2,opt,name=spec,proto3" json:"spec,omitempty"`
	Status        *User_Status           `protobuf:"bytes,3,opt,name=status,proto3" json:"status,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User) Reset() {
	*x = User{}
	mi := &file_users_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User.ProtoReflect.Descriptor instead.
func (*User) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{6}
}

func (x *User) GetMetadata() *User_Metadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *User) GetSpec() *User_Spec {
	if x != nil {
		return x.Spec
	}
	return nil
}

func (x *User) GetStatus() *User_Status {
	if x != nil {
		return x.Status
	}
	return nil
}

type UserRoleAssignment struct {
	state           protoimpl.MessageState   `protogen:"open.v1"`
	RoleName        string                   `protobuf:"bytes,1,opt,name=role_name,json=roleName,proto3" json:"role_name,omitempty"`
	RoleDisplayName string                   `protobuf:"bytes,2,opt,name=role_display_name,json=roleDisplayName,proto3" json:"role_display_name,omitempty"`
	RoleDescription string                   `protobuf:"bytes,3,opt,name=role_description,json=roleDescription,proto3" json:"role_description,omitempty"`
	DomainType      authorization.DomainType `protobuf:"varint,4,opt,name=domain_type,json=domainType,proto3,enum=Superplane.Authorization.DomainType" json:"domain_type,omitempty"`
	DomainId        string                   `protobuf:"bytes,5,opt,name=domain_id,json=domainId,proto3" json:"domain_id,omitempty"`
	AssignedAt      *timestamp.Timestamp     `protobuf:"bytes,6,opt,name=assigned_at,json=assignedAt,proto3" json:"assigned_at,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *UserRoleAssignment) Reset() {
	*x = UserRoleAssignment{}
	mi := &file_users_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRoleAssignment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRoleAssignment) ProtoMessage() {}

func (x *UserRoleAssignment) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRoleAssignment.ProtoReflect.Descriptor instead.
func (*UserRoleAssignment) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{7}
}

func (x *UserRoleAssignment) GetRoleName() string {
	if x != nil {
		return x.RoleName
	}
	return ""
}

func (x *UserRoleAssignment) GetRoleDisplayName() string {
	if x != nil {
		return x.RoleDisplayName
	}
	return ""
}

func (x *UserRoleAssignment) GetRoleDescription() string {
	if x != nil {
		return x.RoleDescription
	}
	return ""
}

func (x *UserRoleAssignment) GetDomainType() authorization.DomainType {
	if x != nil {
		return x.DomainType
	}
	return authorization.DomainType(0)
}

func (x *UserRoleAssignment) GetDomainId() string {
	if x != nil {
		return x.DomainId
	}
	return ""
}

func (x *UserRoleAssignment) GetAssignedAt() *timestamp.Timestamp {
	if x != nil {
		return x.AssignedAt
	}
	return nil
}

type AccountProvider struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ProviderType  string                 `protobuf:"bytes,1,opt,name=provider_type,json=providerType,proto3" json:"provider_type,omitempty"` // e.g., "google", "github", "email"
	ProviderId    string                 `protobuf:"bytes,2,opt,name=provider_id,json=providerId,proto3" json:"provider_id,omitempty"`       // unique ID from the provider
	Email         string                 `protobuf:"bytes,3,opt,name=email,proto3" json:"email,omitempty"`
	DisplayName   string                 `protobuf:"bytes,4,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	AvatarUrl     string                 `protobuf:"bytes,5,opt,name=avatar_url,json=avatarUrl,proto3" json:"avatar_url,omitempty"`
	IsPrimary     bool                   `protobuf:"varint,6,opt,name=is_primary,json=isPrimary,proto3" json:"is_primary,omitempty"`
	CreatedAt     *timestamp.Timestamp   `protobuf:"bytes,7,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt     *timestamp.Timestamp   `protobuf:"bytes,8,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AccountProvider) Reset() {
	*x = AccountProvider{}
	mi := &file_users_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AccountProvider) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AccountProvider) ProtoMessage() {}

func (x *AccountProvider) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AccountProvider.ProtoReflect.Descriptor instead.
func (*AccountProvider) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{8}
}

func (x *AccountProvider) GetProviderType() string {
	if x != nil {
		return x.ProviderType
	}
	return ""
}

func (x *AccountProvider) GetProviderId() string {
	if x != nil {
		return x.ProviderId
	}
	return ""
}

func (x *AccountProvider) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *AccountProvider) GetDisplayName() string {
	if x != nil {
		return x.DisplayName
	}
	return ""
}

func (x *AccountProvider) GetAvatarUrl() string {
	if x != nil {
		return x.AvatarUrl
	}
	return ""
}

func (x *AccountProvider) GetIsPrimary() bool {
	if x != nil {
		return x.IsPrimary
	}
	return false
}

func (x *AccountProvider) GetCreatedAt() *timestamp.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

func (x *AccountProvider) GetUpdatedAt() *timestamp.Timestamp {
	if x != nil {
		return x.UpdatedAt
	}
	return nil
}

type User_Metadata struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Email         string                 `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
	CreatedAt     *timestamp.Timestamp   `protobuf:"bytes,3,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt     *timestamp.Timestamp   `protobuf:"bytes,4,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Metadata) Reset() {
	*x = User_Metadata{}
	mi := &file_users_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Metadata) ProtoMessage() {}

func (x *User_Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Metadata.ProtoReflect.Descriptor instead.
func (*User_Metadata) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{6, 0}
}

func (x *User_Metadata) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *User_Metadata) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *User_Metadata) GetCreatedAt() *timestamp.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

func (x *User_Metadata) GetUpdatedAt() *timestamp.Timestamp {
	if x != nil {
		return x.UpdatedAt
	}
	return nil
}

type User_Spec struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	DisplayName      string                 `protobuf:"bytes,1,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	AvatarUrl        string                 `protobuf:"bytes,2,opt,name=avatar_url,json=avatarUrl,proto3" json:"avatar_url,omitempty"`
	AccountProviders []*AccountProvider     `protobuf:"bytes,3,rep,name=account_providers,json=accountProviders,proto3" json:"account_providers,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *User_Spec) Reset() {
	*x = User_Spec{}
	mi := &file_users_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Spec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Spec) ProtoMessage() {}

func (x *User_Spec) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Spec.ProtoReflect.Descriptor instead.
func (*User_Spec) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{6, 1}
}

func (x *User_Spec) GetDisplayName() string {
	if x != nil {
		return x.DisplayName
	}
	return ""
}

func (x *User_Spec) GetAvatarUrl() string {
	if x != nil {
		return x.AvatarUrl
	}
	return ""
}

func (x *User_Spec) GetAccountProviders() []*AccountProvider {
	if x != nil {
		return x.AccountProviders
	}
	return nil
}

type User_Status struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	IsActive        bool                   `protobuf:"varint,1,opt,name=is_active,json=isActive,proto3" json:"is_active,omitempty"`
	RoleAssignments []*UserRoleAssignment  `protobuf:"bytes,2,rep,name=role_assignments,json=roleAssignments,proto3" json:"role_assignments,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *User_Status) Reset() {
	*x = User_Status{}
	mi := &file_users_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Status) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Status) ProtoMessage() {}

func (x *User_Status) ProtoReflect() protoreflect.Message {
	mi := &file_users_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Status.ProtoReflect.Descriptor instead.
func (*User_Status) Descriptor() ([]byte, []int) {
	return file_users_proto_rawDescGZIP(), []int{6, 2}
}

func (x *User_Status) GetIsActive() bool {
	if x != nil {
		return x.IsActive
	}
	return false
}

func (x *User_Status) GetRoleAssignments() []*UserRoleAssignment {
	if x != nil {
		return x.RoleAssignments
	}
	return nil
}

var File_users_proto protoreflect.FileDescriptor

const file_users_proto_rawDesc = "" +
	"\n" +
	"\vusers.proto\x12\x10Superplane.Users\x1a\x1cgoogle/api/annotations.proto\x1a\x1fgoogle/protobuf/timestamp.proto\x1a.protoc-gen-openapiv2/options/annotations.proto\x1a\x13authorization.proto\x1a\vroles.proto\"\x99\x01\n" +
	"\x1aListUserPermissionsRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\x12\x17\n" +
	"\auser_id\x18\x03 \x01(\tR\x06userId\"\xe2\x01\n" +
	"\x1bListUserPermissionsResponse\x12\x17\n" +
	"\auser_id\x18\x01 \x01(\tR\x06userId\x12E\n" +
	"\vdomain_type\x18\x02 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x03 \x01(\tR\bdomainId\x12F\n" +
	"\vpermissions\x18\x04 \x03(\v2$.Superplane.Authorization.PermissionR\vpermissions\"\x93\x01\n" +
	"\x14ListUserRolesRequest\x12\x17\n" +
	"\auser_id\x18\x01 \x01(\tR\x06userId\x12E\n" +
	"\vdomain_type\x18\x02 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x03 \x01(\tR\bdomainId\"\xc2\x01\n" +
	"\x15ListUserRolesResponse\x12\x17\n" +
	"\auser_id\x18\x01 \x01(\tR\x06userId\x12E\n" +
	"\vdomain_type\x18\x02 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x03 \x01(\tR\bdomainId\x12,\n" +
	"\x05roles\x18\x04 \x03(\v2\x16.Superplane.Roles.RoleR\x05roles\"v\n" +
	"\x10ListUsersRequest\x12E\n" +
	"\vdomain_type\x18\x01 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x02 \x01(\tR\bdomainId\"A\n" +
	"\x11ListUsersResponse\x12,\n" +
	"\x05users\x18\x01 \x03(\v2\x16.Superplane.Users.UserR\x05users\"\xe7\x04\n" +
	"\x04User\x12;\n" +
	"\bmetadata\x18\x01 \x01(\v2\x1f.Superplane.Users.User.MetadataR\bmetadata\x12/\n" +
	"\x04spec\x18\x02 \x01(\v2\x1b.Superplane.Users.User.SpecR\x04spec\x125\n" +
	"\x06status\x18\x03 \x01(\v2\x1d.Superplane.Users.User.StatusR\x06status\x1a\xa6\x01\n" +
	"\bMetadata\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x14\n" +
	"\x05email\x18\x02 \x01(\tR\x05email\x129\n" +
	"\n" +
	"created_at\x18\x03 \x01(\v2\x1a.google.protobuf.TimestampR\tcreatedAt\x129\n" +
	"\n" +
	"updated_at\x18\x04 \x01(\v2\x1a.google.protobuf.TimestampR\tupdatedAt\x1a\x98\x01\n" +
	"\x04Spec\x12!\n" +
	"\fdisplay_name\x18\x01 \x01(\tR\vdisplayName\x12\x1d\n" +
	"\n" +
	"avatar_url\x18\x02 \x01(\tR\tavatarUrl\x12N\n" +
	"\x11account_providers\x18\x03 \x03(\v2!.Superplane.Users.AccountProviderR\x10accountProviders\x1av\n" +
	"\x06Status\x12\x1b\n" +
	"\tis_active\x18\x01 \x01(\bR\bisActive\x12O\n" +
	"\x10role_assignments\x18\x02 \x03(\v2$.Superplane.Users.UserRoleAssignmentR\x0froleAssignments\"\xa9\x02\n" +
	"\x12UserRoleAssignment\x12\x1b\n" +
	"\trole_name\x18\x01 \x01(\tR\broleName\x12*\n" +
	"\x11role_display_name\x18\x02 \x01(\tR\x0froleDisplayName\x12)\n" +
	"\x10role_description\x18\x03 \x01(\tR\x0froleDescription\x12E\n" +
	"\vdomain_type\x18\x04 \x01(\x0e2$.Superplane.Authorization.DomainTypeR\n" +
	"domainType\x12\x1b\n" +
	"\tdomain_id\x18\x05 \x01(\tR\bdomainId\x12;\n" +
	"\vassigned_at\x18\x06 \x01(\v2\x1a.google.protobuf.TimestampR\n" +
	"assignedAt\"\xc4\x02\n" +
	"\x0fAccountProvider\x12#\n" +
	"\rprovider_type\x18\x01 \x01(\tR\fproviderType\x12\x1f\n" +
	"\vprovider_id\x18\x02 \x01(\tR\n" +
	"providerId\x12\x14\n" +
	"\x05email\x18\x03 \x01(\tR\x05email\x12!\n" +
	"\fdisplay_name\x18\x04 \x01(\tR\vdisplayName\x12\x1d\n" +
	"\n" +
	"avatar_url\x18\x05 \x01(\tR\tavatarUrl\x12\x1d\n" +
	"\n" +
	"is_primary\x18\x06 \x01(\bR\tisPrimary\x129\n" +
	"\n" +
	"created_at\x18\a \x01(\v2\x1a.google.protobuf.TimestampR\tcreatedAt\x129\n" +
	"\n" +
	"updated_at\x18\b \x01(\v2\x1a.google.protobuf.TimestampR\tupdatedAt2\x9a\x05\n" +
	"\x05Users\x12\xfe\x01\n" +
	"\x13ListUserPermissions\x12,.Superplane.Users.ListUserPermissionsRequest\x1a-.Superplane.Users.ListUserPermissionsResponse\"\x89\x01\x92A[\n" +
	"\x05Users\x12\x15List user permissions\x1a;Returns all permissions a user has within a specific domain\x82\xd3\xe4\x93\x02%\x12#/api/v1/users/{user_id}/permissions\x12\xd8\x01\n" +
	"\rListUserRoles\x12&.Superplane.Users.ListUserRolesRequest\x1a'.Superplane.Users.ListUserRolesResponse\"v\x92AN\n" +
	"\x05Users\x12\x0eGet user roles\x1a5Returns the roles a user has within a specific domain\x82\xd3\xe4\x93\x02\x1f\x12\x1d/api/v1/users/{user_id}/roles\x12\xb4\x01\n" +
	"\tListUsers\x12\".Superplane.Users.ListUsersRequest\x1a#.Superplane.Users.ListUsersResponse\"^\x92AF\n" +
	"\x05Users\x12\n" +
	"List users\x1a1Returns all users that have roles within a domain\x82\xd3\xe4\x93\x02\x0f\x12\r/api/v1/usersB\xbf\x01\x92A\x86\x01\x12\\\n" +
	"\x14Superplane Users API\x12\x18API for Superplane Users\"%\n" +
	"\vAPI Support\x1a\x16support@superplane.com2\x031.0*\x02\x01\x022\x10application/json:\x10application/jsonZ3github.com/superplanehq/superplane/pkg/protos/usersb\x06proto3"

var (
	file_users_proto_rawDescOnce sync.Once
	file_users_proto_rawDescData []byte
)

func file_users_proto_rawDescGZIP() []byte {
	file_users_proto_rawDescOnce.Do(func() {
		file_users_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_users_proto_rawDesc), len(file_users_proto_rawDesc)))
	})
	return file_users_proto_rawDescData
}

var file_users_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_users_proto_goTypes = []any{
	(*ListUserPermissionsRequest)(nil),  // 0: Superplane.Users.ListUserPermissionsRequest
	(*ListUserPermissionsResponse)(nil), // 1: Superplane.Users.ListUserPermissionsResponse
	(*ListUserRolesRequest)(nil),        // 2: Superplane.Users.ListUserRolesRequest
	(*ListUserRolesResponse)(nil),       // 3: Superplane.Users.ListUserRolesResponse
	(*ListUsersRequest)(nil),            // 4: Superplane.Users.ListUsersRequest
	(*ListUsersResponse)(nil),           // 5: Superplane.Users.ListUsersResponse
	(*User)(nil),                        // 6: Superplane.Users.User
	(*UserRoleAssignment)(nil),          // 7: Superplane.Users.UserRoleAssignment
	(*AccountProvider)(nil),             // 8: Superplane.Users.AccountProvider
	(*User_Metadata)(nil),               // 9: Superplane.Users.User.Metadata
	(*User_Spec)(nil),                   // 10: Superplane.Users.User.Spec
	(*User_Status)(nil),                 // 11: Superplane.Users.User.Status
	(authorization.DomainType)(0),       // 12: Superplane.Authorization.DomainType
	(*authorization.Permission)(nil),    // 13: Superplane.Authorization.Permission
	(*roles.Role)(nil),                  // 14: Superplane.Roles.Role
	(*timestamp.Timestamp)(nil),         // 15: google.protobuf.Timestamp
}
var file_users_proto_depIdxs = []int32{
	12, // 0: Superplane.Users.ListUserPermissionsRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	12, // 1: Superplane.Users.ListUserPermissionsResponse.domain_type:type_name -> Superplane.Authorization.DomainType
	13, // 2: Superplane.Users.ListUserPermissionsResponse.permissions:type_name -> Superplane.Authorization.Permission
	12, // 3: Superplane.Users.ListUserRolesRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	12, // 4: Superplane.Users.ListUserRolesResponse.domain_type:type_name -> Superplane.Authorization.DomainType
	14, // 5: Superplane.Users.ListUserRolesResponse.roles:type_name -> Superplane.Roles.Role
	12, // 6: Superplane.Users.ListUsersRequest.domain_type:type_name -> Superplane.Authorization.DomainType
	6,  // 7: Superplane.Users.ListUsersResponse.users:type_name -> Superplane.Users.User
	9,  // 8: Superplane.Users.User.metadata:type_name -> Superplane.Users.User.Metadata
	10, // 9: Superplane.Users.User.spec:type_name -> Superplane.Users.User.Spec
	11, // 10: Superplane.Users.User.status:type_name -> Superplane.Users.User.Status
	12, // 11: Superplane.Users.UserRoleAssignment.domain_type:type_name -> Superplane.Authorization.DomainType
	15, // 12: Superplane.Users.UserRoleAssignment.assigned_at:type_name -> google.protobuf.Timestamp
	15, // 13: Superplane.Users.AccountProvider.created_at:type_name -> google.protobuf.Timestamp
	15, // 14: Superplane.Users.AccountProvider.updated_at:type_name -> google.protobuf.Timestamp
	15, // 15: Superplane.Users.User.Metadata.created_at:type_name -> google.protobuf.Timestamp
	15, // 16: Superplane.Users.User.Metadata.updated_at:type_name -> google.protobuf.Timestamp
	8,  // 17: Superplane.Users.User.Spec.account_providers:type_name -> Superplane.Users.AccountProvider
	7,  // 18: Superplane.Users.User.Status.role_assignments:type_name -> Superplane.Users.UserRoleAssignment
	0,  // 19: Superplane.Users.Users.ListUserPermissions:input_type -> Superplane.Users.ListUserPermissionsRequest
	2,  // 20: Superplane.Users.Users.ListUserRoles:input_type -> Superplane.Users.ListUserRolesRequest
	4,  // 21: Superplane.Users.Users.ListUsers:input_type -> Superplane.Users.ListUsersRequest
	1,  // 22: Superplane.Users.Users.ListUserPermissions:output_type -> Superplane.Users.ListUserPermissionsResponse
	3,  // 23: Superplane.Users.Users.ListUserRoles:output_type -> Superplane.Users.ListUserRolesResponse
	5,  // 24: Superplane.Users.Users.ListUsers:output_type -> Superplane.Users.ListUsersResponse
	22, // [22:25] is the sub-list for method output_type
	19, // [19:22] is the sub-list for method input_type
	19, // [19:19] is the sub-list for extension type_name
	19, // [19:19] is the sub-list for extension extendee
	0,  // [0:19] is the sub-list for field type_name
}

func init() { file_users_proto_init() }
func file_users_proto_init() {
	if File_users_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_users_proto_rawDesc), len(file_users_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_users_proto_goTypes,
		DependencyIndexes: file_users_proto_depIdxs,
		MessageInfos:      file_users_proto_msgTypes,
	}.Build()
	File_users_proto = out.File
	file_users_proto_goTypes = nil
	file_users_proto_depIdxs = nil
}
