package golang

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Platform struct {}

// Returns whether the given file or directory exists or not
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func decodeUrl(path string) (string, error) {
	var urlLocation string
	if strings.HasPrefix(path, "http://") {
		urlLocation = path[7:]
	} else if strings.HasPrefix(path, "https://") {
		urlLocation = path[8:]
	} else {
		urlLocation = path
	}

	if len(urlLocation) < 2 {
		return "", errors.New("ChaincodeSpec's path/URL invalid")
	}

	if strings.LastIndex(urlLocation, "/") == len(urlLocation)-1 {
		urlLocation = urlLocation[:len(urlLocation)-1]
	}

	return urlLocation, nil
}

func getGopath() (string, error) {
	env, err := getGoEnv()
	if err != nil {
		return "", err
	}
	// Only take the first element of GOPATH
	splitGoPath := filepath.SplitList(env["GOPATH"])
	if len(splitGoPath) == 0 {
		return "", fmt.Errorf("invalid GOPATH environment variable value: %s", env["GOPATH"])
	}
	return splitGoPath[0], nil
}

func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

// Name returns the name of this platform
func (goPlatform *Platform) Name() string {
	return pb.ChaincodeSpec_GOLANG.String()
}

// ValidateSpec validates Go chaincodes
func (goPlatform *Platform) ValidatePath(rawPath string) error {
	path, err := url.Parse(rawPath)
	if err != nil || path == nil {
		return fmt.Errorf("invalid path: %s", err)
	}

	//we have no real good way of checking existence of remote urls except by downloading and testing
	//which we do later anyway. But we *can* - and *should* - test for existence of local paths.
	//Treat empty scheme as a local filesystem path
	if path.Scheme == "" {
		gopath, err := getGopath()
		if err != nil {
			return err
		}
		pathToCheck := filepath.Join(gopath, "src", rawPath)
		exists, err := pathExists(pathToCheck)
		if err != nil {
			return fmt.Errorf("error validating chaincode path: %s", err)
		}
		if !exists {
			return fmt.Errorf("path to chaincode does not exist: %s", pathToCheck)
		}
	}
	return nil
}

func (goPlatform *Platform) ValidateCodePackage(code []byte) error {

	if len(code) == 0 {
		// Nothing to validate if no CodePackage was included
		return nil
	}

	// FAB-2122: Scan the provided tarball to ensure it only contains source-code under
	// /src/$packagename.  We do not want to allow something like ./pkg/shady.a to be installed under
	// $GOPATH within the container.  Note, we do not look deeper than the path at this time
	// with the knowledge that only the go/cgo compiler will execute for now.  We will remove the source
	// from the system after the compilation as an extra layer of protection.
	//
	// It should be noted that we cannot catch every threat with these techniques.  Therefore,
	// the container itself needs to be the last line of defense and be configured to be
	// resilient in enforcing constraints. However, we should still do our best to keep as much
	// garbage out of the system as possible.
	re := regexp.MustCompile(`^(/)?(src|META-INF)/.*`)
	is := bytes.NewReader(code)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return fmt.Errorf("failure opening codepackage gzip stream: %s", err)
	}
	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err != nil {
			// We only get here if there are no more entries to scan
			break
		}

		// --------------------------------------------------------------------------------------
		// Check name for conforming path
		// --------------------------------------------------------------------------------------
		if !re.MatchString(header.Name) {
			return fmt.Errorf("illegal file detected in payload: \"%s\"", header.Name)
		}

		// --------------------------------------------------------------------------------------
		// Check that file mode makes sense
		// --------------------------------------------------------------------------------------
		// Acceptable flags:
		//      ISREG      == 0100000
		//      -rw-rw-rw- == 0666
		//
		// Anything else is suspect in this context and will be rejected
		// --------------------------------------------------------------------------------------
		if header.Mode&^0100666 != 0 {
			return fmt.Errorf("illegal file mode detected for file %s: %o", header.Name, header.Mode)
		}
	}

	return nil
}

