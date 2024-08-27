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

package controller

import (
	"context"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1alpha1 "github.com/muntashir-islam/k8s-operators/postgres-backup-azureblob-operator/api/v1alpha1"
)

// PostgresBackupReconciler reconciles a PostgresBackup object
type PostgresBackupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=database.muntashirislam.com,resources=postgresbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=database.muntashirislam.com,resources=postgresbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=database.muntashirislam.com,resources=postgresbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PostgresBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PostgresBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the PostgresBackup instance
	postgresBackup := &databasev1alpha1.PostgresBackup{}
	err := r.Get(ctx, req.NamespacedName, postgresBackup)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Perform backup and upload to Azure Blob Storage
	err = r.performBackup(ctx, postgresBackup)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Update the status
	postgresBackup.Status.LastBackupTime = metav1.Now()
	postgresBackup.Status.BackupStatus = "Success"
	err = r.Status().Update(ctx, postgresBackup)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 24 * time.Hour}, nil
}

func (r *PostgresBackupReconciler) performBackup(ctx context.Context, backup *databasev1alpha1.PostgresBackup) error {
	// Fetch the PostgreSQL password from the secret
	postgresPassword, err := r.getSecretValue(ctx, backup.Namespace, backup.Spec.PostgresSecret.Name, backup.Spec.PostgresSecret.Key)
	if err != nil {
		return err
	}

	// Fetch the Azure storage key from the secret
	azureStorageKey, err := r.getSecretValue(ctx, backup.Namespace, backup.Spec.AzureSecret.Name, backup.Spec.AzureSecret.Key)
	if err != nil {
		return err
	}

	// Define the backup file name
	backupFile := fmt.Sprintf("/tmp/%s.sql", backup.Name)

	// Define the pod that will run pg_dump
	pod := r.newPgDumpPod(backup, backupFile, postgresPassword)
	err = r.Create(ctx, pod)
	if err != nil {
		return err
	}

	// Wait for the pod to complete
	err = r.waitForPodCompletion(ctx, pod)
	if err != nil {
		return err
	}

	// Upload the backup to Azure Blob Storage
	err = r.uploadToAzureBlobStorage(ctx, backupFile, backup, azureStorageKey)
	if err != nil {
		return err
	}

	// Clean up the pod
	err = r.Delete(ctx, pod)
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresBackupReconciler) newPgDumpPod(backup *databasev1alpha1.PostgresBackup, backupFile, password string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pg-dump-%s", backup.Name),
			Namespace: backup.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "pg-dump",
					Image: "postgres:latest",
					Command: []string{
						"pg_dump",
						fmt.Sprintf("--host=%s", backup.Spec.Host),
						fmt.Sprintf("--port=%d", backup.Spec.Port),
						fmt.Sprintf("--username=%s", backup.Spec.User),
						fmt.Sprintf("--dbname=%s", backup.Spec.DBName),
						fmt.Sprintf("--file=%s", backupFile),
					},
					Env: []corev1.EnvVar{
						{
							Name:  "PGPASSWORD",
							Value: password,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// Set the owner reference to allow garbage collection
	err := ctrl.SetControllerReference(backup, pod, r.Scheme)
	if err != nil {
		return nil
	}
	return pod
}

func (r *PostgresBackupReconciler) waitForPodCompletion(ctx context.Context, pod *corev1.Pod) error {
	for {
		time.Sleep(5 * time.Second)
		err := r.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: pod.Name}, pod)
		if err != nil {
			return err
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			return nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			return fmt.Errorf("pod failed: %s", pod.Status.Message)
		}
	}
}

func (r *PostgresBackupReconciler) uploadToAzureBlobStorage(ctx context.Context, backupFile string, backup *databasev1alpha1.PostgresBackup, storageKey string) error {
	accountName := backup.Spec.StorageAccount
	containerName := backup.Spec.ContainerName

	credential, err := azblob.NewSharedKeyCredential(accountName, storageKey)
	if err != nil {
		return err
	}
	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	containerURL := azblob.NewContainerURL(*URL, pipeline)
	blobURL := containerURL.NewBlockBlobURL(fmt.Sprintf("%s.sql", backup.Name))

	file, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = azblob.UploadFileToBlockBlob(ctx, file, blobURL, azblob.UploadToBlockBlobOptions{
		BlockSize:   4 * 1024 * 1024,
		Parallelism: 16,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresBackupReconciler) getSecretValue(ctx context.Context, namespace, secretName, key string) (string, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretName}, secret)
	if err != nil {
		return "", err
	}

	value, exists := secret.Data[key]
	if !exists {
		return "", fmt.Errorf("key %s not found in secret %s", key, secretName)
	}

	return string(value), nil
}

func (r *PostgresBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.PostgresBackup{}).
		Complete(r)
}
