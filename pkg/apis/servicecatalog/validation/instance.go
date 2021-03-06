/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	sc "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog"
	"github.com/kubernetes-incubator/service-catalog/pkg/controller"
)

// validateServiceInstanceName is the validation function for Instance names.
var validateServiceInstanceName = apivalidation.NameIsDNSSubdomain

// ValidateServiceInstance validates an Instance and returns a list of errors.
func ValidateServiceInstance(instance *sc.ServiceInstance) field.ErrorList {
	return internalValidateServiceInstance(instance, true)
}

func internalValidateServiceInstance(instance *sc.ServiceInstance, create bool) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&instance.ObjectMeta, true, /*namespace*/
		validateServiceInstanceName,
		field.NewPath("metadata"))...)
	allErrs = append(allErrs, validateServiceInstanceSpec(&instance.Spec, field.NewPath("Spec"), create)...)
	allErrs = append(allErrs, validateServiceInstanceStatus(&instance.Status, field.NewPath("Status"), create)...)
	return allErrs
}

func validateServiceInstanceSpec(spec *sc.ServiceInstanceSpec, fldPath *field.Path, create bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if "" == spec.ExternalServiceClassName {
		allErrs = append(allErrs, field.Required(fldPath.Child("externalServiceClassName"), "externalServiceClassName is required"))
	}

	for _, msg := range validateServiceClassName(spec.ExternalServiceClassName, false /* prefix */) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("externalServiceClassName"), spec.ExternalServiceClassName, msg))
	}

	if "" == spec.ExternalServicePlanName {
		allErrs = append(allErrs, field.Required(fldPath.Child("externalServicePlanName"), "externalServicePlanName is required"))
	}

	for _, msg := range validateServicePlanName(spec.ExternalServicePlanName, false /* prefix */) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("externalServicePlanName"), spec.ExternalServicePlanName, msg))
	}

	if spec.ParametersFrom != nil {
		for _, paramsFrom := range spec.ParametersFrom {
			if paramsFrom.SecretKeyRef != nil {
				if paramsFrom.SecretKeyRef.Name == "" {
					allErrs = append(allErrs, field.Required(fldPath.Child("parametersFrom.secretKeyRef.name"), "name is required"))
				}
				if paramsFrom.SecretKeyRef.Key == "" {
					allErrs = append(allErrs, field.Required(fldPath.Child("parametersFrom.secretKeyRef.key"), "key is required"))
				}
			} else {
				allErrs = append(allErrs, field.Required(fldPath.Child("parametersFrom"), "source must not be empty if present"))
			}
		}
	}
	if spec.Parameters != nil {
		if len(spec.Parameters.Raw) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("parameters"), "inline parameters must not be empty if present"))
		}
		if _, err := controller.UnmarshalRawParameters(spec.Parameters.Raw); err != nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("parameters"), "invalid inline parameters"))
		}
	}

	return allErrs
}

func validateServiceInstanceStatus(spec *sc.ServiceInstanceStatus, fldPath *field.Path, create bool) field.ErrorList {
	errors := field.ErrorList{}
	// TODO(vaikas): Implement more comprehensive status validation.
	// https://github.com/kubernetes-incubator/service-catalog/issues/882

	// Do not allow the instance to be ready if an async operation is ongoing
	// ongoing
	if spec.AsyncOpInProgress {
		for _, c := range spec.Conditions {
			if c.Type == sc.ServiceInstanceConditionReady && c.Status == sc.ConditionTrue {
				errors = append(errors, field.Forbidden(fldPath.Child("Conditions"), "Can not set ServiceInstanceConditionReady to true when an async operation is in progress"))
			}
		}
	}

	return errors
}

// internalValidateServiceInstanceUpdateAllowed ensures there is not a
// pending update on-going with the spec of the instance before allowing an update
// to the spec to go through.
func internalValidateServiceInstanceUpdateAllowed(new *sc.ServiceInstance, old *sc.ServiceInstance) field.ErrorList {
	errors := field.ErrorList{}
	if old.Generation != new.Generation && old.Status.ReconciledGeneration != old.Generation {
		errors = append(errors, field.Forbidden(field.NewPath("spec"), "Another update for this service instance is in progress"))
	}
	return errors
}

// ValidateServiceInstanceUpdate validates a change to the Instance's spec.
func ValidateServiceInstanceUpdate(new *sc.ServiceInstance, old *sc.ServiceInstance) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, internalValidateServiceInstanceUpdateAllowed(new, old)...)
	allErrs = append(allErrs, internalValidateServiceInstance(new, false)...)
	return allErrs
}

func internalValidateServiceInstanceStatusUpdateAllowed(new *sc.ServiceInstance, old *sc.ServiceInstance) field.ErrorList {
	errors := field.ErrorList{}
	// TODO(vaikas): Are there any cases where we do not allow updates to
	// Status during Async updates in progress?
	return errors
}

// ValidateServiceInstanceStatusUpdate checks that when changing from an older
// instance to a newer instance is okay. This only checks the instance.Status field.
func ValidateServiceInstanceStatusUpdate(new *sc.ServiceInstance, old *sc.ServiceInstance) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, internalValidateServiceInstanceStatusUpdateAllowed(new, old)...)
	allErrs = append(allErrs, internalValidateServiceInstance(new, false)...)
	return allErrs
}
