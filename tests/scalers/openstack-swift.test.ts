import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'
import test from 'ava';

const testNamespace = 'kedopenstack'

test.before(t => {
    // this runs once before all tests.
    // do setup here. e.g:
    //  - Create a namespace for your tests (using kubectl or kubernetes node-client)
    //  - Create deployment (using kubectl or kubernetes node-client)
    //  - Setup event source (deploy redis, or configure azure storage, etc)
    //  - etc
    
    // deploy scaler
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, scaledObjectYaml)
    t.is(
        0,
        sh.exec(`kubectl -n ${testNamespace} apply -f ${tmpFile.name}`).code, 'creating scaledObject should work.'
    )
});

test.serial('test 1', t => { });


test.after.always.cb('clean up always after all tests', t => {
    // Clean up after your test here. without `always` this will only run if all tests are successful.
    t.end();
});

const scaledObjectYaml = `apiVersion: v1
kind: Secret
metadata:
  name: openstack-secret
  namespace: default
type: Opaque
data:
  userID: MWYwYzI3ODFiNDExNGQxM2E0NGI4ODk4Zjg1MzQwYmU=
  password: YWRtaW5QYXNz
  projectID: YjE2MWRjNTE4Y2QyNGJkYTg0ZDk0ZDlhMGU3M2ZjODc=
  authURL: aHR0cDovLzEwLjEwMC4yNi4xMDA6NTAwMC92My8=
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-openstack-secret
  namespace: default
spec:
  secretTargetRef:
  - parameter: userID
    name: openstack-secret
    key: userID
  - parameter: password
    name: openstack-secret
    key: password
  - parameter: projectID
    name: openstack-secret
    key: projectID
  - parameter: authURL
    name: openstack-secret
    key: authURL
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: swift-scaledobject
  namespace: default
spec:
  scaleTargetRef:
    name: hello-node
  pollingInterval: 10   # Optional. Default: 30 seconds
  cooldownPeriod: 10    # Optional. Default: 300 seconds
  minReplicaCount: 0
  triggers:
  - type: openstack-swift
    metadata:
      swiftURL: http://10.100.26.100:8080/v1/AUTH_b161dc518cd24bda84d94d9a0e73fc87
      containerName: my-container # Required: Name of OpenStack Swift container
      objectCount: "1" # Optional. Amount of objects to scale out on. Default: 5 objects
    authenticationRef:
        name: keda-trigger-auth-openstack-secret`