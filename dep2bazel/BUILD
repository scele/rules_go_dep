load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
    ],
    importpath = "github.com/scele/rules_go_dep/dep2bazel",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "dep2bazel",
    embed = [":go_default_library"],
    importpath = "github.com/scele/rules_go_dep/dep2bazel",
    visibility = ["//visibility:public"],
)
