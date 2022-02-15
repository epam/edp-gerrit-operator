# API Reference

Packages:

- [v2.edp.epam.com/v1alpha1](#v2edpepamcomv1alpha1)

# v2.edp.epam.com/v1alpha1

Resource Types:

- [Gerrit](#gerrit)

- [GerritGroup](#gerritgroup)

- [GerritGroupMember](#gerritgroupmember)

- [GerritProject](#gerritproject)

- [GerritProjectAccess](#gerritprojectaccess)

- [GerritReplicationConfig](#gerritreplicationconfig)

- [GerritMergeRequest](#gerritmergerequest)




## Gerrit
<sup><sup>[↩ Parent](#v2edpepamcomv1alpha1 )</sup></sup>



## GerritGroup
<sup><sup>[↩ Parent](#v2edpepamcomv1alpha1 )</sup></sup>



## GerritGroupMember
<sup><sup>[↩ Parent](#v2edpepamcomv1alpha1 )</sup></sup>








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
      <td>v2.edp.epam.com/v1alpha1</td>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritGroupMember.spec
<sup><sup>[↩ Parent](#gerritgroupmember)</sup></sup>





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
          Property indicates which gerrit car should be taken to initialize correct client.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GerritProject
<sup><sup>[↩ Parent](#v2edpepamcomv1alpha1 )</sup></sup>



## GerritProjectAccess
<sup><sup>[↩ Parent](#v2edpepamcomv1alpha1 )</sup></sup>








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
      <td>v2.edp.epam.com/v1alpha1</td>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritProjectAccess.spec
<sup><sup>[↩ Parent](#gerritprojectaccess)</sup></sup>





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
          gerrit project name<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ownerName</b></td>
        <td>string</td>
        <td>
          Property indicates which gerrit car should be taken to initialize correct client.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>parent</b></td>
        <td>string</td>
        <td>
          parent project<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#gerritprojectaccessspecreferencesindex">references</a></b></td>
        <td>[]object</td>
        <td>
          gerrit references list<br/>
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
          whether the force flag is set<br/>
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
          the max value of the permission range<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>min</b></td>
        <td>integer</td>
        <td>
          the min value of the permission range<br/>
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
          reference pattern, example: refs/heads/*<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GerritReplicationConfig
<sup><sup>[↩ Parent](#v2edpepamcomv1alpha1 )</sup></sup>



## GerritMergeRequest
<sup><sup>[↩ Parent](#v2edpepamcomv1alpha1 )</sup></sup>








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
      <td>v2.edp.epam.com/v1alpha1</td>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GerritMergeRequest.spec
<sup><sup>[↩ Parent](#gerritmergerequest)</sup></sup>





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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>authorName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ownerName</b></td>
        <td>string</td>
        <td>
          Property indicates which gerrit car should be taken to initialize correct client.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>projectName</b></td>
        <td>string</td>
        <td>
          gerrit project name<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceBranch</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>additionalArguments</b></td>
        <td>[]string</td>
        <td>
          merge command additional command line arguments<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>commitMessage</b></td>
        <td>string</td>
        <td>
          merge commit message<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>targetBranch</b></td>
        <td>string</td>
        <td>
          optional, default is master<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>