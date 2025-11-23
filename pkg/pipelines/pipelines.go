package pipelines

import (
	"fmt"
	"strings"

	"github.com/greboid/dfo/pkg/util"
)

type Step struct {
	Name    string
	Content string
}

// PipelineResult contains the generated steps and their dependencies
type PipelineResult struct {
	Steps     []Step
	BuildDeps []string // Temporary packages needed during build, removed after
	Packages  []string // Packages that need to persist in the image
}

type Pipeline func(params map[string]any) (PipelineResult, error)

var Registry = map[string]Pipeline{
	"create-user":              CreateUser,
	"set-ownership":            SetOwnership,
	"download-verify-extract":  DownloadVerifyExtract,
	"make-executable":          MakeExecutable,
	"clone":                    Clone,
	"clone-and-build-go":       CloneAndBuildGo,
	"build-go-static":          BuildGo,
	"clone-and-build-rust":     CloneAndBuildRust,
	"clone-and-build-make":     CloneAndBuildMake,
	"clone-and-build-autoconf": CloneAndBuildAutoconf,
	"setup-users-groups":       SetupUsersGroups,
	"create-directories":       CreateDirectories,
	"copy-files":               CopyFiles,
}

func CreateUser(params map[string]any) (PipelineResult, error) {
	username, err := util.ValidateStringParam(params, "username")
	if err != nil {
		return PipelineResult{}, err
	}

	uidInt, err := util.ValidateIntParam(params, "uid")
	if err != nil {
		return PipelineResult{}, err
	}

	gidInt, err := util.ValidateIntParam(params, "gid")
	if err != nil {
		return PipelineResult{}, err
	}

	return PipelineResult{
		Steps: []Step{{
			Name: "Create application user",
			Content: fmt.Sprintf("RUN addgroup -g %d %s && \\\n    adduser -D -u %d -G %s %s\n",
				gidInt, username, uidInt, username, username),
		}},
		BuildDeps: []string{"busybox"},
	}, nil
}

func SetOwnership(params map[string]any) (PipelineResult, error) {
	user, err := util.ValidateStringParam(params, "user")
	if err != nil {
		return PipelineResult{}, err
	}

	group, err := util.ValidateStringParam(params, "group")
	if err != nil {
		return PipelineResult{}, err
	}

	path, err := util.ValidateStringParam(params, "path")
	if err != nil {
		return PipelineResult{}, err
	}

	return PipelineResult{
		Steps: []Step{{
			Name:    "Change file ownership",
			Content: fmt.Sprintf("RUN chown -R %s:%s %s\n", user, group, path),
		}},
		BuildDeps: []string{"busybox"},
	}, nil
}

