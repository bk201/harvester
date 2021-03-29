I record notes and thoughts about my Hackweek 21 topic [Leveraging Ceph in the Harvester project](https://hackweek.suse.com/20/projects/leveraging-ceph-in-the-harvester-project) here.

Table of Contents
=================

   * [Installation](#installation)
      * [Prepare a Kubernetes cluster](#prepare-a-kubernetes-cluster)
      * [Deploy](#deploy)
         * [Clone this project and switch to hackweek20 branch](#clone-this-project-and-switch-to-hackweek20-branch)
         * [Export your k8s config path to the KUBECONFIG environment variable](#export-your-k8s-config-path-to-the-kubeconfig-environment-variable)
         * [Deploy Harvester helm chart](#deploy-harvester-helm-chart)
         * [Deploy Rook-Ceph](#deploy-rook-ceph)
         * [Create a storage class](#create-a-storage-class)
   * [Other thoughts](#other-thoughts)
      * [Use the CDI "Smart Clone" feature to provision VM disks](#use-the-cdi-smart-clone-feature-to-provision-vm-disks)
         * [Deploy VMs with smart clone feature](#deploy-vms-with-smart-clone-feature)
            * [Enable RBD snapshot](#enable-rbd-snapshot)
            * [Create a DataVolume and fetch an VM image:](#create-a-datavolume-and-fetch-an-vm-image)
            * [Turn on the smart clone feature in KubeVirt:](#turn-on-the-smart-clone-feature-in-kubevirt)
            * [Create a VM with smart clone feature.](#create-a-vm-with-smart-clone-feature)
         * [Possible downsides](#possible-downsides)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc)

# Installation

Assume the workspace is located at `~/hackweek21` directory.

## Prepare a Kubernetes cluster

I use Vagrant to deploy a 4-node k3s cluster for Ceph. 

**Note**: Each node should at least has a spare disk (>= 10GB).

## Deploy

### Clone this project and switch to `hackweek20` branch

```
% cd ~/hackweek21
$ git clone https://github.com/bk201/harvester
$ cd harvester && git checkout hackweek21
```

### Export your k8s config path to the `KUBECONFIG` environment variable
### Deploy Harvester helm chart

```
$ cd deploy/charts
$ helm install harvester harvester --namespace harvester-system --create-namespace

## Monitor created pods
$ kubectl get pods -A -w
```

If everything goes fine, the Harvester's Dashboard should be accessed from NodePort service after a while:

```
https://<one_of_node_ip>:30443
```

### Deploy Rook-Ceph

Clone Rook project:

```
$ cd ~/hackweek21
$ git clone https://github.com/bk201/rook
$ git clone https://github.com/rook/rook
$ cd rook && git checkout hackweek21
```

Apply Rool-Ceph CRDs and create the operator:

```
$ cd cluster/examples/kubernetes/ceph
$ kubectl create -f crds.yaml -f common.yaml -f operator.yaml
```

Create a Rook-Ceph cluster

> :warning: **WARNING**: **All** spare disks on k3s nodes will be formatted and used as Ceph OSDs.

```
$ kubectl create -f cluster.yaml
```

Create a toolbox pod
```
$ kubectl create -f toolbox.yaml
```

After a while, a Ceph cluster should be provisioned.

```
$ kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- bash

# Wait until `health` is `Healthy`
$ ceph -s

# To view OSD tree
$ ceph osd tree
```

### Create a storage class

Create a storage class named `rook-ceph` to provision volumes in an RBD pool.

```
$ kubectl create -f storageclass.yaml
```

At this point, we should be able to deploy VMs with the harvester UI. And the VMs will use RBD images as their disks.

# Other thoughts

## Use the CDI "Smart Clone" feature to provision VM disks

By default, KubeVirt ask CDI to fetch VM images from external sources and write them to persistent volumes:

![import-image-many-times](./docs/hackweek/hackweek20-fetch%20single%20image.png)

If the user provision 10 VMs with the same image, then the following overhead can be spotted:

- CDI fetches the same image 10 times. Convert the same image 10 times (if the source image is not in RAW format).
- Image in the storage is not "thin-provisioned", it occupied 10 times of storage space it needs.

![import-single-image](./docs/hackweek/hackweek20-fetch%20multiple%20images.png)

We can use Ceph RBD's snapshot and clone feature as a remedy:

- CDI fetch and convert VM image to a "golden volume".
- CDI does smart clone: snapshot the golden volume and create a clone from it. The operation is considered "light-weight" since only metadata are involved.
- Boot new the new VM.

![Smart clone](./docs/hackweek/hackweek20-smart%20clone.png)

For more information about CDI smart clone, please check this [link](https://github.com/kubevirt/containerized-data-importer/blob/main/doc/smart-clone.md).

### Deploy VMs with smart clone feature

Unfortunately, at the time of writing, I don't have enough knowledge to modify the frontend to support this feature. I'll record how to do it by applying KubeVirt YAML files.

#### Enable RBD snapshot

```
$ cd ~/hackweek20
$ cd rook
$ kubectl create -f cluster/examples/kubernetes/ceph/csi/rbd/snapshotclass.yaml
```

For more information, check this [link](https://rook.github.io/docs/rook/v1.5/ceph-csi-snapshot.html#rbd-snapshots).

#### Create a DataVolume and fetch an VM image:

```
$ cd ~/hackweek20
$ cd harvester/docs/hackweek20
$ kubectl create -f dv.yaml

# a `cirros` DataVolume should be created
$ kubectl get dv
NAME              PHASE       PROGRESS   RESTARTS   AGE
cirros            Succeeded   100.0%                10s
```

**NOTE**

For an unknown reason, the cdi-importer always hits a segfault during image importing. I mirror the cirros image to my local HTTP server and modify the `url` field in `dv.yaml` file, then everything goes fine.

#### Turn on the smart clone feature in KubeVirt:

```
kubectl create -f enable-smart-clone.yaml
```

#### Create a VM with smart clone feature.

```
kubectl create -f vm-dv-clone.yaml
```

In the `dataVolumeTemplates` spec, we specify a PVC as the source. The CDI importer is smart enough to know the volume can be created from an RBD snapshot.

```yaml
dataVolumeTemplates
...
      source:
        pvc:
          name: cirros
          namespace: default
```




### Possible downsides

- Longhorn volume might provide data locality in some cases. E.g., the read replica is on the VM node.

- CDI creates a new snapshot for each new volume. This can be eliminated.

