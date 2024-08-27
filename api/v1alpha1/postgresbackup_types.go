/*
Copyright 2024.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresBackupSpec defines the desired state of PostgresBackup
type PostgresBackupSpec struct {
	Host           string       `json:"host"`
	Port           int32        `json:"port"`
	User           string       `json:"user"`
	DBName         string       `json:"dbName"`
	ContainerName  string       `json:"containerName"`
	StorageAccount string       `json:"storageAccount"`
	PostgresSecret SecretKeyRef `json:"postgresSecret"`
	AzureSecret    SecretKeyRef `json:"azureSecret"`
}

// SecretKeyRef contains information to locate a secret in the same namespace.
type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// PostgresBackupStatus defines the observed state of PostgresBackup
type PostgresBackupStatus struct {
	LastBackupTime metav1.Time `json:"lastBackupTime,omitempty"`
	BackupStatus   string      `json:"backupStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PostgresBackup is the Schema for the postgresbackups API
type PostgresBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresBackupSpec   `json:"spec,omitempty"`
	Status PostgresBackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostgresBackupList contains a list of PostgresBackup
type PostgresBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresBackup{}, &PostgresBackupList{})
}
