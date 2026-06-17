package main

import (
	"github.com/marcbran/jsonnet-plugin-kubernetes/kubernetes"
)

func main() {
	kubernetes.Plugin().Serve()
}
