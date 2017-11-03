package client

import (
	"errors"
	"github.com/splitio/go-client/splitio/engine/evaluator"
	"github.com/splitio/go-client/splitio/service/dtos"
	"github.com/splitio/go-client/splitio/storage"
	"github.com/splitio/go-client/splitio/util/metrics"
	"github.com/splitio/go-toolkit/asynctask"
	"github.com/splitio/go-toolkit/logging"
	"time"
)

// SplitClient is the entry-point of the split SDK.
type SplitClient struct {
	apikey       string
	logger       logging.LoggerInterface
	loggerConfig logging.LoggerOptions
	evaluator    *evaluator.Evaluator
	sync         *sdkSync
	impressions  storage.ImpressionStorage
	metrics      storage.MetricsStorage
}

type sdkSync struct {
	splitSync      *asynctask.AsyncTask
	segmentSync    *asynctask.AsyncTask
	impressionSync *asynctask.AsyncTask
	gaugeSync      *asynctask.AsyncTask
	countersSync   *asynctask.AsyncTask
	latenciesSync  *asynctask.AsyncTask
}

func parseKeys(key interface{}) (string, *string, error) {
	matchingKey, ok := key.(string)
	bucketingKey := matchingKey
	if !ok {
		ckey, ok := key.(Key)
		if !ok {
			return "", nil, errors.New("Supplied key is neither a string or a Key struct")
		}
		matchingKey = ckey.MatchingKey
		bucketingKey = ckey.BucketingKey
	}

	return matchingKey, &bucketingKey, nil
}

// Treatment implements the main functionality of split. Retrieve treatments of a specific feature
// for a certain key and set of attributes
func (c *SplitClient) Treatment(key interface{}, feature string, attributes map[string]interface{}) string {
	// Set up a guard deferred function to recover if the SDK starts panicking
	defer func() string {
		if r := recover(); r != nil {
			// At this point we'll only trust that the logger isn't panicking trust that the logger isn't panicking
			c.logger.Error("SDK is panicking with the following error")
			c.logger.Error(r)
			c.logger.Error("Returning CONTROL")
		}
		return "CONTROL"
	}()

	matchingKey, bucketingKey, err := parseKeys(key)
	if err != nil {
		c.logger.Error("Error parsing key")
		c.logger.Error(err.Error())
		return "CONTROL"
	}

	evaluationResult := c.evaluator.Evaluate(matchingKey, bucketingKey, feature, attributes)

	// Store impression
	if c.impressions != nil {
		c.impressions.Put(feature, &dtos.ImpressionDTO{
			BucketingKey: *bucketingKey,
			ChangeNumber: evaluationResult.SplitChangeNumber,
			KeyName:      matchingKey,
			Label:        evaluationResult.Label,
			Treatment:    evaluationResult.Treatment,
			Time:         time.Now().Unix() * 1000, // Convert Unix timestamp to java's ms timestamps
		})
	} else {
		c.logger.Warning("No impression storage set in client. Not sending impressions!")
	}

	// Store latency
	bucket := metrics.Bucket(evaluationResult.EvaluationTimeNs)
	c.metrics.IncLatency("sdk.getTreatment", bucket)

	return evaluationResult.Treatment
}
