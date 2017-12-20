load("@io_bazel_rules_go//go:def.bzl", "go_prefix", "gazelle")

go_prefix("github.com/scele/rules_go_dep")

gazelle(
    name = "gazelle",
    args = [
        "-build_file_name",
        "BUILD,BUILD.bazel",
        "-proto",
        "disable",
    ],
    prefix = "github.com/scele/rules_go_dep",
)