// Vendor any packages that are not already within our chaincode's primary package
// or vendored by it.  We take the name of the primary package and a list of files
// that have been previously determined to comprise the package's dependencies.
// For anything that needs to be vendored, we simply update its path specification.
// Everything else, we pass through untouched.
func vendorDependencies(pkg string, files Sources) {

	exclusions := make([]string, 0)
	elements := strings.Split(pkg, "/")

	// --------------------------------------------------------------------------------------
	// First, add anything already vendored somewhere within our primary package to the
	// "exclusions".  For a package "foo/bar/baz", we want to ensure we don't auto-vendor
	// any of the following:
	//
	//     [ "foo/vendor", "foo/bar/vendor", "foo/bar/baz/vendor"]
	//
	// and we therefore employ a recursive path building process to form this list
	// --------------------------------------------------------------------------------------
	prev := filepath.Join("src")
	for _, element := range elements {
		curr := filepath.Join(prev, element)
		vendor := filepath.Join(curr, "vendor")
		exclusions = append(exclusions, vendor)
		prev = curr
	}

	// --------------------------------------------------------------------------------------
	// Next add our primary package to the list of "exclusions"
	// --------------------------------------------------------------------------------------
	exclusions = append(exclusions, filepath.Join("src", pkg))

	count := len(files)
	sem := make(chan bool, count)

	// --------------------------------------------------------------------------------------
	// Now start a parallel process which checks each file in files to see if it matches
	// any of the excluded patterns.  Any that match are renamed such that they are vendored
	// under src/$pkg/vendor.
	// --------------------------------------------------------------------------------------
	vendorPath := filepath.Join("src", pkg, "vendor")
	for i, file := range files {
		go func(i int, file SourceDescriptor) {
			excluded := false

			for _, exclusion := range exclusions {
				if strings.HasPrefix(file.Name, exclusion) == true {
					excluded = true
					break
				}
			}

			if excluded == false {
				origName := file.Name
				file.Name = strings.Replace(origName, "src", vendorPath, 1)
				fmt.Println("vendoring %s -> %s", origName, file.Name)
			}

			files[i] = file
			sem <- true
		}(i, file)
	}

	for i := 0; i < count; i++ {
		<-sem
	}
}

func (s SourceMap) Sources() Sources {
	var sources Sources
	for _, src := range s {
		sources = append(sources, src)
	}

	sort.Sort(sources)
	return sources
}

func (s SourceMap) Directories() []string {
	dirMap := map[string]bool{}
	for entryName := range s {
		dir := path.Dir(entryName)
		for dir != "." && !dirMap[dir] {
			dirMap[dir] = true
			dir = path.Dir(dir)
		}
	}

	var dirs []string
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	return dirs
}

const c_ISDIR = 040000

var gzipCompressionLevel = gzip.DefaultCompression

// Generates a deployment payload for GOLANG as a series of src/$pkg entries in .tar.gz format
func (p *Platform) GetDeploymentPayload(codePath string) ([]byte, error) {
	codeDescriptor, err := DescribeCode(codePath)
	if err != nil {
		return nil, err
	}

	fileMap, err := findSource(codeDescriptor)
	if err != nil {
		return nil, err
	}

	var dependencyPackageInfo []PackageInfo
	if !codeDescriptor.Module {
		for _, dist := range distributions() {
			pi, err := gopathDependencyPackageInfo(dist.goos, dist.goarch, codeDescriptor.Path)
			if err != nil {
				return nil, err
			}
			dependencyPackageInfo = append(dependencyPackageInfo, pi...)
		}
	}

	for _, pkg := range dependencyPackageInfo {
		for _, filename := range pkg.Files() {
			sd := SourceDescriptor{
				Name: path.Join("src", pkg.ImportPath, filename),
				Path: filepath.Join(pkg.Dir, filename),
			}
			fileMap[sd.Name] = sd
		}
	}

	payload := bytes.NewBuffer(nil)
	gw, err := gzip.NewWriterLevel(payload, gzipCompressionLevel)
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(gw)

	// Create directories so they get sane ownership and permissions
	for _, dirname := range fileMap.Directories() {
		err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     dirname + "/",
			Mode:     c_ISDIR | 0755,
			Uid:      500,
			Gid:      500,
		})
		if err != nil {
			return nil, err
		}
	}

	for _, file := range fileMap.Sources() {
		err = WriteFileToPackage(file.Path, file.Name, tw)
		if err != nil {
			return nil, fmt.Errorf("Error writing %s to tar: %s", file.Name, err)
		}
	}

	err = tw.Close()
	if err == nil {
		err = gw.Close()
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create tar for chaincode")
	}

	return payload.Bytes(), nil
}

const staticLDFlagsOpts = "-ldflags \"-linkmode external -extldflags '-static'\""

func (p *Platform) NormalizePath(rawPath string) (string, error) {
	modInfo, err := moduleInfo(rawPath)
	if err != nil {
		return "", err
	}

	// not a module
	if modInfo == nil {
		return rawPath, nil
	}

	return modInfo.ImportPath, nil
}


