// +build !ignore_autogenerated

// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Acme) DeepCopyInto(out *Acme) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Acme.
func (in *Acme) DeepCopy() *Acme {
	if in == nil {
		return nil
	}
	out := new(Acme)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CA) DeepCopyInto(out *CA) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CA.
func (in *CA) DeepCopy() *CA {
	if in == nil {
		return nil
	}
	out := new(CA)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CertManagerComponent) DeepCopyInto(out *CertManagerComponent) {
	*out = *in
	out.Certificate = in.Certificate
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CertManagerComponent.
func (in *CertManagerComponent) DeepCopy() *CertManagerComponent {
	if in == nil {
		return nil
	}
	out := new(CertManagerComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Certificate) DeepCopyInto(out *Certificate) {
	*out = *in
	out.Acme = in.Acme
	out.CA = in.CA
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Certificate.
func (in *Certificate) DeepCopy() *Certificate {
	if in == nil {
		return nil
	}
	out := new(Certificate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentSpec) DeepCopyInto(out *ComponentSpec) {
	*out = *in
	out.CertManager = in.CertManager
	out.DNS = in.DNS
	in.Ingress.DeepCopyInto(&out.Ingress)
	in.Istio.DeepCopyInto(&out.Istio)
	in.Keycloak.DeepCopyInto(&out.Keycloak)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentSpec.
func (in *ComponentSpec) DeepCopy() *ComponentSpec {
	if in == nil {
		return nil
	}
	out := new(ComponentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Condition.
func (in *Condition) DeepCopy() *Condition {
	if in == nil {
		return nil
	}
	out := new(Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DNSComponent) DeepCopyInto(out *DNSComponent) {
	*out = *in
	out.XIPIO = in.XIPIO
	out.OCI = in.OCI
	out.External = in.External
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNSComponent.
func (in *DNSComponent) DeepCopy() *DNSComponent {
	if in == nil {
		return nil
	}
	out := new(DNSComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *External) DeepCopyInto(out *External) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new External.
func (in *External) DeepCopy() *External {
	if in == nil {
		return nil
	}
	out := new(External)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressNginxComponent) DeepCopyInto(out *IngressNginxComponent) {
	*out = *in
	if in.NGINXInstallArgs != nil {
		in, out := &in.NGINXInstallArgs, &out.NGINXInstallArgs
		*out = make([]InstallArgs, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]v1.ServicePort, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressNginxComponent.
func (in *IngressNginxComponent) DeepCopy() *IngressNginxComponent {
	if in == nil {
		return nil
	}
	out := new(IngressNginxComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstallArgs) DeepCopyInto(out *InstallArgs) {
	*out = *in
	if in.ValueList != nil {
		in, out := &in.ValueList, &out.ValueList
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstallArgs.
func (in *InstallArgs) DeepCopy() *InstallArgs {
	if in == nil {
		return nil
	}
	out := new(InstallArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceInfo) DeepCopyInto(out *InstanceInfo) {
	*out = *in
	if in.Console != nil {
		in, out := &in.Console, &out.Console
		*out = new(string)
		**out = **in
	}
	if in.KeyCloakURL != nil {
		in, out := &in.KeyCloakURL, &out.KeyCloakURL
		*out = new(string)
		**out = **in
	}
	if in.RancherURL != nil {
		in, out := &in.RancherURL, &out.RancherURL
		*out = new(string)
		**out = **in
	}
	if in.ElasticURL != nil {
		in, out := &in.ElasticURL, &out.ElasticURL
		*out = new(string)
		**out = **in
	}
	if in.KibanaURL != nil {
		in, out := &in.KibanaURL, &out.KibanaURL
		*out = new(string)
		**out = **in
	}
	if in.GrafanaURL != nil {
		in, out := &in.GrafanaURL, &out.GrafanaURL
		*out = new(string)
		**out = **in
	}
	if in.PrometheusURL != nil {
		in, out := &in.PrometheusURL, &out.PrometheusURL
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceInfo.
func (in *InstanceInfo) DeepCopy() *InstanceInfo {
	if in == nil {
		return nil
	}
	out := new(InstanceInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IstioComponent) DeepCopyInto(out *IstioComponent) {
	*out = *in
	if in.IstioInstallArgs != nil {
		in, out := &in.IstioInstallArgs, &out.IstioInstallArgs
		*out = make([]InstallArgs, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IstioComponent.
func (in *IstioComponent) DeepCopy() *IstioComponent {
	if in == nil {
		return nil
	}
	out := new(IstioComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeycloakComponent) DeepCopyInto(out *KeycloakComponent) {
	*out = *in
	if in.KeycloakInstallArgs != nil {
		in, out := &in.KeycloakInstallArgs, &out.KeycloakInstallArgs
		*out = make([]InstallArgs, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.MySQL.DeepCopyInto(&out.MySQL)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeycloakComponent.
func (in *KeycloakComponent) DeepCopy() *KeycloakComponent {
	if in == nil {
		return nil
	}
	out := new(KeycloakComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MySQLComponent) DeepCopyInto(out *MySQLComponent) {
	*out = *in
	if in.MySQLInstallArgs != nil {
		in, out := &in.MySQLInstallArgs, &out.MySQLInstallArgs
		*out = make([]InstallArgs, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeSource != nil {
		in, out := &in.VolumeSource, &out.VolumeSource
		*out = new(v1.VolumeSource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MySQLComponent.
func (in *MySQLComponent) DeepCopy() *MySQLComponent {
	if in == nil {
		return nil
	}
	out := new(MySQLComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCI) DeepCopyInto(out *OCI) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCI.
func (in *OCI) DeepCopy() *OCI {
	if in == nil {
		return nil
	}
	out := new(OCI)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RoleBindingSubject) DeepCopyInto(out *RoleBindingSubject) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RoleBindingSubject.
func (in *RoleBindingSubject) DeepCopy() *RoleBindingSubject {
	if in == nil {
		return nil
	}
	out := new(RoleBindingSubject)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecuritySpec) DeepCopyInto(out *SecuritySpec) {
	*out = *in
	out.AdminBinding = in.AdminBinding
	out.MonitorBinding = in.MonitorBinding
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecuritySpec.
func (in *SecuritySpec) DeepCopy() *SecuritySpec {
	if in == nil {
		return nil
	}
	out := new(SecuritySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Verrazzano) DeepCopyInto(out *Verrazzano) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Verrazzano.
func (in *Verrazzano) DeepCopy() *Verrazzano {
	if in == nil {
		return nil
	}
	out := new(Verrazzano)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Verrazzano) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoList) DeepCopyInto(out *VerrazzanoList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Verrazzano, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoList.
func (in *VerrazzanoList) DeepCopy() *VerrazzanoList {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoSpec) DeepCopyInto(out *VerrazzanoSpec) {
	*out = *in
	in.Components.DeepCopyInto(&out.Components)
	out.Security = in.Security
	if in.DefaultVolumeSource != nil {
		in, out := &in.DefaultVolumeSource, &out.DefaultVolumeSource
		*out = new(v1.VolumeSource)
		(*in).DeepCopyInto(*out)
	}
	if in.VolumeClaimSpecTemplates != nil {
		in, out := &in.VolumeClaimSpecTemplates, &out.VolumeClaimSpecTemplates
		*out = make([]VolumeClaimSpecTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoSpec.
func (in *VerrazzanoSpec) DeepCopy() *VerrazzanoSpec {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoStatus) DeepCopyInto(out *VerrazzanoStatus) {
	*out = *in
	if in.VerrazzanoInstance != nil {
		in, out := &in.VerrazzanoInstance, &out.VerrazzanoInstance
		*out = new(InstanceInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoStatus.
func (in *VerrazzanoStatus) DeepCopy() *VerrazzanoStatus {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeClaimSpecTemplate) DeepCopyInto(out *VolumeClaimSpecTemplate) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeClaimSpecTemplate.
func (in *VolumeClaimSpecTemplate) DeepCopy() *VolumeClaimSpecTemplate {
	if in == nil {
		return nil
	}
	out := new(VolumeClaimSpecTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XIPIO) DeepCopyInto(out *XIPIO) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XIPIO.
func (in *XIPIO) DeepCopy() *XIPIO {
	if in == nil {
		return nil
	}
	out := new(XIPIO)
	in.DeepCopyInto(out)
	return out
}
