module github.com/epmd-edp/gerrit-operator/v2

go 1.12

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0

require (
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/epmd-edp/jenkins-operator/v2 v2.1.0-43
	github.com/epmd-edp/keycloak-operator v1.0.30-alpha-55
	github.com/go-openapi/spec v0.19.3
	github.com/google/uuid v1.1.1
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/operator-framework/operator-sdk v0.0.0-20190530173525-d6f9cdf2f52e
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3
	golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	gopkg.in/resty.v1 v1.12.0
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/kube-openapi v0.0.0-20181109181836-c59034cc13d5
	sigs.k8s.io/controller-runtime v0.1.12
)
