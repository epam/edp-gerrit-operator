# Replicate Gerrit to GitLab

In order to replicate Gerrit to GitLab, perform the following steps:

1. Create ConfigMap for SSH config. In OpenShift, navigate to Resources → ConfigMap → Gerrit → Add item → type config into the Key field:
    ```bash
    Host git.epam.com
        User git
        Port 22
        IdentityFile /var/gerrit/review_site/etc/replication_rsa_key
        StrictHostKeyChecking no
        UserKnownHostsFile /dev/null
    ```
2. Mount the ConfigMap. Open Applications → Deployments → Gerrit → Actions → Edit YAML:

    ```yaml
           volumeMounts:
             - mountPath: /var/gerrit/.ssh
               name: ssh-config

    ---
        volumes:
            - name: gerrit-data
            persistentVolumeClaim:
                claimName: gerrit-data
            - configMap:
                defaultMode: 420
                items:
                - key: config
                    path: config
                name: gerrit
            name: ssh-config
    ```
3. Create an RSA key. Go to Applications → Pods → Gerrit → Terminal and enter the corresponding code.

   3.1 If the Gerrit version is lower than 2.16:
    ```yaml
    ssh-keygen -t rsa -b 4096 -f /var/gerrit/review_site/etc/replication_rsa_key
    ```
   3.2 If the Gerrit version is 2.16 and higher:
    ```yaml
    ssh-keygen -t rsa -b 4096 -m PEM -f /var/gerrit/review_site/etc/replication_rsa_key
    ```
4. Add the .pub key. Open the Firefox browser → New incognito window → GitLab → connect as autouser → add the created SSH key (it is indicated in the third point).
5. Create projects in GitLab with the same as in Gerrit names, and add an autouser to the projects.
6. Turn off the protected mode in the master branch in GitLab.
7. Customize the replication config. Find an empty file **vi /var/gerrit/review_site/etc/replication.config** and add the following by navigating to Applications → Pods → gerrit → Terminal:

    ```bash
    [gerrit]
    defaultForceUpdate = true

    [remote "project-name"]
    url = git@git.epam.com:main-project/project-name.git
    fetch = +refs/*:refs/*
    push = +refs/heads/*:refs/heads/*
    projects = project-name
    replicatePermissions = false

    [remote "project-name"]
    url = git@git.epam.com:main-project/project-name.git
    fetch = +refs/*:refs/*
    push = +refs/heads/*:refs/heads/*
    projects = project-name
    replicatePermissions = false

    [remote "project-name"]
    url = git@git.epam.com:main-project/project-name.git
    fetch = +refs/*:refs/*
    push = +refs/heads/*:refs/heads/*
    projects = project-name
    replicatePermissions = false
    ```

8. Go to Jenkins pods (Applications → Pods → Jenkins → Terminal), and find a port in the Gerrit service:

```bash
ssh -p 30777 gerrit gerrit plugin reload replication
# wait for 30 sec
ssh -p 30777 gerrit replication start --all --wait
```

>_**NOTE**: If you encounter an error, please check the /var/gerrit/review_site/logs/replication.log file._