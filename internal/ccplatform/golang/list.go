package golang


import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// necessary to calculate chaincode package dependencies.
type PackageInfo struct {
	ImportPath     string
	Dir            string
	GoFiles        []string
	Goroot         bool
	CFiles         []string
	CgoFiles       []string
	HFiles         []string
	SFiles         []string
	IgnoredGoFiles []string
	Incomplete     bool
}

func (p PackageInfo) Files() []string {
	var files []string
	files = append(files, p.GoFiles...)
	files = append(files, p.CFiles...)
	files = append(files, p.CgoFiles...)
	files = append(files, p.HFiles...)
	files = append(files, p.SFiles...)
	files = append(files, p.IgnoredGoFiles...)
	return files
}

//runProgram non-nil Env, timeout (typically secs or millisecs), program name and args
func runProgram(env Env, timeout time.Duration, pgm string, args ...string) ([]byte, error) {
	if env == nil {
		return nil, fmt.Errorf("<%s, %v>: nil env provided", pgm, args)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, pgm, args...)
	cmd.Env = flattenEnv(env)
	stdErr := &bytes.Buffer{}
	cmd.Stderr = stdErr

	out, err := cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %s", timeout)
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"command <%s %s>: failed with error: \"%s\"\n%s",
				pgm,
				strings.Join(args, " "),
				err,
				string(stdErr.Bytes()))
	}
	return out, nil
}

// Logic inspired by: https://dave.cheney.net/2014/09/14/go-list-your-swiss-army-knife
func list(env Env, template, pkg string) ([]string, error) {
	if env == nil {
		env = getEnv()
	}

	lst, err := runProgram(env, 60*time.Second, "go", "list", "-f", template, pkg)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.Trim(string(lst), "\n"), "\n"), nil
}

func listDeps(env Env, pkg string) ([]string, error) {
	return list(env, "{{ join .Deps \"\\n\"}}", pkg)
}

func listImports(env Env, pkg string) ([]string, error) {
	return list(env, "{{ join .Imports \"\\n\"}}", pkg)
}

type ModuleInfo struct {
	Dir        string
	GoMod      string
	ImportPath string
	ModulePath string
}

const listTimeout = 3 * time.Minute

// listModuleInfo extracts module information for the curent working directory.
func listModuleInfo(extraEnv ...string) (*ModuleInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-json", ".")
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Env = append(cmd.Env, extraEnv...)

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.WithMessage(err, "'go list' failed")
	}

	var moduleData struct {
		ImportPath string
		Module     struct {
			Dir   string
			Path  string
			GoMod string
		}
	}

	if err := json.Unmarshal(output, &moduleData); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal output from 'go list'")
	}

	return &ModuleInfo{
		Dir:        moduleData.Module.Dir,
		GoMod:      moduleData.Module.GoMod,
		ImportPath: moduleData.ImportPath,
		ModulePath: moduleData.Module.Path,
	}, nil
}

func regularFileExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err != nil:
		return false, err
	default:
		return fi.Mode().IsRegular(), nil
	}
}

func moduleInfo(path string) (*ModuleInfo, error) {
	entryWD, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get working directory")
	}

	// directory doesn't exist so unlikely to be a module
	if err := os.Chdir(path); err != nil {
		return nil, nil
	}
	defer func() {
		if err := os.Chdir(entryWD); err != nil {
			panic(fmt.Sprintf("failed to restore working directory: %s", err))
		}
	}()

	// Using `go list -m -f '{{ if .Main }}{{.GoMod}}{{ end }}' all` may try to
	// generate a go.mod when a vendor tool is in use. To avoid that behavior
	// we use `go env GOMOD` followed by an existence check.
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to determine module root")
	}

	modExists, err := regularFileExists(strings.TrimSpace(string(output)))
	if err != nil {
		return nil, err
	}
	if !modExists {
		return nil, nil
	}

	return listModuleInfo()
}

// gopathDependencyPackageInfo extracts dependency information for
// specified package.
func gopathDependencyPackageInfo(goos, goarch, pkg string) ([]PackageInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-json", pkg)
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.WithMessage(err, "'go list -deps' failed")
	}
	decoder := json.NewDecoder(stdout)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var list []PackageInfo
	for {
		var packageInfo PackageInfo
		err := decoder.Decode(&packageInfo)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if packageInfo.Incomplete {
			return nil, fmt.Errorf("failed to calculate dependencies: incomplete package: %s", packageInfo.ImportPath)
		}
		if packageInfo.Goroot {
			continue
		}

		list = append(list, packageInfo)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, errors.Wrapf(err, "listing deps for package %s failed", pkg)
	}

	return list, nil
}

