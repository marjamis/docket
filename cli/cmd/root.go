package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/marjamis/docket/cli/internal/pkg/formatting"
	"github.com/marjamis/docket/lambda/reader/pkg/event"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	startFrom string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "docket",
	Short: "Pull stored CloudWatch Events for ECS for quick reference ECS timelines/activities",
	Run: func(cmd *cobra.Command, args []string) {
		settingsPrint()

		dynamodb_svc := dynamodb.New(session.New(&aws.Config{
			Region: aws.String(viper.GetString("Resources.DynamoDB.Region")),
		}))
		// xray.AWS(dynamodb_svc.Client)

		scanOutput, err := dynamodb_svc.Scan(&dynamodb.ScanInput{
			TableName:        aws.String(viper.GetString("Resources.DynamoDB.TableName")),
			FilterExpression: aws.String("begins_with(#n0, :v0)"),
			ExpressionAttributeNames: map[string]*string{
				"#n0": aws.String("time"),
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":v0": {
					S: aws.String("2022-01-06"),
				},
			},
		})
		if err != nil {
			fmt.Println(err)
		}

		// fmt.Printf("%+v", scanOutput)

		var events []event.EventData
		dynamodbattribute.UnmarshalListOfMaps(scanOutput.Items, &events)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', tabwriter.DiscardEmptyColumns)
		for i, v := range events {
			fmt.Fprintf(w, "%d\t%s\n", i, formatting.FormatEventData(&v))
		}
		w.Flush()
	},
}

func settingsPrint() {
	// TODO make prettier
	fmt.Printf("Table settings:\n* TableName: %s\n* Region: %s\n", viper.Get("Resources.DynamoDB.TableName"), viper.Get("Resources.DynamoDB.Region"))
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
	rootCmd.Flags().StringVarP(&startFrom, "startFrom", "s", "sads", "Time from which to start the docker timeline")
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
