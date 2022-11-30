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
)

var (
	// AnnotationReadOnly should be applied to any Wayfinder-generated resources which are managed
	// entirely by Wayfinder and must not be editable by users.
	AnnotationReadOnly = Annotation("readonly")
	// AnnotationSystem should be applied only to system-critical resources delivered directly by
	// Wayfinder - it should NOT be used for generated resources relating to user resources (use
	// AnnotationReadOnly for that)
	AnnotationSystem = Annotation("system")

	AnnotationAdoptionPolicy          = Annotation("adoptionPolicy")
	AnnotationAzureRoleName           = Annotation("azureRoleName")
	AnnotationClaimedBy               = Annotation("claimedBy")
	AnnotationCloudIdentity           = Annotation("cloudIdentity")
	AnnotationCloudRoleCredential     = Annotation("cloudRoleCredential")
	AnnotationCloudRoleMeta           = Annotation("cloudRoleObjs")
	AnnotationSetupRolesLastRun       = Annotation("setupRolesLastRun")
	AnnotationDeletionPolicy          = Annotation("deletionPolicy")
	AnnotationDescription             = Annotation("description")
	AnnotationEnabled                 = Annotation("enabled")
	AnnotationExternalName            = Annotation("externalName")
	AnnotationInstallOnce             = Annotation("installOnce")
	AnnotationKeywords                = Annotation("keywords")
	AnnotationLastApplied             = Annotation("lastApplied")
	AnnotationOwner                   = Annotation("owner")
	AnnotationPriority                = Annotation("priority")
	AnnotationRevision                = Annotation("revision")
	AnnotationSecretsProvider         = Annotation("secretsProvider")
	AnnotationSystemNamespace         = Annotation("systemNamespace")
	AnnotationType                    = Annotation("type")
	AnnotationDependenciesUpdated     = Annotation("dependenciesUpdated")
	AnnotationNoInfraReconcile        = Annotation("noInfraReconcile")
	AnnotationNoReconcile             = Annotation("noReconcile")
	AnnotationCapabilitiesName        = NamedAnnotation(CapabilitiesPrefixName, "name")
	AnnotationCapabilitiesDescription = NamedAnnotation(CapabilitiesPrefixName, "description")
	AnnotationsCapabilitiesReadOnly   = NamedAnnotation(CapabilitiesPrefixName, "readOnly")
	AnnotationPackageValuesEdited     = NamedAnnotation(PackagesPrefixName, "values-edited")
	AnnotationChartIcon               = NamedAnnotation(PackagesPrefixName, "icon")

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
	AnnotationCapabilitiesName,
	AnnotationCapabilitiesDescription,
}

// IsReadOnlyResource returns true if the given Kubernetes object is read-only
func IsReadOnlyResource(object metav1.Object) bool {
	return object.GetAnnotations()[AnnotationReadOnly] == AnnotationValueTrue
}

// IsSystemResource returns true if the given Kubernetes object is managed by Wayfinder
func IsSystemResource(object metav1.Object) bool {
	return object.GetAnnotations()[AnnotationSystem] == AnnotationValueTrue
}

// IsEnabled check if the object is enabled
func IsEnabled(object metav1.Object) bool {
	annotations := object.GetAnnotations()

	return annotations[AnnotationEnabled] != "false"
}
