package utils

import (
	"bufio"
	"bytes"
	"strings"
)

type CloudConfig struct {
	WriteFiles []WriteFile
	RunCmd     []string
}

type WriteFile struct {
	Path, Owner, Permissions string
	Contents                 []string
}

func NewCloudConfig() *CloudConfig {
	cloudConfig := &CloudConfig{}
	cloudConfig.WriteFiles = []WriteFile{}
	cloudConfig.RunCmd = []string{}

	return cloudConfig
}

const (
	StartCondiWriteFile = "-   path: "
	PrefixPath          = "-   path: "
	PrefixOwner         = "    owner: "
	PrefixPermission    = "    permissions: "
	EndCondiWriteFile   = "-----END"
	StartCondiRunCmd    = "runcmd"
	PrefixRunCmd        = "  - "
)

func (cloudConfig *CloudConfig) FromBytes(input []byte) error {
	reader := bufio.NewReader(bytes.NewReader(input))

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.HasPrefix(line, StartCondiWriteFile) {
			writeFile := WriteFile{}
			path := strings.Split(line, PrefixPath)[1]

			scanner.Scan()
			line = scanner.Text()
			owner := strings.Split(line, PrefixOwner)[1]

			scanner.Scan()
			line = scanner.Text()
			permission := strings.Split(strings.Split(line, PrefixPermission)[1], "'")[1]

			scanner.Scan()

			var contents []string

			for {
				scanner.Scan()
				line = scanner.Text()
				contents = append(contents, line[6:])

				if strings.Contains(line, EndCondiWriteFile) {
					break
				}

				if strings.HasPrefix(line, StartCondiRunCmd) {
					contents = contents[:len(contents)-1]

					for {
						scanner.Scan()
						line = scanner.Text()
						if strings.Contains(line, PrefixRunCmd) {
							splitIndex := 1
							if len(strings.Split(strings.Split(line, PrefixRunCmd)[1], "'")) == 1 {
								splitIndex = 0
							}
							cloudConfig.RunCmd = append(cloudConfig.RunCmd, strings.Split(strings.Split(line, PrefixRunCmd)[1], "'")[splitIndex])
						}
						break
					}
					break
				}
			}

			writeFile.Path = path
			writeFile.Owner = owner
			writeFile.Permissions = permission
			writeFile.Contents = contents

			cloudConfig.WriteFiles = append(cloudConfig.WriteFiles, writeFile)
		}

	}

	return nil
}
