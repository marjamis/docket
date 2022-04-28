package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

var (
	defaultConfigurationFileLocation = fmt.Sprintf("%s/.docket.yml", os.Getenv("HOME"))
)

// DynamoDBConfiguration holds data about the DynamoDB table resource used as the cli's source
type DynamoDBConfiguration struct {
	Region    string `yaml:"Region"`
	TableName string `yaml:"TableName"`
}

// Resources holds the data about the resources used by the cli to render its information
type Resources struct {
	DynamoDB DynamoDBConfiguration `yaml:"DynamoDB"`
}

// ConfigurationFile holds all the configuration information that docker may need
type ConfigurationFile struct {
	Resources Resources `yaml:"Resources"`
}

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Prompts for inputs to generate a configuration file to be used for docket",
	Run: func(cmd *cobra.Command, args []string) {
		configurationFileLocation := defaultConfigurationFileLocation
		configurationFile := ConfigurationFile{
			Resources: Resources{
				DynamoDB: DynamoDBConfiguration{
					Region:    "ap-southeast-2",
					TableName: "docket-EventsDDBTable-1TP5IEPWXBI0L",
				}},
		}

		configurationFile.Resources.DynamoDB.Region = readStringAndReturnValue("Region with DynamoDB table", configurationFile.Resources.DynamoDB.Region)
		configurationFile.Resources.DynamoDB.TableName = readStringAndReturnValue("DynamoDB table name", configurationFile.Resources.DynamoDB.TableName)
		configurationFileLocation = readStringAndReturnValue("Save configuration file at", configurationFileLocation)

		err := writeConfigurationFileToDisk(configurationFile, configurationFileLocation)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	},
}

func readStringAndReturnValue(prompt string, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)

	reader := bufio.NewReader(os.Stdin)
	// ReadString will block until the delimiter is entered
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	input = strings.TrimSuffix(input, "\n")

	if input != "" {
		return input
	}

	return defaultValue
}

func writeConfigurationFileToDisk(configurationFile ConfigurationFile, configurationFileLocation string) error {
	configurationFileYaml, err := yaml.Marshal(&configurationFile)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configurationFileLocation, configurationFileYaml, 0644)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
