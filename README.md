# Go [dep](https://github.com/golang/dep) rules for [Bazel](https://bazel.build/)

See [go-dep-bazel-vscode-example](https://github.com/scele/go-dep-bazel-vscode-example) for an example project.

## Workspace rules

Generate `Gopkg.lock` file with `dep init`, and add following rules to your `WORKSPACE` file:

Refer to

```bzl
http_archive(
    name = "com_github_scele_rules_go_dep",
    urls = ["https://github.com/scele/rules_go_dep/archive/70fa72816cae64f67634740bc6c0233f39b5d8c6.tar.gz"],
    strip_prefix = "rules_go_dep-70fa72816cae64f67634740bc6c0233f39b5d8c6",
    sha256 = "20eeac91a621af97a39e9e30848727d6c14c7d68fcf3fcc139e98e3c363b4661",
)

load("@com_github_scele_rules_go_dep//dep:dep.bzl", "dep_import")

dep_import(
    name = "godeps",
    gopkg_lock = "//:Gopkg.lock",
)
load("@godeps//:Gopkg.bzl", "go_deps")
go_deps()
```

This will load all go dependencies expressed in `Gopkg.lock` into your workspace.
