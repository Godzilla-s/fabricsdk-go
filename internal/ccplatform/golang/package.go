package golang


import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var includeFileTypes = map[string]bool{
	".c":    true,
	".h":    true,
	".s":    true,
	".go":   true,
	".yaml": true,
	".json": true,
}

func getCodeFromFS(path string) (codegopath string, err error) {

	gopath, err := getGopath()
	if err != nil {
		return "", err
	}

	tmppath := filepath.Join(gopath, "src", path)
	if err := IsCodeExist(tmppath); err != nil {
		return "", fmt.Errorf("code does not exist %s", err)
	}

	return gopath, nil
}

type SourceDescriptor struct {
	Name, Path string
	IsMetadata bool
	Info       os.FileInfo
}
type SourceMap map[string]SourceDescriptor

type Sources []SourceDescriptor

func (s Sources) Len() int {
	return len(s)
}

func (s Sources) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Sources) Less(i, j int) bool {
	return strings.Compare(s[i].Name, s[j].Name) < 0
}

func findSource(cd *CodeDescriptor) (SourceMap, error) {
	sources := SourceMap{}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Allow import of the top level chaincode directory into chaincode code package
			if path == cd.Source {
				return nil
			}

			// Allow import of META-INF metadata directories into chaincode code package tar.
			// META-INF directories contain chaincode metadata artifacts such as statedb index definitions
			if cd.isMetadata(path) {
				return nil
			}

			// include everything except hidden dirs when we're not vendoring
			if cd.Module && !strings.HasPrefix(info.Name(), ".") {
				return nil
			}

			// Do not import any other directories into chaincode code package
			return filepath.SkipDir
		}

		relativeRoot := cd.Source
		if cd.isMetadata(path) {
			relativeRoot = cd.MetadataRoot
		}

		name, err := filepath.Rel(relativeRoot, path)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate relative path for %s", path)
		}

		switch {
		case cd.isMetadata(path):
			// Skip hidden files in metadata
			if strings.HasPrefix(info.Name(), ".") {
				return nil
			}
			name = filepath.Join("META-INF", name)
			err := validateMetadata(name, path)
			if err != nil {
				return err
			}
		case cd.Module:
			name = filepath.Join("src", name)
		default:
			// skip top level go.mod and go.sum when not in module mode
			if name == "go.mod" || name == "go.sum" {
				return nil
			}
			name = filepath.Join("src", cd.Path, name)
		}

		name = filepath.ToSlash(name)
		sources[name] = SourceDescriptor{Name: name, Path: path}
		return nil
	}

	if err := filepath.Walk(cd.Source, walkFn); err != nil {
		return nil, errors.Wrap(err, "walk failed")
	}

	return sources, nil
}

func validateMetadata(name, path string) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	// Validate metadata file for inclusion in tar
	// Validation is based on the passed filename with path
	err = ValidateMetadataFile(filepath.ToSlash(name), contents)
	if err != nil {
		return err
	}

	return nil
}


// isMetadataDir checks to see if the current path is in the META-INF directory at the root of the chaincode directory
func isMetadataDir(path, tld string) bool {
	return strings.HasPrefix(path, filepath.Join(tld, "META-INF"))
}

type CodeDescriptor struct {
	Source       string // absolute path of the source to package
	MetadataRoot string // absolute path META-INF
	Path         string // import path of the package
	Module       bool   // does this represent a go module
}

func (cd CodeDescriptor) isMetadata(path string) bool {
	return strings.HasPrefix(
		filepath.Clean(path),
		filepath.Clean(cd.MetadataRoot),
	)
}

// DescribeCode returns GOPATH and package information.
func DescribeCode(path string) (*CodeDescriptor, error) {
	if path == "" {
		return nil, errors.New("cannot collect files from empty chaincode path")
	}

	// Use the module root as the source path for go modules
	modInfo, err := moduleInfo(path)
	if err != nil {
		return nil, err
	}

	if modInfo != nil {
		// calculate where the metadata should be relative to module root
		relImport, err := filepath.Rel(modInfo.ModulePath, modInfo.ImportPath)
		if err != nil {
			return nil, err
		}

		return &CodeDescriptor{
			Module:       true,
			MetadataRoot: filepath.Join(modInfo.Dir, relImport, "META-INF"),
			Path:         modInfo.ImportPath,
			Source:       modInfo.Dir,
		}, nil
	}

	return describeGopath(path)
}

func describeGopath(importPath string) (*CodeDescriptor, error) {
	output, err := exec.Command("go", "list", "-f", "{{.Dir}}", importPath).Output()
	if err != nil {
		return nil, errors.WithMessage(err, "'go list' failed")
	}
	sourcePath := filepath.Clean(strings.TrimSpace(string(output)))

	return &CodeDescriptor{
		Path:         importPath,
		MetadataRoot: filepath.Join(sourcePath, "META-INF"),
		Source:       sourcePath,
	}, nil
}

// dist holds go "distribution" information. The full list of distributions can
// be obtained with `go tool dist list.
type dist struct{ goos, goarch string }

// distributions returns the list of OS and ARCH combinations that we calcluate
// deps for.
func distributions() []dist {
	// pre-populate linux architecutures
	dists := map[dist]bool{
		{goos: "linux", goarch: "amd64"}: true,
	}

	// add local OS and ARCH
	dists[dist{goos: runtime.GOOS, goarch: runtime.GOARCH}] = true

	var list []dist
	for d := range dists {
		list = append(list, d)
	}

	return list
}
