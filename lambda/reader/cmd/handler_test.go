package cmd

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	fuzz "github.com/google/gofuzz"

	"github.com/stretchr/testify/assert"
)

func TestFormatDDBEntryFuzz(t *testing.T) {
	t.Run("Checking Common Attributes", func(t *testing.T) {
		for i := 0; i < 10000; i++ {
			object := events.CloudWatchEvent{}
			f := fuzz.New()
			f.Fuzz(&object)
			object.Detail = []byte{
				123,
				125,
			}

			data, err := formatDDBEntry(object)
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println(data)
			assert.Equal(t, object.ID, *data["id"].S)
			assert.Equal(t, object.DetailType, *data["detail-type"].S)
			assert.Equal(t, object.Source, *data["source"].S)
			assert.Equal(t, object.AccountID, *data["account"].S)
			// RFC3339Nano is the time format used by DynamoDB Attribute
			assert.Equal(t, object.Time.Format(time.RFC3339Nano), *data["time"].S)
			assert.Equal(t, object.Region, *data["region"].S)

			var r []*dynamodb.AttributeValue
			for i := range object.Resources {
				r = append(r, &dynamodb.AttributeValue{S: &object.Resources[i]})
			}
			assert.Equal(t, r, data["resources"].L)

			epoch, err := strconv.Atoi(*data["epoch-ttl"].N)
			if err != nil {
				fmt.Println(err)
			}
			// TODO improve this check to be more "accurate"
			assert.Greater(t, int64(epoch), time.Now().Unix())
			//EventJSON payload / type specific
		}
	})
}