func DownloadVerifyExtract(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("download-verify-extract", params); err != nil {
		return PipelineResult{}, err
	}

	url, err := util.ValidateStringParam(params, "url")
	if err != nil {
		return PipelineResult{}, err
	}

	destination, err := util.ValidateStringParam(params, "destination")
	if err != nil {
		return PipelineResult{}, err
	}

	checksum, err := util.ValidateOptionalStringParamStrict(params, "checksum", "")
	if err != nil {
		return PipelineResult{}, err
	}
	checksumURL, err := util.ValidateOptionalStringParamStrict(params, "checksum-url", "")
	if err != nil {
		return PipelineResult{}, err
	}
	checksumPattern, err := util.ValidateOptionalStringParamStrict(params, "checksum-pattern", "")
	if err != nil {
		return PipelineResult{}, err
	}

	hasChecksum := checksum != ""
	hasChecksumURL := checksumURL != ""

	if err := util.ValidateMutuallyExclusiveRequired(hasChecksum, hasChecksumURL, "checksum", "checksum-url"); err != nil {
		return PipelineResult{}, err
	}

	extractDir, err := util.ValidateOptionalStringParamStrict(params, "extract-dir", "")
	if err != nil {
		return PipelineResult{}, err
	}
	stripComponents, err := util.ValidateOptionalIntParam(params, "strip-components", 0)
	if err != nil {
		return PipelineResult{}, err
	}

	if extractDir != "" {
		if err := validateArchiveFormat(destination); err != nil {
			return PipelineResult{}, err
		}
	}

	var cmdParts []string

	if hasChecksumURL {
		checksumDest := destination + ".checksum"
		cmdParts = append(cmdParts, fmt.Sprintf("curl -fsSL -o %s %q", checksumDest, checksumURL))
	}

	cmdParts = append(cmdParts, fmt.Sprintf("curl -fsSL -o %s %q", destination, url))

	var verifyCmd string
	if hasChecksumURL {
		checksumDest := destination + ".checksum"
		if checksumPattern != "" {
			verifyCmd = fmt.Sprintf("echo \"$(grep %q %s | awk '{print $1}') *%s\" | sha256sum -wc -",
				checksumPattern, checksumDest, destination)
		} else {
			verifyCmd = fmt.Sprintf("echo \"$(cat %s | awk '{print $1}') *%s\" | sha256sum -wc -",
				checksumDest, destination)
		}
	} else {
		verifyCmd = fmt.Sprintf("echo %q | sha256sum -c", checksum+"  "+destination)
	}
	cmdParts = append(cmdParts, verifyCmd)

	if extractDir != "" {
		extractCmd := buildExtractCommand(destination, extractDir, stripComponents)
		cmdParts = append(cmdParts, extractCmd)
	}

	combinedCmd := strings.Join(cmdParts, " && \\\n    ")

	// Determine build deps based on what's needed
	buildDeps := []string{"busybox", "curl"}
	if extractDir != "" && strings.HasSuffix(destination, ".zip") {
		buildDeps = append(buildDeps, "unzip")
	}

	return PipelineResult{
		Steps: []Step{
			{
				Name:    "Download, verify and extract",
				Content: fmt.Sprintf("RUN %s\n", combinedCmd),
			},
		},
		BuildDeps: buildDeps,
	}, nil
}

func MakeExecutable(params map[string]any) (PipelineResult, error) {
	path, err := util.ValidateStringParam(params, "path")
	if err != nil {
		return PipelineResult{}, err
	}

	return PipelineResult{
		Steps: []Step{{
			Name:    "Set executable permission",
			Content: fmt.Sprintf("RUN chmod +x %s\n", path),
		}},
		BuildDeps: []string{"busybox"},
	}, nil
}

func buildExtractCommand(destination, extractDir string, stripComponents int) string {
	mkdirCmd := fmt.Sprintf("mkdir -p %q", extractDir)

	if strings.HasSuffix(destination, ".zip") {
		return fmt.Sprintf("%s && unzip -q %q -d %q", mkdirCmd, destination, extractDir)
	}

	if isTarArchive(destination) {
		return fmt.Sprintf("%s && tar -xf %q -C %q --strip-components=%d",
			mkdirCmd, destination, extractDir, stripComponents)
	}

	return fmt.Sprintf("echo \"Unsupported archive format: %s\" && exit 1", destination)
}

