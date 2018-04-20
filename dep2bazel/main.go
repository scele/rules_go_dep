package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"golang.org/x/tools/go/vcs"

	"github.com/BurntSushi/toml"
)

var checksum = flag.Bool("sha256", false, "whether to include tarball checksums")
var buildFileGeneration = flag.String("build-file-generation", "", "the value of build_file_generation attribute")
var buildFileProtoMode = flag.String("build-file-proto-mode", "disable", "the value of build_file_proto_mode attribute")
var outputFilename = flag.String("o", "", "output filename")
var outputGopathRoot = flag.String("gopath", "", "output gopath root")
var bazelOutputRoot = flag.String("bazel-output-base", "", "bazel output base (obtained with \"bazel info output_base\")")
var sourceDirectory = flag.String("source-directory", "", "source directory path")
var goPrefix = flag.String("go-prefix", "", "go prefix (e.g. github.com/scele/rules_go_dep)")

// Lock represents the parsed Gopkg.toml file.
type Lock struct {
	Projects []LockedProject `toml:"projects"`
}

// LockedProject represents one locked project, parsed from the Gopkg.toml file.
type LockedProject struct {
	Name     string   `toml:"name"`
	Branch   string   `toml:"branch,omitempty"`
	Revision string   `toml:"revision"`
	Version  string   `toml:"version,omitempty"`
	Source   string   `toml:"source,omitempty"`
	Packages []string `toml:"packages"`
}

type RemoteTarball struct {
	url         string
	stripPrefix string
	sha256      string
}

type RemoteGitRepo struct {
	revision string
}

type RemoteRepository interface {
	GetRepoString(name string, importPath string) string
}

