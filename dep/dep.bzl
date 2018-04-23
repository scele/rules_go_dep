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
    ctx.file("BUILD.bazel", """
package(default_visibility = ["//visibility:public"])

sh_binary(
    name = "update",
    srcs = ["update.sh"],
)
""")

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

    # TODO(lpeltonen): Is there a better way to get path to the WORKSPACE root?
    result = ctx.execute([ctx.which("cat"), ctx.path("../../DO_NOT_BUILD_HERE")])
    if result.return_code:
        fail("Could not figure out workspace root: %s (%s)" % (result.stdout, result.stderr))
    workspace_root_path = result.stdout

    ctx.template(
        "update.sh",
        Label("//dep:update.sh.tpl"),
        substitutions = {
            "%{dep2bazel}": str(ctx.path("bin/dep2bazel")),
            "%{build_file_generation}": ctx.attr.build_file_generation,
            "%{build_file_proto_mode}": ctx.attr.build_file_proto_mode,
            "%{go_prefix}": ctx.attr.prefix,
            "%{workspace_root_path}": workspace_root_path,
            "%{gopkg_lock}": str(ctx.path(ctx.attr.gopkg_lock)),
            "%{gopkg_bzl}": "" if ctx.attr.gopkg_bzl == None else str(ctx.path(ctx.attr.gopkg_bzl)),
            "%{mirrors}": " ".join(["-mirror=%s=%s" % (k, v) for k, v in ctx.attr.mirrors.items()]),
        },
        executable=True,
    )

    cmd = [
        ctx.path("bin/dep2bazel"),
        "-build-file-generation",
        ctx.attr.build_file_generation,
        "-build-file-proto-mode",
        ctx.attr.build_file_proto_mode,
        "-gopath",
        ctx.path("."),
        "-bazel-output-base",
        ctx.path("../.."),
        "-go-prefix",
        ctx.attr.prefix,
        "-source-directory",
        workspace_root_path,
    ]
    cmd += ["-mirror=%s=%s" % (k, v) for k, v in ctx.attr.mirrors.items()]
    if ctx.attr.gopkg_bzl == None:
        cmd += ["-o", "Gopkg.bzl"]
    else:
        ctx.symlink(ctx.path(ctx.attr.gopkg_bzl), "Gopkg.bzl")

    cmd += [ctx.path(ctx.attr.gopkg_lock)]

    result = ctx.execute(cmd, quiet=False)
    if result.return_code:
        fail("dep_import failed: %s (%s)" % (result.stdout, result.stderr))

    ctx.execute(["rm", workspace_root_path + "/bazel-gopath"])
    ctx.execute(["ln", "-s", ctx.path("."), workspace_root_path + "/bazel-gopath"])
    ctx.symlink(workspace_root_path + "/bazel-genfiles", "genfiles/src/" + ctx.attr.prefix)


dep_import = repository_rule(
    attrs = {
        "gopkg_lock": attr.label(
            allow_files = True,
            mandatory = True,
            single_file = True,
        ),
        "gopkg_bzl": attr.label(
            allow_files = True,
            single_file = True,
        ),
        "prefix": attr.string(mandatory = True),
        "build_file_generation": attr.string(default = "on"),
        "build_file_proto_mode": attr.string(default = "disable"),
        "mirrors": attr.string_dict(),
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
