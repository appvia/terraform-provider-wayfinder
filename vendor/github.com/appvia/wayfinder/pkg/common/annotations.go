/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package common

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// WfAnnotation is wayfinder annotation
	WfAnnotation = "appvia.io"
)

// Annotation returns an annotation on a resource
func Annotation(tag string) string {
	return fmt.Sprintf("%s/%s", WfAnnotation, tag)
}

// NamedAnnotation returns a named annotation on a resource
func NamedAnnotation(name, tag string) string {
	return fmt.Sprintf("%s.%s/%s", name, WfAnnotation, tag)
}

// AnnotationPrefix returns an annotation prefix
func AnnotationPrefix() string {
	return fmt.Sprintf("%s/", WfAnnotation)
}

const (
	AnnotationValueTrue  = "true"
	AnnotationValueFalse = "false"
	// AnnotationValueScopePlatform is used to indicate that a particular resource is applicable at
	// a platform, not workspace, scope.
	AnnotationValueScopePlatform = "platform"
	// AnnotationValueScopeWorkspace is used to indicate that a particular resource is applicable at
	// a workspace, not platform, scope.
	AnnotationValueScopeWorkspace = "workspace"
)

var (
	// AnnotationReadOnly should be applied to any Wayfinder-generated resources which are managed
	// entirely by Wayfinder and must not be editable by users.
	AnnotationReadOnly = Annotation("readonly")
	// AnnotationSystem should be applied only to system-critical resources delivered directly by
	// Wayfinder - it should NOT be used for generated resources relating to user resources (use
	// AnnotationReadOnly for that)
	AnnotationSystem = Annotation("system")
	// AnnotationBuiltin should be applied to packages and global packages which are shipped
	// with Wayfinder
	AnnotationBuiltin = Annotation("builtin")
	// AnnotationInternal should be applied to any objects which are internal implementation details
	// of Wayfinder and not typically used by users directly (e.g. internal Wayfinder roles such as
	// 'anonymous' which are required for the system to function but would never be bound directly
	// by end users)
	AnnotationInternal = Annotation("internal")
	// AnnotationScope is used to provide an indication that a particular resource is applicable
	// at either platform or workspace scope. Used primarily to filter things on display to users.
	AnnotationScope = Annotation("scope")

	// AnnotationCreatedBy is automatically applied in our object handler to identify the user or
	// token that created the resource.
	AnnotationCreatedBy = Annotation("createdBy")
	// AnnotationLastUpdatedBy is automatically applied in our object handler to identify the user
	// or token that last changed the spec of the resource (or deleted it in the case of a deleted
	// object)
	AnnotationLastUpdatedBy = Annotation("lastUpdatedBy")
	// AnnotationLastUpdated is automatically applied in our object handler with the timestamp at
	// which this resource was last updated by a user
	AnnotationLastUpdated = Annotation("lastUpdated")

	// AnnotationSource describes an upstream source of an item, for information only - e.g. the
	// upstream terranetes cloud resource that a cloud resource plan was derived from
	AnnotationSource = Annotation("source")

	AnnotationAdoptionPolicy      = Annotation("adoptionPolicy")
	AnnotationAzureRoleName       = Annotation("azureRoleName")
	AnnotationClaimedBy           = Annotation("claimedBy")
	AnnotationChanged             = Annotation("changed")
	AnnotationCloudIdentity       = Annotation("cloudIdentity")
	AnnotationCloudRoleCredential = Annotation("cloudRoleCredential")
	AnnotationCloudRoleMeta       = Annotation("cloudRoleObjs")
	AnnotationSetupRolesLastRun   = Annotation("setupRolesLastRun")
	AnnotationDeletionPolicy      = Annotation("deletionPolicy")
	AnnotationDescription         = Annotation("description")
	AnnotationUniqueName          = Annotation("uniqueName")
	AnnotationEnabled             = Annotation("enabled")
	AnnotationExternalName        = Annotation("externalName")
	AnnotationInstallOnce         = Annotation("installOnce")
	AnnotationKeywords            = Annotation("keywords")
	AnnotationLastApplied         = Annotation("lastApplied")
	AnnotationOwner               = Annotation("owner")
	AnnotationPriority            = Annotation("priority")
	AnnotationRevision            = Annotation("revision")
	AnnotationSecretsProvider     = Annotation("secretsProvider")
	AnnotationSystemNamespace     = Annotation("systemNamespace")
	AnnotationType                = Annotation("type")
	AnnotationDependenciesUpdated = Annotation("dependenciesUpdated")
	AnnotationNoInfraReconcile    = Annotation("noInfraReconcile")
	AnnotationNoReconcile         = Annotation("noReconcile")
	AnnotationPackageVersion      = NamedAnnotation(PackagesPrefixName, "version")
	AnnotationPackageValuesEdited = NamedAnnotation(PackagesPrefixName, "values-edited")
	AnnotationChartIcon           = NamedAnnotation(PackagesPrefixName, "icon")

	// For controlling in-built defaults objects:

	// AnnotationDefaultsVersion will be applied by Wayfinder to all defaults when they are created
	// indicating which version of the defaults the object represents
	AnnotationDefaultsVersion = NamedAnnotation("defaults", "version")
	// AnnotationDefaultsUpdateAvailable will be applied by Wayfinder to any default which has been
	// frozen if the default is out of date with the latest version of that default
	AnnotationDefaultsUpdateAvailable = NamedAnnotation("defaults", "updateAvailable")
	// AnnotationDefaultsFrozen will cause a default not to be updated by Wayfinder any more until
	// the annotation is removed
	AnnotationDefaultsFrozen = NamedAnnotation("defaults", "frozen")
)

const (
	// AdoptionPolicyImport indicates we import a resource if it already exists
	AdoptionPolicyImport = "import"
	// AdoptionPolicyFail indicates we fail if a resource already exists
	AdoptionPolicyFail = "fail"
	// DeletionPolicyOrphan indicates we remove but do not delete
	DeletionPolicyOrphan = "orphan"
	// DeletionPolicyDelete indicates we remove and delete (default)
	DeletionPolicyDelete = "delete"
)

// Annotations are the list of known annotations
var Annotations = []string{
	AnnotationDeletionPolicy,
	AnnotationInstallOnce,
	AnnotationKeywords,
	AnnotationOwner,
	AnnotationPriority,
	AnnotationReadOnly,
	AnnotationSystem,
	AnnotationSystemNamespace,
	AnnotationCloudIdentity,
	AnnotationCloudRoleMeta,
	AnnotationPackageVersion,
}

// IsReadOnlyResource returns true if the given Kubernetes object is read-only
func IsReadOnlyResource(object metav1.Object) bool {
	return object.GetAnnotations()[AnnotationReadOnly] == AnnotationValueTrue
}

// IsSystemResource returns true if the given Kubernetes object is managed by Wayfinder
func IsSystemResource(object metav1.Object) bool {
	return object.GetAnnotations()[AnnotationSystem] == AnnotationValueTrue
}

// IsBuiltinResource returns true if the given Kubernetes object is shipped with Wayfinder
func IsBuiltInResource(object metav1.Object) bool {
	return object.GetAnnotations()[AnnotationBuiltin] == AnnotationValueTrue
}

// IsEnabled check if the object is enabled
func IsEnabled(object metav1.Object) bool {
	annotations := object.GetAnnotations()

	return annotations[AnnotationEnabled] != "false"
}
