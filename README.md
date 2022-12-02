# xcontroller
A sample Kubernetes Controller using protobuf to define CRD. 

This repository is base on the concept of [kubernetes/sample-controller](https://github.com/kubernetes/sample-controller) but
more details. Additionally, this repository also use protobuf to define CRD model.

Basically, `go-client` using `json` to marshal and unmarshal CRD after retrieve them from API, 
and you are unable to unmarshal a proto message by `encoding/json`. To do that, you need a marshal/unmarshal redirect 
to `jsonpb`.

```go
package v1alpha1

import (
    bytes "bytes"
    jsonpb "github.com/golang/protobuf/jsonpb"
)

type Bar struct { } // Your CRD

func (this *Bar) MarshalJSON() ([]byte, error) {
	str, err := BarMarshaler.MarshalToString(this)
	return []byte(str), err
}

func (this *Bar) UnmarshalJSON(b []byte) error {
	return BarUnmarshaler.Unmarshal(bytes.NewReader(b), this)
}

var (
	BarMarshaler   = &jsonpb.Marshaler{}
	BarUnmarshaler = &jsonpb.Unmarshaler{AllowUnknownFields: true}
)
```

The second thing you need is the deepcopy.

```go
package v1alpha1

import (
	proto "google.golang.org/protobuf/proto"
)

func (in *Bar) DeepCopyInto(out *Bar) {
	p := proto.Clone(in).(*Bar)
	*out = *p
}

func (in *Bar) DeepCopy() *Bar {
	if in == nil {
		return nil
	}
	out := new(Bar)
	in.DeepCopyInto(out)
	return out
}

func (in *Bar) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}
```

Fortunately, `istio/tools` support us several tools to automatically generate these codes, this will save a lot of times
for us when we're working with Kubernetes CRDs. Big thanks for `istio` contributors.

## Usage

### To generate code from protobuf
Make sure that you already install `buf`.

```console
# install istio tools
$ go install istio.io/tools/cmd/protoc-gen-golang-deepcopy

$ go install istio.io/tools/cmd/protoc-gen-golang-jsonshim

# lint check before generate code
$ buf lint --path api

$ buf generate --path api
```

### To update codegen
```
$ export CODEGEN_PKG=~/go/pkg/mod/k8s.io/code-generator@v0.21.0
$ ./hack/update-codegen.sh 
```

### To start `xcontroller`
```console
$ xcontroller --help
Usage:
  xcontroller [flags]

Flags:
  -h, --help                help for xcontroller
      --kubeconfig string   Full path to kubernetes client configuration, i.e. ~/.kube/config
      --log-level string    Log level (default "info")
      --workers int         Number of workers (default 10)
```

## References
* https://github.com/kubernetes/sample-controller
* https://github.com/istio/tools
