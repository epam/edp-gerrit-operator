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
echo \$1 > project.config
echo \"global:Change-Owner\tChange Owner\" > groups
echo \$2 > groups
echo \$3 > groups
git add .
git commit -a -m \"Uploaded EDP Gerrit config\"
git push origin HEAD:refs/meta/config
rm -rf groups
git rm groups
git config -f /var/gerrit/review_site/etc/gerrit.config auth.trustedOpenID ^.*\$