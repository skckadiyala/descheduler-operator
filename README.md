# Descheduling

Descheduling involves evicting pods based on the defined policies, so that pods can be rescheduled onto more appropriate nodes.

Benifits of descheduling and rescheduling are
- Balancing pods in Under/Over utilized Nodes  
- Node failure caused pods to be moved other available nodes, descheduler balances the pods when the new nodes are added to the cluster.

## descheduler-operator
This operator developed using operator-sdk to run descheduler on kuberneties.

Objects created and deployed to the cluster
- CustomResourceDefinition (CRD), 
- Roles, 
- Role Binding 
- Service Accounts 
- Operator

**Use helm to deploy descheduler operator**

```
helm upgrade --install descheduler-operator descheduler-operator 

kubectl apply -f deploy/crds/descheduler_v1alpha1_descheduler_cr.yaml

```

Deschdeular operator watches for the deschdeuler Customer Resource (CR), when CR is applied/modified 
The operator creates/updates the deschdeuler configmap and 
creates a new cronjob to run the deschdeuler. The Cronjob creates creates a descheduler job as per configured schedule.

The configmap is created from the CR object, whenever there is change in the CR object the descheduler operator is responsible for identifying changes and updating the configmap. Also in few cases operatort deletes the current running cronjob and creates a new cronjob with the updated flags.


**Delete Descheduler Operator**
```
kubectl delete -f deploy/crds/descheduler_v1alpha1_descheduler_cr.yaml
helm delete --purge descheduler-operator
kubectl delete jobs --all
```