func isTarArchive(filename string) bool {
	tarExtensions := []string{
		".tar", ".tar.gz", ".tgz",
		".tar.bz2", ".tbz2",
		".tar.xz", ".txz",
	}

	for _, ext := range tarExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

func validateArchiveFormat(filename string) error {
	if strings.HasSuffix(filename, ".zip") {
		return nil
	}
	if isTarArchive(filename) {
		return nil
	}
	return fmt.Errorf("unsupported archive format: %s (supported: .zip, .tar, .tar.gz, .tgz, .tar.bz2, .tbz2, .tar.xz, .txz)", filename)
}

func extractGitHubOwnerRepo(repoURL string) string {
	if !strings.Contains(repoURL, "github.com") {
		return ""
	}

	url := strings.TrimPrefix(repoURL, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")

	url = strings.TrimPrefix(url, "github.com/")
	url = strings.TrimPrefix(url, "github.com:")

	url = strings.TrimSuffix(url, ".git")

	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}

	return ""
}

func extractRepoWorkdir(repo string, params map[string]any) (string, error) {
	defaultWorkdir := "/src"
	if ownerRepo := extractGitHubOwnerRepo(repo); ownerRepo != "" {
		defaultWorkdir = "/src/" + ownerRepo
	}
	return util.ValidateOptionalStringParamStrict(params, "workdir", defaultWorkdir)
}

func generateMakeStep(workdir string, makeSteps []string) Step {
	makeCmd := strings.Join(makeSteps, "; \\\n    ")
	return Step{
		Name:    "Build with make",
		Content: fmt.Sprintf("WORKDIR %s\nRUN %s\n", workdir, makeCmd),
	}
}

func generateStripStep(workdir string) Step {
	return Step{
		Name:    "Strip binaries",
		Content: fmt.Sprintf("RUN find %s -type f -executable -exec strip {} + 2>/dev/null || true\n", workdir),
	}
}

func generateGoModDownloadStep(workdir string) Step {
	return Step{
		Name:    "Download dependencies",
		Content: fmt.Sprintf("WORKDIR %s\nRUN go mod download\n", workdir),
	}
}

func generateCloneStep(repo, tag, commit, workdir string) Step {
	var cloneCmd string
	if commit != "" {
		cloneCmd = fmt.Sprintf("RUN git clone %q %s && \\\n    cd %s && \\\n    git checkout %s\n", repo, workdir, workdir, commit)
	} else if tag != "" {
		cloneCmd = fmt.Sprintf("RUN git clone --depth=1 --branch %s %q %s\n", tag, repo, workdir)
	} else {
		ownerRepo := extractGitHubOwnerRepo(repo)
		if ownerRepo != "" {
			cloneCmd = fmt.Sprintf("RUN git clone --depth=1 --branch {{github_tag %q}} %q %s\n", ownerRepo, repo, workdir)
		} else {
			cloneCmd = fmt.Sprintf("RUN git clone --depth=1 %q %s\n", repo, workdir)
		}
	}

	return Step{
		Name:    "Clone repository",
		Content: cloneCmd,
	}
}

func Clone(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("clone", params); err != nil {
		return PipelineResult{}, err
	}

	repo, err := util.ValidateStringParam(params, "repo")
	if err != nil {
		return PipelineResult{}, err
	}

	workdir, err := util.ValidateOptionalStringParamStrict(params, "workdir", "/src")
	if err != nil {
		return PipelineResult{}, err
	}

	tag, err := util.ValidateOptionalStringParamStrict(params, "tag", "")
	if err != nil {
		return PipelineResult{}, err
	}
	commit, err := util.ValidateOptionalStringParamStrict(params, "commit", "")
	if err != nil {
		return PipelineResult{}, err
	}

	if tag != "" && commit != "" {
		return PipelineResult{}, fmt.Errorf("cannot specify both tag and commit")
	}

	return PipelineResult{
		Steps:     []Step{generateCloneStep(repo, tag, commit, workdir)},
		BuildDeps: []string{"git"},
	}, nil
}

func generateGoBuildStep(pkg, output, extraLdflags, extraTags string, cgo bool) Step {
	ldflags := `-s -w -extldflags "-static"`
	if extraLdflags != "" {
		ldflags += " " + extraLdflags
	}
	tags := "netgo,osusergo"
	if extraTags != "" {
		tags += "," + extraTags
	}
	cgoEnabled := "1"
	if !cgo {
		cgoEnabled = "0"
	}
	return Step{
		Name:    "Build binary",
		Content: fmt.Sprintf("RUN CGO_ENABLED=%s go build -trimpath -tags '%s' -ldflags='%s' -o %s %s\n", cgoEnabled, tags, ldflags, output, pkg),
	}
}

func generateLicenseStep(pkg, output, ignore string) Step {
	noticesPath := "/notices" + output
	var licenseCmd string
	if ignore != "" {
		licenseCmd = fmt.Sprintf("RUN go run github.com/google/go-licenses@latest save %s --save_path=%s --ignore %s\n", pkg, noticesPath, ignore)
	} else {
		licenseCmd = fmt.Sprintf("RUN go run github.com/google/go-licenses@latest save %s --save_path=%s\n", pkg, noticesPath)
	}
	return Step{
		Name:    "Generate license notices",
		Content: licenseCmd,
	}
}

func CloneAndBuildGo(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("clone-and-build-go", params); err != nil {
		return PipelineResult{}, err
	}

	repo, err := util.ValidateStringParam(params, "repo")
	if err != nil {
		return PipelineResult{}, err
	}

	pkg, err := util.ValidateOptionalStringParamStrict(params, "package", ".")
	if err != nil {
		return PipelineResult{}, err
	}

	output, err := util.ValidateOptionalStringParamStrict(params, "output", "/main")
	if err != nil {
		return PipelineResult{}, err
	}

	tag, err := util.ValidateOptionalStringParamStrict(params, "tag", "")
	if err != nil {
		return PipelineResult{}, err
	}

	goTags, err := util.ValidateOptionalStringParamStrict(params, "go-tags", "")
	if err != nil {
		return PipelineResult{}, err
	}

	cgo, err := util.ValidateOptionalBoolParam(params, "cgo", false)
	if err != nil {
		return PipelineResult{}, err
	}

	ignore, err := util.ValidateOptionalStringParamStrict(params, "ignore", "")
	if err != nil {
		return PipelineResult{}, err
	}

	workdir, err := extractRepoWorkdir(repo, params)
	if err != nil {
		return PipelineResult{}, err
	}

	return PipelineResult{
		Steps: []Step{
			generateCloneStep(repo, tag, "", workdir),
			generateGoModDownloadStep(workdir),
			generateGoBuildStep(pkg, output, "", goTags, cgo),
			generateLicenseStep(pkg, output, ignore),
		},
		BuildDeps: []string{"git", "go"},
	}, nil
}

func BuildGo(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("build-go-static", params); err != nil {
		return PipelineResult{}, err
	}

	repo, err := util.ValidateStringParam(params, "repo")
	if err != nil {
		return PipelineResult{}, err
	}

	workdir, err := extractRepoWorkdir(repo, params)
	if err != nil {
		return PipelineResult{}, err
	}

	pkg, err := util.ValidateOptionalStringParamStrict(params, "package", ".")
	if err != nil {
		return PipelineResult{}, err
	}

	output, err := util.ValidateOptionalStringParamStrict(params, "output", "/main")
	if err != nil {
		return PipelineResult{}, err
	}

	ignore, err := util.ValidateOptionalStringParamStrict(params, "ignore", "")
	if err != nil {
		return PipelineResult{}, err
	}
	tag, err := util.ValidateOptionalStringParamStrict(params, "tag", "")
	if err != nil {
		return PipelineResult{}, err
	}

	goTags, err := util.ValidateOptionalStringParamStrict(params, "go-tags", "")
	if err != nil {
		return PipelineResult{}, err
	}

	cgo, err := util.ValidateOptionalBoolParam(params, "cgo", false)
	if err != nil {
		return PipelineResult{}, err
	}

	return PipelineResult{
		Steps: []Step{
			generateCloneStep(repo, tag, "", workdir),
			generateGoModDownloadStep(workdir),
			generateGoBuildStep(pkg, output, "", goTags, cgo),
			generateLicenseStep(pkg, output, ignore),
		},
		BuildDeps: []string{"git", "go"},
	}, nil
}

func CloneAndBuildRust(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("clone-and-build-rust", params); err != nil {
		return PipelineResult{}, err
	}

	repo, err := util.ValidateStringParam(params, "repo")
	if err != nil {
		return PipelineResult{}, err
	}

	workdir, err := extractRepoWorkdir(repo, params)
	if err != nil {
		return PipelineResult{}, err
	}

	features, err := util.ValidateOptionalStringParamStrict(params, "features", "")
	if err != nil {
		return PipelineResult{}, err
	}
	output, err := util.ValidateOptionalStringParamStrict(params, "output", "/main")
	if err != nil {
		return PipelineResult{}, err
	}

	tag, err := util.ValidateOptionalStringParamStrict(params, "tag", "")
	if err != nil {
		return PipelineResult{}, err
	}

	patches := util.ExtractStringSlice(params, "patches")

	steps := []Step{
		generateCloneStep(repo, tag, "", workdir),
	}

	buildDeps := []string{"busybox", "git", "cargo", "rust"}
	if len(patches) > 0 {
		buildDeps = append(buildDeps, "patch")
	}

	for _, patch := range patches {
		steps = append(steps, Step{
			Name:    fmt.Sprintf("Apply patch %s", patch),
			Content: fmt.Sprintf("COPY %s %s/\nRUN cd %s && patch -p1 < %s\n", patch, workdir, workdir, patch),
		})
	}

	var buildCmd string
	if features != "" {
		buildCmd = fmt.Sprintf("RUN cd %s && cargo build --release --target x86_64-unknown-linux-musl --features %s\n", workdir, features)
	} else {
		buildCmd = fmt.Sprintf("RUN cd %s && cargo build --release --target x86_64-unknown-linux-musl\n", workdir)
	}

	steps = append(steps, Step{
		Name:    "Build binary",
		Content: buildCmd,
	})

	steps = append(steps, Step{
		Name:    "Copy binary to final location",
		Content: fmt.Sprintf("RUN find %s/target/x86_64-unknown-linux-musl/release -maxdepth 1 -type f -executable -exec cp {} %s \\;\n", workdir, output),
	})

	return PipelineResult{
		Steps:     steps,
		BuildDeps: buildDeps,
	}, nil
}

func CloneAndBuildMake(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("clone-and-build-make", params); err != nil {
		return PipelineResult{}, err
	}

	repo, err := util.ValidateStringParam(params, "repo")
	if err != nil {
		return PipelineResult{}, err
	}

	workdir, err := extractRepoWorkdir(repo, params)
	if err != nil {
		return PipelineResult{}, err
	}

	tag, err := util.ValidateOptionalStringParamStrict(params, "tag", "")
	if err != nil {
		return PipelineResult{}, err
	}

	makeSteps := util.ExtractStringSlice(params, "make-steps")

	strip, err := util.ValidateOptionalBoolParam(params, "strip", true)
	if err != nil {
		return PipelineResult{}, err
	}

	steps := []Step{
		generateCloneStep(repo, tag, "", workdir),
	}

	if len(makeSteps) > 0 {
		steps = append(steps, generateMakeStep(workdir, makeSteps))
	}

	buildDeps := []string{"busybox", "git", "make"}
	if strip {
		steps = append(steps, generateStripStep(workdir))
		buildDeps = append(buildDeps, "binutils")
	}

	return PipelineResult{
		Steps:     steps,
		BuildDeps: buildDeps,
	}, nil
}

func CloneAndBuildAutoconf(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("clone-and-build-autoconf", params); err != nil {
		return PipelineResult{}, err
	}

	repo, err := util.ValidateStringParam(params, "repo")
	if err != nil {
		return PipelineResult{}, err
	}

	workdir, err := extractRepoWorkdir(repo, params)
	if err != nil {
		return PipelineResult{}, err
	}

	tag, err := util.ValidateOptionalStringParamStrict(params, "tag", "")
	if err != nil {
		return PipelineResult{}, err
	}

	configureOptions := util.ExtractStringSlice(params, "configure-options")
	makeSteps := util.ExtractStringSlice(params, "make-steps")

	strip, err := util.ValidateOptionalBoolParam(params, "strip", true)
	if err != nil {
		return PipelineResult{}, err
	}

	steps := []Step{
		generateCloneStep(repo, tag, "", workdir),
	}

	configureCmd := "./configure"
	if len(configureOptions) > 0 {
		configureCmd = fmt.Sprintf("./configure %s", strings.Join(configureOptions, " "))
	}
	steps = append(steps, Step{
		Name:    "Configure",
		Content: fmt.Sprintf("WORKDIR %s\nRUN %s\n", workdir, configureCmd),
	})

	if len(makeSteps) > 0 {
		steps = append(steps, Step{
			Name:    "Build with make",
			Content: fmt.Sprintf("RUN %s\n", strings.Join(makeSteps, "; \\\n    ")),
		})
	}

	buildDeps := []string{"busybox", "git", "autoconf", "automake", "make"}
	if strip {
		steps = append(steps, generateStripStep(workdir))
		buildDeps = append(buildDeps, "binutils")
	}

	return PipelineResult{
		Steps:     steps,
		BuildDeps: buildDeps,
	}, nil
}

func SetupUsersGroups(params map[string]any) (PipelineResult, error) {
	if err := ValidateParams("setup-users-groups", params); err != nil {
		return PipelineResult{}, err
	}

	rootfs, err := util.ValidateOptionalStringParamStrict(params, "rootfs", "")
	if err != nil {
		return PipelineResult{}, err
	}

	var groups []groupDef
	if g, ok := params["groups"]; ok {
		var err error
		groups, err = parseGroups(g)
		if err != nil {
			return PipelineResult{}, fmt.Errorf("parsing groups: %w", err)
		}
	}

	var users []userDef
	if u, ok := params["users"]; ok {
		var err error
		users, err = parseUsers(u)
		if err != nil {
			return PipelineResult{}, fmt.Errorf("parsing users: %w", err)
		}
	}

	if len(groups) == 0 && len(users) == 0 {
		return PipelineResult{}, fmt.Errorf("no users or groups specified")
	}

	var commands []string

	if rootfs != "" {
		commands = append(commands, fmt.Sprintf("mkdir -p %s/etc", rootfs))
	}

	for _, group := range groups {
		commands = append(commands,
			fmt.Sprintf("echo \"%s:x:%d:\" >> %s/etc/group",
				group.Name, group.GID, rootfs))
	}

	for _, user := range users {
		shell := user.Shell
		if shell == "" {
			shell = "/sbin/nologin"
		}
		home := user.Home
		if home == "" {
			home = "/nonexistent"
		}

		commands = append(commands,
			fmt.Sprintf("echo \"%s:x:%d:%d:%s:%s:%s\" >> %s/etc/passwd",
				user.Username, user.UID, user.GID, user.Username, home, shell, rootfs))

		if home != "/nonexistent" {
			commands = append(commands,
				fmt.Sprintf("mkdir -p %s%s", rootfs, home))
			commands = append(commands,
				fmt.Sprintf("chown %d:%d %s%s", user.UID, user.GID, rootfs, home))
		}
	}

	cmdStr := strings.Join(commands, "; \\\n    ")

	return PipelineResult{
		Steps: []Step{{
			Name:    "Set up users and groups",
			Content: fmt.Sprintf("RUN %s\n", cmdStr),
		}},
		BuildDeps: []string{"busybox"},
	}, nil
}

type groupDef struct {
	Name string
	GID  int
}

type userDef struct {
	Username string
	UID      int
	GID      int
	Home     string
	Shell    string
}

func parseGroups(data any) ([]groupDef, error) {
	return util.ParseArrayParam(data, "groups", func(m map[string]any, i int) (groupDef, error) {
		name, err := util.ExtractRequiredString(m, "name", fmt.Sprintf("group at index %d", i))
		if err != nil {
			return groupDef{}, err
		}

		gid, err := util.ExtractRequiredInt(m, "gid", fmt.Sprintf("group at index %d", i))
		if err != nil {
			return groupDef{}, err
		}

		return groupDef{
			Name: name,
			GID:  gid,
		}, nil
	})
}

func parseUsers(data any) ([]userDef, error) {
	return util.ParseArrayParam(data, "users", func(m map[string]any, i int) (userDef, error) {
		username, err := util.ExtractRequiredString(m, "username", fmt.Sprintf("user at index %d", i))
		if err != nil {
			return userDef{}, err
		}

		uid, err := util.ExtractRequiredInt(m, "uid", fmt.Sprintf("user at index %d", i))
		if err != nil {
			return userDef{}, err
		}

		gid, err := util.ExtractRequiredInt(m, "gid", fmt.Sprintf("user at index %d", i))
		if err != nil {
			return userDef{}, err
		}

		return userDef{
			Username: username,
			UID:      uid,
			GID:      gid,
			Home:     util.ExtractOptionalString(m, "home"),
			Shell:    util.ExtractOptionalString(m, "shell"),
		}, nil
	})
}

func CreateDirectories(params map[string]any) (PipelineResult, error) {
	dirsParam, ok := params["directories"]
	if !ok {
		return PipelineResult{}, fmt.Errorf("directories parameter is required")
	}

	dirs, err := parseDirectories(dirsParam)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("parsing directories: %w", err)
	}

	if len(dirs) == 0 {
		return PipelineResult{}, fmt.Errorf("at least one directory must be specified")
	}

	var commands []string

	var paths []string
	for _, dir := range dirs {
		paths = append(paths, dir.Path)
	}

	commands = append(commands, fmt.Sprintf("mkdir -p %s", strings.Join(paths, " ")))

	for _, dir := range dirs {
		if dir.Permissions != "" {
			commands = append(commands, fmt.Sprintf("chmod %s %s", dir.Permissions, dir.Path))
		}
	}

	cmdStr := strings.Join(commands, "; \\\n    ")

	return PipelineResult{
		Steps: []Step{{
			Name:    "Create directories",
			Content: fmt.Sprintf("RUN %s\n", cmdStr),
		}},
		BuildDeps: []string{"busybox"},
	}, nil
}

