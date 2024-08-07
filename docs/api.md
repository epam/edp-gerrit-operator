# API Reference

Packages:

- [v2.edp.epam.com/v1](#v2edpepamcomv1)

# v2.edp.epam.com/v1

Resource Types:

- [GerritGroupMember](#gerritgroupmember)

- [GerritGroup](#gerritgroup)

- [GerritMergeRequest](#gerritmergerequest)

- [GerritProjectAccess](#gerritprojectaccess)

- [GerritProject](#gerritproject)

- [GerritReplicationConfig](#gerritreplicationconfig)

- [Gerrit](#gerrit)




## GerritGroupMember
<sup><sup>[↩ Parent](#v2edpepamcomv1 )</sup></sup>






GerritGroupMember is the Schema for the gerrit group member API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v2.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>GerritGroupMember</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#gerritgroupmemberspec">spec</a></b></td>
        <td>object</td>
        <td>
          GerritGroupMemberSpec defines the desired state of GerritGroupMember.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritgroupmemberstatus">status</a></b></td>
        <td>object</td>
        <td>
          GerritGroupMemberStatus defines the observed state of GerritGroupMember.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritGroupMember.spec
<sup><sup>[↩ Parent](#gerritgroupmember)</sup></sup>



GerritGroupMemberSpec defines the desired state of GerritGroupMember.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accountId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ownerName</b></td>
        <td>string</td>
        <td>
          OwnerName indicates which gerrit CR should be taken to initialize correct client.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritGroupMember.status
<sup><sup>[↩ Parent](#gerritgroupmember)</sup></sup>



GerritGroupMemberStatus defines the observed state of GerritGroupMember.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>failureCount</b></td>
        <td>integer</td>
        <td>
          Preserves Number of Failures during reconciliation phase. Used for exponential back-off calculation<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GerritGroup
<sup><sup>[↩ Parent](#v2edpepamcomv1 )</sup></sup>






GerritGroup is the Schema for the gerrit group API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v2.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>GerritGroup</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#gerritgroupspec">spec</a></b></td>
        <td>object</td>
        <td>
          GerritGroupSpec defines the desired state of GerritGroup.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritgroupstatus">status</a></b></td>
        <td>object</td>
        <td>
          GerritGroupStatus defines the observed state of GerritGroup.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritGroup.spec
<sup><sup>[↩ Parent](#gerritgroup)</sup></sup>



GerritGroupSpec defines the desired state of GerritGroup.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gerritOwner</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>visibleToAll</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritGroup.status
<sup><sup>[↩ Parent](#gerritgroup)</sup></sup>



GerritGroupStatus defines the observed state of GerritGroup.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>groupId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GerritMergeRequest
<sup><sup>[↩ Parent](#v2edpepamcomv1 )</sup></sup>






GerritMergeRequest is the Schema for the gerrit merge request API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v2.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>GerritMergeRequest</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#gerritmergerequestspec">spec</a></b></td>
        <td>object</td>
        <td>
          GerritMergeRequestSpec defines the desired state of GerritMergeRequest.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritmergerequeststatus">status</a></b></td>
        <td>object</td>
        <td>
          GerritMergeRequestStatus defines the observed state of GerritMergeRequest.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritMergeRequest.spec
<sup><sup>[↩ Parent](#gerritmergerequest)</sup></sup>



GerritMergeRequestSpec defines the desired state of GerritMergeRequest.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>authorEmail</b></td>
        <td>string</td>
        <td>
          AuthorEmail is the email of the user who creates the merge request.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>authorName</b></td>
        <td>string</td>
        <td>
          AuthorName is the name of the user who creates the merge request.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>projectName</b></td>
        <td>string</td>
        <td>
          ProjectName is gerrit project name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>additionalArguments</b></td>
        <td>[]string</td>
        <td>
          AdditionalArguments contains merge command additional command line arguments.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>changesConfigMap</b></td>
        <td>string</td>
        <td>
          ChangesConfigMap is the name of the ConfigMap, which contains files contents that should be merged.
ConfigMap should contain eny data keys with content in the json
format: {"path": "/controllers/user.go", "contents": "some code here"} - to add file
or format: {"path": "/controllers/user.go"} - to remove file.
If files already exist in the project, they will be overwritten.
If empty, sourceBranch should be set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>commitMessage</b></td>
        <td>string</td>
        <td>
          CommitMessage is the commit message for the merge request.
If empty, the operator will generate the commit message.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ownerName</b></td>
        <td>string</td>
        <td>
          OwnerName is the name of Gerrit CR, which should be used to initialize the client.
If empty, the operator will get first Gerrit CR from the namespace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sourceBranch</b></td>
        <td>string</td>
        <td>
          SourceBranch is the name of the branch from which the changes should be merged.
If empty, changesConfigMap should be set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetBranch</b></td>
        <td>string</td>
        <td>
          TargetBranch is the name of the branch to which the changes should be merged.
If changesConfigMap is set, the targetBranch can be only the origin HEAD branch.<br/>
          <br/>
            <i>Default</i>: master<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritMergeRequest.status
<sup><sup>[↩ Parent](#gerritmergerequest)</sup></sup>



GerritMergeRequestStatus defines the observed state of GerritMergeRequest.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>changeId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>changeUrl</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GerritProjectAccess
<sup><sup>[↩ Parent](#v2edpepamcomv1 )</sup></sup>






GerritProjectAccess is the Schema for the gerrit project access API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v2.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>GerritProjectAccess</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#gerritprojectaccessspec">spec</a></b></td>
        <td>object</td>
        <td>
          GerritProjectAccessSpec defines the desired state of GerritProjectAccess.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritprojectaccessstatus">status</a></b></td>
        <td>object</td>
        <td>
          GerritProjectAccessStatus defines the observed state of GerritProjectAccess.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritProjectAccess.spec
<sup><sup>[↩ Parent](#gerritprojectaccess)</sup></sup>



GerritProjectAccessSpec defines the desired state of GerritProjectAccess.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>projectName</b></td>
        <td>string</td>
        <td>
          ProjectName is gerrit project name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ownerName</b></td>
        <td>string</td>
        <td>
          OwnerName indicates which gerrit CR should be taken to initialize correct client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>parent</b></td>
        <td>string</td>
        <td>
          Parent is parent project.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritprojectaccessspecreferencesindex">references</a></b></td>
        <td>[]object</td>
        <td>
          References contains gerrit references.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritProjectAccess.spec.references[index]
<sup><sup>[↩ Parent](#gerritprojectaccessspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>action</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>force</b></td>
        <td>boolean</td>
        <td>
          Force indicates whether the force flag is set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>max</b></td>
        <td>integer</td>
        <td>
          Max is the max value of the permission range.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>min</b></td>
        <td>integer</td>
        <td>
          Min is the min value of the permission range.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>permissionLabel</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>permissionName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>refPattern</b></td>
        <td>string</td>
        <td>
          Patter is reference pattern, example: refs/heads/*.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritProjectAccess.status
<sup><sup>[↩ Parent](#gerritprojectaccess)</sup></sup>



GerritProjectAccessStatus defines the observed state of GerritProjectAccess.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>created</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GerritProject
<sup><sup>[↩ Parent](#v2edpepamcomv1 )</sup></sup>






GerritProject is the Schema for the gerrit project API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v2.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>GerritProject</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#gerritprojectspec">spec</a></b></td>
        <td>object</td>
        <td>
          GerritProjectSpec defines the desired state of GerritProject.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritprojectstatus">status</a></b></td>
        <td>object</td>
        <td>
          GerritProjectStatus defines the observed state of GerritProject.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritProject.spec
<sup><sup>[↩ Parent](#gerritproject)</sup></sup>



GerritProjectSpec defines the desired state of GerritProject.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>branches</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>createEmptyCommit</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ownerName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>owners</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>parent</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>permissionsOnly</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>rejectEmptyCommit</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>submitType</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritProject.status
<sup><sup>[↩ Parent](#gerritproject)</sup></sup>



GerritProjectStatus defines the observed state of GerritProject.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>branches</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GerritReplicationConfig
<sup><sup>[↩ Parent](#v2edpepamcomv1 )</sup></sup>






GerritReplicationConfig is the Schema for the gerrit replication config API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v2.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>GerritReplicationConfig</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#gerritreplicationconfigspec">spec</a></b></td>
        <td>object</td>
        <td>
          GerritReplicationConfigSpec defines the desired state of GerritReplicationConfig.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritreplicationconfigstatus">status</a></b></td>
        <td>object</td>
        <td>
          GerritReplicationConfigStatus defines the observed state of GerritReplicationConfig.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritReplicationConfig.spec
<sup><sup>[↩ Parent](#gerritreplicationconfig)</sup></sup>



GerritReplicationConfigSpec defines the desired state of GerritReplicationConfig.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ssh_url</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>owner_name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritReplicationConfig.status
<sup><sup>[↩ Parent](#gerritreplicationconfig)</sup></sup>



GerritReplicationConfigStatus defines the observed state of GerritReplicationConfig.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>available</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lastTimeUpdated</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Gerrit
<sup><sup>[↩ Parent](#v2edpepamcomv1 )</sup></sup>






Gerrit is the Schema for the gerrits API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>v2.edp.epam.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Gerrit</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#gerritspec">spec</a></b></td>
        <td>object</td>
        <td>
          GerritSpec defines the desired state of Gerrit.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritstatus">status</a></b></td>
        <td>object</td>
        <td>
          GerritStatus defines the observed state of Gerrit.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Gerrit.spec
<sup><sup>[↩ Parent](#gerrit)</sup></sup>



GerritSpec defines the desired state of Gerrit.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#gerritspeckeycloakspec">keycloakSpec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>basePath</b></td>
        <td>string</td>
        <td>
          BasePath gerrit http route base path.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalURL</b></td>
        <td>string</td>
        <td>
          ExternalURL gerrit full external url for keycloak or other integrations<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>restAPIUrl</b></td>
        <td>string</td>
        <td>
          RestAPIUrl gerrit http full api url.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sshPort</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sshUrl</b></td>
        <td>string</td>
        <td>
          SSHUrl gerrit ssh url.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Gerrit.spec.keycloakSpec
<sup><sup>[↩ Parent](#gerritspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>realm</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Gerrit.status
<sup><sup>[↩ Parent](#gerrit)</sup></sup>



GerritStatus defines the observed state of Gerrit.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>externalUrl</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>available</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lastTimeUpdated</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>