func downloadFile(f *os.File, url string) (err error) {
	fmt.Fprintf(os.Stderr, "Downloading %v\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// github.com/scele/dep2bazel => com_github_scele_dep2bazel
func bazelName(importpath string) string {
	parts := strings.Split(importpath, "/")
	hostparts := strings.Split(parts[0], ".")
	var slice []string
	for i := len(hostparts) - 1; i >= 0; i-- {
		slice = append(slice, hostparts[i])
	}
	slice = append(slice, parts[1:]...)
	name := strings.Join(slice, "_")
	return strings.ToLower(strings.NewReplacer("-", "_", ".", "_").Replace(name))
}

func githubTarball(url string, revision string) (*RemoteTarball, error) {

	tarball := fmt.Sprintf("%v.tar.gz", revision)
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	filename := f.Name()
	defer os.Remove(filename)

	downloadURL := fmt.Sprintf("%v/archive/%v", url, tarball)
	err = downloadFile(f, downloadURL)
	if err != nil {
		return nil, err
	}
	f.Close()

	// Github tarballs have one top-level directory that we want to strip out.
	// Determine the name of that directory by inspecting the tarball.
	// Usually the directory name is just importname-revision, but we can't assume
	// it since capitalization might differ.
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	gzf, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(gzf)

	// The root directory is the second entry in the tarball.
	head, err := tarReader.Next()
	if err != nil {
		return nil, err
	}
	head, err = tarReader.Next()
	if err != nil {
		return nil, err
	}
	stripPrefix := head.Name

	// Also compute checksum for the downloaded file.
	// NOTE: Github checksums are not stable either, see e.g.
	// https://github.com/bazelbuild/rules_go/issues/820
	// https://github.com/kubernetes/kubernetes/issues/46443
	sha := ""
	if *checksum {
		sha = fmt.Sprintf("%x", sha256.Sum256(b))
	}

	return &RemoteTarball{
		url:         downloadURL,
		stripPrefix: stripPrefix,
		sha256:      sha,
	}, nil
}

func googlesourceTarball(url string, revision string) (*RemoteTarball, error) {
	return &RemoteTarball{
		url:         fmt.Sprintf("%v/+archive/%v.tar.gz", url, revision),
		stripPrefix: "",
		// Astonishingly, archives downloaded from go.googlesource.com produce
		// different checksum for each download...
		sha256: "",
	}, nil
}

var gopkgInPatternOld = regexp.MustCompile(`^/(?:([a-z0-9][-a-z0-9]+)/)?((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(?:-unstable)?)/([a-zA-Z][-a-zA-Z0-9]*)(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)
var gopkgInPatternNew = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(?:-unstable)?)(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

// We map some urls to github urls, since github has good support for downloading
// tarball snapshots.
func remapURL(url string) string {
	if strings.HasPrefix(url, "https://gopkg.in/") {
		// Special handling for gopkg.in which does not support downloading tarballs.
		// Remap gopkg.in => github.com.
		tail := url[len("https://gopkg.in"):]
		m := gopkgInPatternNew.FindStringSubmatch(tail)
		if m == nil {
			m = gopkgInPatternOld.FindStringSubmatch(tail)
			if m == nil {
				return url
			}
			// "/v2/name" <= "/name.v2"
			m[2], m[3] = m[3], m[2]
		}
		repoUser := m[1]
		repoName := m[2]
		if repoUser != "" {
			return "https://github.com/" + repoUser + "/" + repoName
		}
		return "https://github.com/go-" + repoName + "/" + repoName
	} else if strings.HasPrefix(url, "https://go.googlesource.com/") {
		// Try github mirror because go.googlesource.com does not give deterministic
		// checksums for tarball downloads.
		_, repoName := path.Split(url)
		return "https://github.com/golang/" + repoName
	}
	return url
}

func tryTarball(url string, revision string) (*RemoteTarball, error) {
	if strings.HasPrefix(url, "https://github.com/") {
		return githubTarball(url, revision)
	} else if strings.HasPrefix(url, "https://go.googlesource.com/") {
		return googlesourceTarball(url, revision)
	} else {
		return &RemoteTarball{}, fmt.Errorf("Unknown server")
	}
}

// GetRepoString returns the go_repository rule string.
func (t *RemoteTarball) GetRepoString(name string, importPath string) string {
	str := fmt.Sprintf("\n")
	str += fmt.Sprintf("    go_repository(\n")
	str += fmt.Sprintf("        name = \"%v\",\n", name)
	str += fmt.Sprintf("        importpath = \"%v\",\n", importPath)
	str += fmt.Sprintf("        urls = [\"%v\"],\n", t.url)
	str += fmt.Sprintf("        strip_prefix = \"%v\",\n", t.stripPrefix)
	if t.sha256 != "" {
		str += fmt.Sprintf("        sha256 = \"%v\",\n", t.sha256)
	}
	if *buildFileGeneration != "" {
		str += fmt.Sprintf("        build_file_generation = \"%v\",\n", *buildFileGeneration)
	}
	if *buildFileProtoMode != "" {
		str += fmt.Sprintf("        build_file_proto_mode = \"%v\",\n", *buildFileProtoMode)
	}
	str += fmt.Sprintf("    )\n")
	return str
}

// GetRepoString returns the go_repository rule string.
func (t *RemoteGitRepo) GetRepoString(name string, importPath string) string {
	str := fmt.Sprintf("\n")
	str += fmt.Sprintf("    go_repository(\n")
	str += fmt.Sprintf("        name = \"%v\",\n", name)
	str += fmt.Sprintf("        importpath = \"%v\",\n", importPath)
	str += fmt.Sprintf("        commit = \"%v\",\n", t.revision)
	str += fmt.Sprintf("        build_file_proto_mode = \"disable\",\n")
	str += fmt.Sprintf("    )\n")
	return str
}

func remoteRepository(url string, revision string) (RemoteRepository, error) {

	remappedURL := remapURL(url)

	// First, try downloading a tarball using our remapped url.
	tarball, err := tryTarball(remappedURL, revision)
	if err == nil {
		return tarball, nil
	}

	// Then, try downloading a tarball using the original url.
	tarball, err = tryTarball(url, revision)
	if err == nil {
		return tarball, nil
	}

	// If downloading a tarball failed, default to downloading with git.
	return &RemoteGitRepo{revision: revision}, nil
}

const repoTemplateNoChecksum = `
    go_repository(
        name = "%v",
        importpath = "%v",
        urls = ["%v"],
        strip_prefix = "%v",
        build_file_proto_mode = "disable",
    )
`

func usage() {
	fmt.Println("usage: dep2bazel [OPTIONS] <Gopkg.lock>")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		usage()
	}

	var outputFile io.Writer
	if *outputFilename != "" {
		var err error
		outputFile, err = os.Create(*outputFilename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create file %v\n", *outputFilename)
			os.Exit(1)
		}
	}

	filename := strings.TrimSpace(flag.Arg(0))
	if filename == "" {
		usage()
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read Gopkg.lock", err)
		os.Exit(1)
	}

	raw := Lock{}
	err = toml.Unmarshal(content, &raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse Gopkg.lock", err)
		os.Exit(1)
	}

	if outputFile != nil {
		fmt.Fprintf(outputFile, `# This file is autogenerated with dep2bazel, do not edit.
load("@io_bazel_rules_go//go:def.bzl", "go_repository")

def go_deps():
`)
	}

	for _, lp := range raw.Projects {
		remote := lp.Name
		if lp.Source != "" {
			remote = lp.Source
		}
		root, err := vcs.RepoRootForImportPath(remote, false)
		if err != nil {
			fmt.Println(err)
			continue
		}
		importpath := lp.Name
		if outputFile != nil {
			repo, err := remoteRepository(root.Repo, lp.Revision)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse %v (%v@%v): %v\n", lp.Name, root.Repo, lp.Revision, err)
			} else {
				fmt.Fprint(outputFile, repo.GetRepoString(bazelName(importpath), importpath))
			}
		}
		if *outputGopathRoot != "" && *bazelOutputRoot != "" {
			dirpath, dir := path.Split(importpath)
			dirpath = path.Join(*outputGopathRoot, "src", dirpath)
			err = os.MkdirAll(dirpath, 0775)
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to create directory", err)
				os.Exit(1)
			}
			symlinkName := path.Join(dirpath, dir)
			bazelPath := path.Join(*bazelOutputRoot, "external", bazelName(importpath))
			//fmt.Fprintf(os.Stderr, "Creating symlink %v -> %v\n", symlinkName, bazelPath)
			err = os.Symlink(bazelPath, symlinkName)
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to create symlink", err)
				os.Exit(1)
			}
		}
	}

	if *outputGopathRoot != "" && *bazelOutputRoot != "" {
		dirpath, dir := path.Split(*goPrefix)
		dirpath = path.Join(*outputGopathRoot, "src", dirpath)
		err = os.MkdirAll(dirpath, 0775)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to create directory", err)
			os.Exit(1)
		}
		symlinkName := path.Join(dirpath, dir)
		//fmt.Fprintf(os.Stderr, "Creating symlink %v -> %v\n", symlinkName, sourceDirectory)
		err = os.Symlink(*sourceDirectory, symlinkName)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to create symlink", err)
			os.Exit(1)
		}
	}
}
