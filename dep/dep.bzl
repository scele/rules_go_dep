"""Import go dep dependencies into Bazel."""

def _dep_import_impl(repository_ctx):
  """Core implementation of dep_import."""

  # Add an empty top-level BUILD file.
  # This is because Bazel requires BUILD files along all paths accessed
  # via //this/sort/of:path and we wouldn't be able to load our generated
  # Gopkg.bzl without it.
  repository_ctx.file("BUILD", "")

  # To see the output, pass: quiet=False
  result = repository_ctx.execute([
    repository_ctx.path(repository_ctx.attr._script),
    repository_ctx.path(repository_ctx.attr.deps),
    "--output", repository_ctx.path("Gopkg.bzl"),
  ])

  if result.return_code:
    fail("dep_import failed: %s (%s)" % (result.stdout, result.stderr))

dep_import = repository_rule(
    attrs = {
        "deps": attr.label(
            allow_files = True,
            mandatory = True,
            single_file = True,
        ),
        "_script": attr.label(
            executable = True,
            default = Label("@com_github_scele_rules_go_dep//dep2bazel:dep2bazel"),
            cfg = "host",
        ),
    },
    implementation = _dep_import_impl,
)
