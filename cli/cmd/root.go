package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

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
		setLogLevel(viper.GetString("log-level"))

		printScanOutput(scan())
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
				S: aws.String(viper.GetString("start-from")),
			},
		},
	}

	log.Debugf("%+v", input)

	return input
}

func printScanOutput(scanOutput *dynamodb.ScanOutput) {
	if *scanOutput.Count == int64(0) {
		log.Printf("No data in table. Exiting...")
		return
	}

	var events []event.EventData

	dynamodbattribute.UnmarshalListOfMaps(scanOutput.Items, &events)

	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 10, ' ', tabwriter.DiscardEmptyColumns)
	defer writer.Flush()

	for _, event := range events {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", event.Time, event.ID, event.DetailType, event.Resources, event.EventJSON)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func setLogLevel(level string) {
	switch viper.GetString("log-level") {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.docket.yaml)")

	var logLevel string
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "log level to use through the applications execution (currently allowed values are (info and debug)")
	viper.BindPFlag("log-level", rootCmd.Flags().Lookup("log-level"))

	var startFrom string
	rootCmd.Flags().StringVarP(&startFrom, "start-from", "s", time.Now().UTC().Format("2006-01-02"), "Time from which to start the docker timeline")
	viper.BindPFlag("start-from", rootCmd.Flags().Lookup("start-from"))
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
