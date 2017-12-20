workspace(name="com_github_scele_rules_go_dep")

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.8.1/rules_go-0.8.1.tar.gz",
    sha256 = "90bb270d0a92ed5c83558b2797346917c46547f6f7103e648941ecdb6b9d0e72",
)

#http_archive(
#    name = "com_github_scele_rules_go_dep",
#    urls = ["https://github.com/scele/rules_go_dep/archive/33da00fdf845d0b1ebddb9d698ae244097317ecf.tar.gz"],
#    sha256 = "c6af60f60a7dd90bf9eca462273901839f84300fcfb7f41718a063fe1131897b",
#    strip_prefix = "rules-go-dep-33da00fdf845d0b1ebddb9d698ae244097317ecf",
#)

local_repository(
    name = "com_github_scele_rules_go_dep",
    path = "/Users/lauri/go/src/github.com/scele/rules_go_dep",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains(go_version="1.9")

load("@com_github_scele_rules_go_dep//dep:dep.bzl", "dep_import", "repositories")
repositories()
dep_import(
    name = "deps",
    deps = ":Gopkg.lock",
)

load("@deps//:Gopkg.bzl", "godeps")
godeps()
