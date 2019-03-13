load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.18.0/rules_go-0.18.0.tar.gz"],
    sha256 = "301c8b39b0808c49f98895faa6aa8c92cbd605ab5ad4b6a3a652da33a1a2ba2e",
)
http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

go_repository(
    name = "com_github_gebn_go_stamp",
    tag = "1.0.0",
    importpath = "github.com/gebn/go-stamp",
)
go_repository(
    name = "com_github_fsnotify_fsnotify",
    tag = "v1.4.7",
    importpath = "github.com/fsnotify/fsnotify",
)
go_repository(
    name = "com_github_aws_aws_sdk_go",
    tag = "v1.16.26",
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