type directoryDef struct {
	Path        string
	Permissions string
}

func parseDirectories(data any) ([]directoryDef, error) {
	return util.ParseArrayParam(data, "directories", func(m map[string]any, i int) (directoryDef, error) {
		path, err := util.ExtractRequiredString(m, "path", fmt.Sprintf("directory at index %d", i))
		if err != nil {
			return directoryDef{}, err
		}

		return directoryDef{
			Path:        path,
			Permissions: util.ExtractOptionalString(m, "permissions"),
		}, nil
	})
}

func CopyFiles(params map[string]any) (PipelineResult, error) {
	filesParam, ok := params["files"]
	if !ok {
		return PipelineResult{}, fmt.Errorf("files parameter is required")
	}

	files, err := parseFiles(filesParam)
	if err != nil {
		return PipelineResult{}, fmt.Errorf("parsing files: %w", err)
	}

	if len(files) == 0 {
		return PipelineResult{}, fmt.Errorf("at least one file must be specified")
	}

	var steps []Step
	for _, file := range files {
		var copyCmd strings.Builder
		copyCmd.WriteString("COPY")

		if file.Chown != "" {
			copyCmd.WriteString(fmt.Sprintf(" --chown=%s", file.Chown))
		}
		if file.Chmod != "" {
			copyCmd.WriteString(fmt.Sprintf(" --chmod=%s", file.Chmod))
		}

		copyCmd.WriteString(fmt.Sprintf(" %s %s\n", file.From, file.To))

		steps = append(steps, Step{
			Name:    fmt.Sprintf("Copy %s to %s", file.From, file.To),
			Content: copyCmd.String(),
		})
	}

	return PipelineResult{
		Steps: steps,
	}, nil
}

type fileDef struct {
	From  string
	To    string
	Chown string
	Chmod string
}

func parseFiles(data any) ([]fileDef, error) {
	return util.ParseArrayParam(data, "files", func(m map[string]any, i int) (fileDef, error) {
		from, err := util.ExtractRequiredString(m, "from", fmt.Sprintf("file at index %d", i))
		if err != nil {
			return fileDef{}, err
		}

		to, err := util.ExtractRequiredString(m, "to", fmt.Sprintf("file at index %d", i))
		if err != nil {
			return fileDef{}, err
		}

		return fileDef{
			From:  from,
			To:    to,
			Chown: util.ExtractOptionalString(m, "chown"),
			Chmod: util.ExtractOptionalString(m, "chmod"),
		}, nil
	})
}
