go get github.com/mitchellh/gox
gox \
  -output "bin/{{.Dir}}_{{.OS}}_{{.Arch}}" \
  -osarch="linux/amd64" \
  -osarch="linux/arm" \
  -osarch="windows/amd64" \
  -osarch="darwin/amd64" \
  -osarch="freebsd/amd64"
