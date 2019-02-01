load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.17.0/rules_go-0.17.0.tar.gz"],
    sha256 = "492c3ac68ed9dcf527a07e6a1b2dcbf199c6bf8b35517951467ac32e421c06c1",
)
http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
    sha256 = "7949fc6cc17b5b191103e97481cf8889217263acf52e00b560683413af204fcb",
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
