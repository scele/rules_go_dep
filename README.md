# Go [dep](https://github.com/golang/dep) rules for [Bazel](https://bazel.build/)

See [go-dep-bazel-vscode-example](https://github.com/scele/go-dep-bazel-vscode-example) for an example project.

## Workspace rules

Generate `Gopkg.lock` file with `dep init`, and add following rules to your `WORKSPACE` file:

```bzl
http_archive(
    name = "com_github_scele_rules_go_dep",
    urls = ["https://github.com/scele/rules_go_dep/archive/4aa1bd3550191b39abded31bcf06d233b67fa8bb.tar.gz"],
    strip_prefix = "rules_go_dep-4aa1bd3550191b39abded31bcf06d233b67fa8bb",
    sha256 = "068d102168fdef7bb9da4f7c699df6b1b1ff25230f6a45e3b8da5e8ab15c6c36",
)

load("@com_github_scele_rules_go_dep//dep:dep.bzl", "dep_import")

dep_import(
    name = "godeps",
    gopkg_lock = "//:Gopkg.lock",
    prefix = "github.com/my/project",
    # Optional: if you want to use checked-in Gopkg.bzl.
    gopkg_bzl = "//:Gopkg.bzl",
)
load("@godeps//:Gopkg.bzl", "go_deps")
go_deps()
```

This will load all go dependencies expressed in `Gopkg.lock` into your workspace.

Using checked-in `Gopkg.bzl` can result in faster builds, since `Gopkg.bzl` does not need to be generated
on the fly.  If `gopkg_bzl` is used, then the checked-in `Gopkg.bzl` can be updated with:

```sh
bazel run @godeps//:update
```
