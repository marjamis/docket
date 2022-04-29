package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/marjamis/docket/cli/internal/pkg/formatting"
	"github.com/marjamis/docket/lambda/reader/pkg/event"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "docket",
	Short: "Pull stored CloudWatch Events for ECS for quick reference ECS timelines/activities",
	Run: func(cmd *cobra.Command, args []string) {
		printOutput(scan())
	},
}

func scan() *dynamodb.ScanOutput {
	ddbClient := dynamodb.New(session.New(&aws.Config{
		Region: aws.String(viper.GetString("Resources.DynamoDB.Region")),
	}))

	scanOutput, err := ddbClient.Scan(generateQuery())
	if err != nil {
		log.Fatal(err)
	}

	return scanOutput
}

func generateQuery() *dynamodb.ScanInput {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(viper.GetString("Resources.DynamoDB.TableName")),
		FilterExpression: aws.String("begins_with(#n0, :v0)"),
		ExpressionAttributeNames: map[string]*string{
			"#n0": aws.String("time"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v0": {
				S: aws.String(viper.GetString("startFrom")),
			},
		},
	}

	log.Debugf("%+v", input)

	return input
}

// TODO fix the entire output to look nicer and fix the formatters
func printOutput(scanOutput *dynamodb.ScanOutput) {
	if *scanOutput.Count == int64(0) {
		log.Printf("No data in table. Exiting...")
		return
	}

	var events []event.EventData

	dynamodbattribute.UnmarshalListOfMaps(scanOutput.Items, &events)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', tabwriter.DiscardEmptyColumns)
	fmt.Fprint(w, "Number\tDate\tEvent Type\tARN\tDetail Type\tEvent Id\n")

	for i, v := range events {
		fmt.Fprintf(w, "%d\t%s\n", i, formatting.FormatEventData(&v))
	}
	w.Flush()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.docket.yaml)")

	var flag string
	rootCmd.Flags().StringVarP(&flag, "startFrom", "s", time.Now().UTC().Format("2006-01-02"), "Time from which to start the docker timeline")
	viper.BindPFlag("startFrom", rootCmd.Flags().Lookup("startFrom"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".docket" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".docket")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
