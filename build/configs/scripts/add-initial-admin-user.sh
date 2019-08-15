mkdir -p /tmp/git
cp -r /var/gerrit/review_site/git/All-Users.git /tmp/git/.git
cd /tmp/git
git config core.bare false
git config --global user.email "admin@example.com"
git config --global user.name "Admin"
git checkout -q refs/meta/group-names
ADMIN_GROUP=\$(grep uuid \$(grep Administrator * | awk -F : '{print \$1}') | awk '{print \$3}')
ADMIN_GROUP_REF=\$(git show-ref | grep \$ADMIN_GROUP | awk '{print \$2}')
git checkout -q \$ADMIN_GROUP_REF
echo 1000000 > members
git add members
git commit -m \"Add Admin user to Administrators group\"
origin=\$(git remote -v | grep /var/gerrit/review_site/git/All-Users.git | awk 'NR==1{print \$2}')
if [[ \"\$origin\" != \"/var/gerrit/review_site/git/All-Users.git\" ]]; then
  git remote add origin /var/gerrit/review_site/git/All-Users.git
fi
git push origin HEAD:\$ADMIN_GROUP_REF
rm -rf members
git rm members
echo \$1 > authorized_keys
git add authorized_keys
git commit -m \"Add Admin user ssh key\"
git push origin HEAD:refs/users/00/1000000
rm -rf authorized_keys
git rm authorized_keys
cat << EOF >> b54915000d281bb92f990131b8356c67fa065353
[externalId \"username:admin\"]
        accountId = 1000000
        password = bcrypt:4:Tx3ksWeawlYm0uIh/HXw6w==:FWA2CWWI92yKHXKLMCy91Nfvk9leasFq
        email = admin@example.com
EOF
git add b54915000d281bb92f990131b8356c67fa065353
git commit -m \"Add Admin external user\"
git push origin HEAD:refs/meta/external-ids
BLOB_ID=\$(echo '1000001' | git hash-object -w --stdin)
BLOB_H=\$(echo \$BLOB_ID | head -c2)
BLOB_L=\$(echo \$BLOB_ID | tail -c+3)
mkdir -p /var/gerrit/review_site/git/All-Users.git/objects/\$BLOB_H
cp .git/objects/\$BLOB_H/\$BLOB_L /var/gerrit/review_site/git/All-Users.git/objects/\$BLOB_H/\$BLOB_L
echo \$BLOB_ID > /var/gerrit/review_site/git/All-Users.git/refs/sequences/accounts