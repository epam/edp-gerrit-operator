mkdir -p /tmp/all-projects
cp -r /var/gerrit/review_site/git/All-Projects.git /tmp/all-projects/.git
cd /tmp/all-projects
git config core.bare false
git config --global user.email "admin@example.com"
git config --global user.name "Admin"
origin=\$(git remote -v | grep /var/gerrit/review_site/git/All-Projects.git | awk 'NR==1{print \$2}')
if [[ \"\$origin\" != \"/var/gerrit/review_site/git/All-Projects.git\" ]]; then
  git remote add origin /var/gerrit/review_site/git/All-Projects.git
fi
git fetch -q origin refs/meta/config:refs/remotes/origin/meta/config
git checkout meta/config
printf \"\$1\" > project.config
printf \"global:Change-Owner\\tChange Owner\\n\" >> groups
printf \"\$2\\tContinuous Integration Tools\\n\" >> groups
printf \"\$3\\tProject Bootstrappers\\n\" >> groups
printf \"\$4\\tDevelopers\\n\" >> groups
printf \"\$5\\tReadOnly\\n\" >> groups

cat << EOF > "webhooks.config"
[remote "changemerged"]
  url = http://el-gerrit-listener:8080
  event = change-merged

[remote "patchsetcreated"]
  url = http://el-gerrit-listener:8080
  event = patchset-created
EOF

git add .
git commit -a -m \"Uploaded EDP Gerrit config\"
git push origin HEAD:refs/meta/config
git config -f /var/gerrit/review_site/etc/gerrit.config auth.trustedOpenID ^.*\$
