# postgres-backup-azureblob-operator
A Kubernetes Operator for Backing Up PostgreSQL Databases and Storing Backups in Azure Blob Storage

## Description
We can easily develop a program to take backups and run it as a cron job from a Kubernetes cluster. However, I want to achieve this in a more Kubernetes-native way.

Every 24 hours, this operator will execute with the help of defined CRDs (Custom Resource Definitions). It will create a PostgreSQL pod and run the pg_dump command to generate the backup file. Subsequently, the process will push the generated backup file into Azure Blob Storage.

For production run after deploying the deploy.yaml you have to create custom resource to provide connection profile for postgres db connection
```yaml
apiVersion: database.muntashirislam.com/v1alpha1
kind: PostgresBackup
metadata:
  name: example-postgresbackup
spec:
  host: <azure-postgres-host>
  port: 5432
  user: <your-db-username>
  dbName: <your-db-name>
  containerName: <your-container-name>
  storageAccount: <your-storage-account-name>
  postgresSecret:
    name: postgres-secret
    key: password
  azureSecret:
    name: azure-secret
    key: storage-key
```
Along with this you will aslo need two secrets to be present in the same namespaces
```yaml
# postgres-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: default
type: Opaque
data:
  password: <base64-encoded-password>

# azure-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: azure-secret
  namespace: default
type: Opaque
data:
  storage-key: <base64-encoded-storage-key>
```

or you can use following way

```shell
kubectl create secret generic postgres-secret --from-literal=password=$(echo -n "your-postgres-password" | base64)
kubectl create secret generic azure-secret --from-literal=storage-key=$(echo -n "your-storage-key" | base64)
```

## Getting Started Development RUN

### Prerequisites
- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/postgres-backup-azureblob-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/postgres-backup-azureblob-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Contributing
TBA

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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

