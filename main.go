package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-steputils/tools"
)

const defaultReleaseConfig = `release:
  development_branch: master
changelog:
  path: CHANGELOG.md
  content_template: |-
    {{range .ContentItems}}### {{.EndTaggedCommit.Tag}} ({{.EndTaggedCommit.Date.Format "2006 Jan 02"}})

    {{range .Commits}}* [{{firstChars .Hash 7}}] {{.Message}}
    {{end}}
    {{end}}
  header_template: '## Changelog (Current version: {{.Version}})'
  footer_template: 'Updated: {{.CurrentDate.Format "2006 Jan 02"}}'`

func installedInPath(name string) bool {
	cmd := exec.Command("which", name)
	outBytes, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(outBytes)) != ""
}

func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func main() {
	version := os.Getenv("new_version")
	changelogPth := os.Getenv("changelog_pth")
	releaseConfigContent := os.Getenv("release_config")

	log.Infof("Configs:")
	log.Printf("new_version: %s", version)
	log.Printf("changelog_pth: %s", changelogPth)
	log.Printf("release_config:\n%s", releaseConfigContent)

	if version == "" {
		failf("Next version not defined")
	}

	if changelogPth == "" {
		failf("Changelog path not defined")
	}

	if releaseConfigContent == "" {
		releaseConfigContent = defaultReleaseConfig
	}

	if !installedInPath("releaseman") {
		cmd := command.New("go", "get", "-u", "github.com/bitrise-tools/releaseman")

		log.Infof("\nInstalling releaseman")
		log.Donef("$ %s", cmd.PrintableCommandArgs())

		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			failf("Failed to install releaseman: %s", out)
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		failf("Failed to get working directory")
	}

	releaseConfigPth := filepath.Join(pwd, "release_config.yml")
	if exist, err := pathutil.IsPathExists(releaseConfigPth); err != nil {
		failf("Failed to check if release config file exist: %s", err)
	} else if !exist {
		if err := fileutil.WriteStringToFile(releaseConfigPth, releaseConfigContent); err != nil {
			failf("Failed to create release config file: %s", err)
		}
	}

	// sandbox
	if err := command.New("git", "fetch", "--all", "--tags", "--prune").Run(); err != nil {
		failf("Failed to fetch tags: %s", err)
	}

	if err := command.New("git", "checkout", "master").Run(); err != nil {
		failf("Failed to checkout tag: %s", err)
	}
	//

	log.Infof("\nGenerating changelog...")

	cmd := command.New("releaseman", "--ci", "create-changelog", "--version", version, "--changelog-path", changelogPth)

	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		log.Errorf("command failed:")
		log.Printf(out)
		os.Exit(1)
	}

	changelog, err := fileutil.ReadStringFromFile(changelogPth)
	if err != nil {
		failf("Failed to read changelog: %s", err)
	}

	log.Infof("\nChangelog:")
	log.Printf(changelog)

	if err := tools.ExportEnvironmentWithEnvman("BITRSE_CHANGELOG", changelog); err != nil {
		failf("Failed to export changelog: %s", err)
	}

	log.Donef("\nThe changelog content is available in the BITRSE_CHANGELOG environment variable")
}
