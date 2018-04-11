package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"

	"github.com/bitrise-io/go-utils/fileutil"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

const defaultReleaseConfig = `release:
development_branch: master
release_branch: master
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
	log.Printf("release_config: %s", releaseConfigContent)

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

	cmd := command.NewWithStandardOuts("releaseman", "--ci", "create-changelog", "--version", version, "--changelog-path", changelogPth)

	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		failf("Failed to generate changelog: %s", err)
	}
}