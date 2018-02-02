"""Import go dep dependencies into Bazel."""

def executable_extension(ctx):
    extension = ""
    if ctx.os.name.startswith('windows'):
        extension = ".exe"
    return extension

def env_execute(ctx, arguments, environment = {}, **kwargs):
    """env_executes a command in a repository context. It prepends "env -i"
    to "arguments" before calling "ctx.execute".

    Variables that aren't explicitly mentioned in "environment"
    are removed from the environment. This should be preferred to "ctx.execute"
    in most situations.
    """
    if ctx.os.name.startswith('windows'):
        return ctx.execute(arguments, environment=environment, **kwargs)
    env_args = ["env", "-i"]
    environment = dict(environment)
    for var in ["TMP", "TMPDIR"]:
        if var in ctx.os.environ and not var in environment:
            environment[var] = ctx.os.environ[var]
    for k, v in environment.items():
        env_args.append("%s=%s" % (k, v))
    arguments = env_args + arguments
    return ctx.execute(arguments, **kwargs)

def _dep_import_impl(ctx):
    ctx.file("BUILD.bazel", """package(default_visibility = ["//visibility:public"])""")

    extension = executable_extension(ctx)
    go_tool = ctx.path(Label("@go_sdk//:bin/go{}".format(extension)))
    dep2bazel_path = ctx.path(ctx.attr._rules_go_dep).dirname

    # Build something that looks like a normal GOPATH so go install will work
    ctx.symlink(dep2bazel_path, "src/github.com/scele/rules_go_dep")
    env = {
        'GOROOT': str(go_tool.dirname.dirname),
        'GOPATH': str(ctx.path('')),
    }
    result = env_execute(ctx, [go_tool, "install", "github.com/scele/rules_go_dep/dep2bazel"], environment = env)
    if result.return_code:
        fail("failed to build dep2bazel: {}".format(result.stderr))

    result = ctx.execute([
        ctx.path("bin/dep2bazel"),
        "-build-file-generation",
        ctx.attr.build_file_generation,
        "-build-file-proto-mode",
        ctx.attr.build_file_proto_mode,
        "-o",
        "Gopkg.bzl",
        "-gopath",
        ctx.path("."),
        "-bazel-output-base",
        ctx.path("../.."),
        ctx.path(ctx.attr.gopkg_lock)
    ])

    if result.return_code:
        fail("dep_import failed: %s (%s)" % (result.stdout, result.stderr))

dep_import = repository_rule(
    attrs = {
        "gopkg_lock": attr.label(
            allow_files = True,
            mandatory = True,
            single_file = True,
        ),
        "build_file_generation": attr.string(default = "on"),
        "build_file_proto_mode": attr.string(default = "disable"),
        "_rules_go_dep": attr.label(
            default = Label("//:WORKSPACE"),
            allow_files = True,
            single_file = True,
            executable = True,
            cfg = "host",
        ),
    },
    implementation = _dep_import_impl,
)
