# Go [dep](https://github.com/golang/dep) rules for [Bazel](https://bazel.build/)

## dep2bazel

`dep2bazel` is a utility that onverts `Gopkg.lock` to bazel `go_repository` workspace rules.

After modifying Gopkg.lock with `dep ensure`, do:

```sh
go get -u github.com/scele/rules_go_dep/dep2bazel
dep2bazel ./Gopkg.lock > Gopkg.bzl
```

Refer to the generated Gopkg.bzl from WORKSPACE file like this:

```bzl
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains(go_version="1.9")

load("//:Gopkg.bzl", "go_deps")
go_deps()
```

The tool attempts use [http_archive](https://docs.bazel.build/versions/master/be/workspace.html#http_archive)-based
dependencies with sha256 checksums.  If that fails, it will fall back to
[git_repository](https://docs.bazel.build/versions/master/be/workspace.html#git_repository)-based dependency.
