load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "io_bazel_rules_go",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/v0.20.3/rules_go-v0.20.3.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.20.3/rules_go-v0.20.3.tar.gz",
    ],
    sha256 = "e88471aea3a3a4f19ec1310a55ba94772d087e9ce46e41ae38ecebe17935de7b",
)
http_archive(
    name = "bazel_gazelle",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.1/bazel-gazelle-v0.19.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.1/bazel-gazelle-v0.19.1.tar.gz",
    ],
    sha256 = "86c6d481b3f7aedc1d60c1c211c6f76da282ae197c3b3160f54bd3a8f847896f",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

go_repository(
    name = "com_github_gebn_go_stamp",
    tag = "v2.0.0",
    importpath = "github.com/gebn/go-stamp",
)
go_repository(
    name = "com_github_fsnotify_fsnotify",
    tag = "v1.4.7",
    importpath = "github.com/fsnotify/fsnotify",
)
go_repository(
    name = "com_github_aws_aws_sdk_go",
    tag = "v1.18.3",
    importpath = "github.com/aws/aws-sdk-go",
)
go_repository(
    name = "com_github_alecthomas_kingpin",
    tag = "v2.2.6",
    importpath = "gopkg.in/alecthomas/kingpin.v2",
)
go_repository(
    name = "com_github_alecthomas_units",
    commit = "2efee857e7cfd4f3d0138cc3cbb1b4966962b93a",  # master as of 2015-10-22
    importpath = "github.com/alecthomas/units",
)
go_repository(
    name = "com_github_alecthomas_template",
    commit = "a0175ee3bccc567396460bf5acd36800cb10c49c",  # master as of 2016-04-05
    importpath = "github.com/alecthomas/template",
)
