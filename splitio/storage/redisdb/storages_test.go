package redisdb

import (
	"fmt"
	"strings"
	"testing"

	"github.com/splitio/go-client/splitio"
	"github.com/splitio/go-client/splitio/conf"
	"github.com/splitio/go-client/splitio/service/dtos"
	"github.com/splitio/go-client/splitio/storage"
	"github.com/splitio/go-toolkit/datastructures/set"
	"github.com/splitio/go-toolkit/logging"
)

type LoggerInterface interface {
	Debug(msg ...interface{})
	Error(msg ...interface{})
	Info(msg ...interface{})
	Verbose(msg ...interface{})
	Warning(msg ...interface{})
	GetLog(key string) int
}

type MockedLogger struct {
	logs map[string]int
}

// NewMockedLogger creates a mocked logger to store logs
func NewMockedLogger() LoggerInterface {
	return &MockedLogger{
		logs: make(map[string]int),
	}
}

func (l *MockedLogger) Debug(msg ...interface{}) {
	messageList := make([]string, len(msg))
	for i, v := range msg {
		messageList[i] = fmt.Sprint(v)
	}
	var m string
	m = strings.Join(messageList, m)
	n, added := l.logs[m]
	if added == false {
		l.logs[m] = 1
	} else {
		l.logs[m] = n + 1
	}
}

func (l *MockedLogger) GetLog(key string) int {
	n, added := l.logs[key]
	if added == false {
		return -1
	}
	return n
}

func (l *MockedLogger) Error(msg ...interface{})   {}
func (l *MockedLogger) Info(msg ...interface{})    {}
func (l *MockedLogger) Verbose(msg ...interface{}) {}
func (l *MockedLogger) Warning(msg ...interface{}) {}
func TestRedisSplitStorage(t *testing.T) {
	logger := logging.NewLogger(&logging.LoggerOptions{})
	prefixedClient, err := NewPrefixedRedisClient(&conf.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Database: 1,
		Password: "",
		Prefix:   "testPrefix",
	})
	if err != nil {
		t.Error(err.Error())
		return
	}

	splitStorage := NewRedisSplitStorage(prefixedClient, logger)

	splitStorage.PutMany([]dtos.SplitDTO{
		{Name: "split1", ChangeNumber: 1},
		{Name: "split2", ChangeNumber: 2},
		{Name: "split3", ChangeNumber: 3},
		{Name: "split4", ChangeNumber: 4},
	}, 123)

	s1 := splitStorage.Get("split1")
	if s1 == nil || s1.Name != "split1" || s1.ChangeNumber != 1 {
		t.Error("Incorrect split fetched/stored")
	}

	sns := splitStorage.SplitNames()
	snsSet := set.NewSet(sns[0], sns[1], sns[2], sns[3])
	if !snsSet.IsEqual(set.NewSet("split1", "split2", "split3", "split4")) {
		t.Error("Incorrect split names fetched")
		t.Error(sns)
	}

	if splitStorage.Till() != 123 {
		t.Error("Incorrect till")
		t.Error(splitStorage.Till())
	}

	splitStorage.PutMany([]dtos.SplitDTO{
		{
			Name: "split5",
			Conditions: []dtos.ConditionDTO{
				{
					MatcherGroup: dtos.MatcherGroupDTO{
						Matchers: []dtos.MatcherDTO{
							{
								UserDefinedSegment: &dtos.UserDefinedSegmentMatcherDataDTO{
									SegmentName: "segment1",
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "split6",
			Conditions: []dtos.ConditionDTO{
				{
					MatcherGroup: dtos.MatcherGroupDTO{
						Matchers: []dtos.MatcherDTO{
							{
								UserDefinedSegment: &dtos.UserDefinedSegmentMatcherDataDTO{
									SegmentName: "segment2",
								},
							},
							{
								UserDefinedSegment: &dtos.UserDefinedSegmentMatcherDataDTO{
									SegmentName: "segment3",
								},
							},
						},
					},
				},
			},
		},
	}, 123)

	segmentNames := splitStorage.SegmentNames()
	hcSegments := set.NewSet("segment1", "segment2", "segment3")
	if !segmentNames.IsEqual(hcSegments) {
		t.Error("Incorrect segments retrieved")
		t.Error(segmentNames)
		t.Error(hcSegments)
	}

	allSplits := splitStorage.GetAll()
	allNames := set.NewSet()
	for _, split := range allSplits {
		allNames.Add(split.Name)
	}
	if !allNames.IsEqual(set.NewSet("split1", "split2", "split3", "split4", "split5", "split6")) {
		t.Error("GetAll returned incorrect splits")
	}

	// Test FetchMany before removing all the splits
	splitsToFetch := []string{"split1", "split2", "split3", "split4", "split5", "split6"}
	allSplitFetched := splitStorage.FetchMany(splitsToFetch)
	if len(allSplitFetched) != 6 {
		t.Error("It should return 6 splits")
	}
	for _, split := range splitsToFetch {
		if allSplitFetched[split] == nil {
			t.Error("It should not be nil")
		}
	}

	for _, name := range []string{"split1", "split2", "split3", "split4", "split5", "split6"} {
		splitStorage.Remove(name)
	}

	allSplits = splitStorage.GetAll()
	if len(allSplits) > 0 {
		t.Error("All splits should have been deleted")
		t.Error(allSplits)
	}

	// To test the .Clear() method we will add two splits and some random keys
	// we will check that the splits are deleted but the other keys remain intact
	splitStorage.PutMany([]dtos.SplitDTO{
		{
			Name: "split5",
			Conditions: []dtos.ConditionDTO{
				{
					MatcherGroup: dtos.MatcherGroupDTO{
						Matchers: []dtos.MatcherDTO{
							{
								UserDefinedSegment: &dtos.UserDefinedSegmentMatcherDataDTO{
									SegmentName: "segment1",
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "split6",
			Conditions: []dtos.ConditionDTO{
				{
					MatcherGroup: dtos.MatcherGroupDTO{
						Matchers: []dtos.MatcherDTO{
							{
								UserDefinedSegment: &dtos.UserDefinedSegmentMatcherDataDTO{
									SegmentName: "segment2",
								},
							},
							{
								UserDefinedSegment: &dtos.UserDefinedSegmentMatcherDataDTO{
									SegmentName: "segment3",
								},
							},
						},
					},
				},
			},
		},
	}, 123)
	splitStorage.client.client.Set("key1", "value1", 0)
	splitStorage.client.client.Set("key2", "value2", 0)
	splitStorage.Clear()

	allSplits = splitStorage.GetAll()
	if len(allSplits) > 0 {
		t.Error("All splits should have been deleted")
		t.Error(allSplits)
	}

	key1, _ := splitStorage.client.client.Get("key1").Result()
	key2, _ := splitStorage.client.client.Get("key2").Result()

	if key1 != "value1" || key2 != "value2" {
		t.Error("Keys that are not splits should not have been altered")
	}

	allSplitFetched = splitStorage.FetchMany(splitsToFetch)
	if len(allSplitFetched) != 6 {
		t.Error("It should return 6 splits")
	}
	for _, split := range splitsToFetch {
		if allSplitFetched[split] != nil {
			t.Error("It should be nil")
		}
	}

	splitStorage.client.client.Del("key1", "key2")
}

func TestSegmentStorage(t *testing.T) {
	logger := logging.NewLogger(&logging.LoggerOptions{})
	prefixedClient, err := NewPrefixedRedisClient(&conf.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Database: 1,
		Password: "",
		Prefix:   "testPrefix",
	})
	if err != nil {
		t.Error(err.Error())
		return
	}

	segmentStorage := NewRedisSegmentStorage(prefixedClient, logger)

	segmentStorage.Put("segment1", set.NewSet("item1", "item2", "item3"), 123)
	segmentStorage.Put("segment2", set.NewSet("item4", "item5", "item6"), 124)

	segment1 := segmentStorage.Get("segment1")
	if segment1 == nil || !segment1.IsEqual(set.NewSet("item1", "item2", "item3")) {
		t.Error("Incorrect segment1")
		t.Error(segment1)
	}

	for _, key := range []string{"item1", "item2", "item3"} {
		contained, _ := segmentStorage.SegmentContainsKey("segment1", key)
		if !contained {
			t.Errorf("SegmentContainsKey should return true for segment '%s' and key '%s'", "segment1", key)
		}
	}

	segment2 := segmentStorage.Get("segment2")
	if segment2 == nil || !segment2.IsEqual(set.NewSet("item4", "item5", "item6")) {
		t.Error("Incorrect segment2")
	}

	for _, key := range []string{"item4", "item5", "item6"} {
		contained, _ := segmentStorage.SegmentContainsKey("segment2", key)
		if !contained {
			t.Errorf("SegmentContainsKey should return true for segment '%s' and key '%s'", "segment2", key)
		}
	}

	if segmentStorage.Till("segment1") != 123 || segmentStorage.Till("segment2") != 124 {
		t.Error("Incorrect till stored")
	}

	segmentStorage.Put("segment1", set.NewSet("item7"), 222)
	segment1 = segmentStorage.Get("segment1")
	if !segment1.IsEqual(set.NewSet("item7")) {
		t.Error("Segment 1 not overwritten correctly")
	}

	if segmentStorage.Till("segment1") != 222 {
		t.Error("segment 1 till not updated correctly")
	}

	segmentStorage.Remove("segment1")
	if segmentStorage.Get("segment1") != nil || segmentStorage.Till("segment1") != -1 {
		t.Error("Segment 1 and it's till value should have been removed")
		t.Error(segmentStorage.Get("segment1"))
		t.Error(segmentStorage.Till("segment1"))
	}

	// To test the .Clear() method we add a couple of segments and random keys
	// we check that the segments are the deleted but the other keys remain intact
	segmentStorage.Put("segment1", set.NewSet("item1", "item2"), 222)
	segmentStorage.Put("segment2", set.NewSet("item3", "item4"), 222)
	segmentStorage.client.client.Set("key1", "value1", 0)
	segmentStorage.client.client.Set("key2", "value2", 0)
	segmentStorage.Clear()
	if segmentStorage.Get("segment1") != nil || segmentStorage.Get("segment2") != nil {
		t.Error("All segments should have been cleared")
	}

	key1, _ := segmentStorage.client.client.Get("key1").Result()
	key2, _ := segmentStorage.client.client.Get("key2").Result()

	if key1 != "value1" || key2 != "value2" {
		t.Error("random keys should have not been altered")
	}

	segmentStorage.client.client.Del("key1", "key2")
}

func TestImpressionStorage(t *testing.T) {
	logger := NewMockedLogger()
	prefixedClient, err := NewPrefixedRedisClient(&conf.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Database: 1,
		Password: "",
		Prefix:   "testPrefix",
	})
	if err != nil {
		t.Error(err.Error())
		return
	}
	metadata := &splitio.SdkMetadata{
		SDKVersion:  "go-test",
		MachineName: "instance123",
	}
	impressionStorage := NewRedisImpressionStorage(prefixedClient, metadata, logger)

	var impression1 = storage.Impression{
		FeatureName:  "feature1",
		BucketingKey: "abc",
		ChangeNumber: 123,
		KeyName:      "key1",
		Label:        "label1",
		Time:         111,
		Treatment:    "on",
	}
	impressionStorage.LogImpressions([]storage.Impression{impression1})

	impressionStorage.client.client.Del(impressionStorage.redisKey)

	var ttl = impressionStorage.client.client.TTL(impressionStorage.redisKey).Val()

	if ttl > impressionStorage.impressionsTTL {
		t.Error("TTL should be less than or equal to default")
	}

	var impression2 = storage.Impression{
		FeatureName:  "feature2",
		BucketingKey: "abc",
		ChangeNumber: 123,
		KeyName:      "key1",
		Label:        "label1",
		Time:         111,
		Treatment:    "off",
	}
	impressionStorage.LogImpressions([]storage.Impression{impression2})

	impressions, _ := impressionStorage.PopN(2)

	if len(impressions) != 2 {
		t.Error("Incorrect number of impressions fetched")
	}

	expirationAdded := logger.GetLog("Proceeding to set expiration for: " + impressionStorage.redisKey)

	if expirationAdded != 1 {
		t.Error("It should added expiration only once")
	}

	var i1 = impressions[0]
	if i1.FeatureName != impression1.FeatureName {
		t.Error("Wrong Impression Stored, actual:", i1.FeatureName, " expected: ", impression1.FeatureName)
	}

	var i2 = impressions[1]
	if i2.FeatureName != impression2.FeatureName {
		t.Error("Wrong Impression Stored, actual:", i2.FeatureName, " expected: ", impression2.FeatureName)
	}
}

func TestMetricsStorage(t *testing.T) {
	logger := logging.NewLogger(&logging.LoggerOptions{})
	prefixedClient, err := NewPrefixedRedisClient(&conf.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Database: 1,
		Password: "",
		Prefix:   "testPrefix",
	})
	if err != nil {
		t.Error(err.Error())
		return
	}
	metadata := &splitio.SdkMetadata{
		SDKVersion:  "go-test",
		MachineName: "instance123",
	}
	metricsStorage := NewRedisMetricsStorage(prefixedClient, metadata, logger)

	// Gauges

	metricsStorage.PutGauge("g1", 3.345)
	metricsStorage.PutGauge("g2", 4.456)
	if metricsStorage.client.client.Exists(
		"testPrefix.SPLITIO/go-test/instance123/gauge.g1",
		"testPrefix.SPLITIO/go-test/instance123/gauge.g2",
	).Val() != 2 {
		t.Error("Keys or stored in an incorrect format")
	}
	gauges := metricsStorage.PopGauges()

	if len(gauges) != 2 {
		t.Error("Incorrect number of gauges fetched")
		t.Error(gauges)
	}

	var g1, g2 dtos.GaugeDTO
	if gauges[0].MetricName == "g1" {
		g1 = gauges[0]
		g2 = gauges[1]
	} else if gauges[0].MetricName == "g2" {
		g1 = gauges[1]
		g2 = gauges[0]
	} else {
		t.Error("Incorrect gauges names")
		return
	}

	if g1.Gauge != 3.345 || g2.Gauge != 4.456 {
		t.Error("Incorrect gauge values retrieved")
	}

	if metricsStorage.client.client.Exists(
		"testPrefix.SPLITIO/go-test/instance123/gauge.g1",
		"testPrefix.SPLITIO/go-test/instance123/gauge.g2",
	).Val() != 0 {
		t.Error("Gauge keys should have been removed after PopAll() function call")
	}

	// Latencies
	metricsStorage.IncLatency("m1", 13)
	metricsStorage.IncLatency("m1", 13)
	metricsStorage.IncLatency("m1", 13)
	metricsStorage.IncLatency("m1", 1)
	metricsStorage.IncLatency("m1", 1)
	metricsStorage.IncLatency("m2", 1)
	metricsStorage.IncLatency("m2", 2)

	if metricsStorage.client.client.Exists(
		"testPrefix.SPLITIO/go-test/instance123/latency.m1.bucket.13",
		"testPrefix.SPLITIO/go-test/instance123/latency.m1.bucket.1",
		"testPrefix.SPLITIO/go-test/instance123/latency.m2.bucket.1",
		"testPrefix.SPLITIO/go-test/instance123/latency.m2.bucket.2",
	).Val() != 4 {
		t.Error("Keys or stored in an incorrect format")
	}

	latencies := metricsStorage.PopLatencies()
	var m1, m2 dtos.LatenciesDTO
	if latencies[0].MetricName == "m1" {
		m1 = latencies[0]
		m2 = latencies[1]
	} else if latencies[0].MetricName == "m2" {
		m1 = latencies[1]
		m2 = latencies[0]
	} else {
		t.Error("Incorrect latency names")
		return
	}

	if m1.Latencies[13] != 3 || m1.Latencies[1] != 2 {
		t.Error("Incorrect latencies for m1")
	}

	if m2.Latencies[1] != 1 || m2.Latencies[2] != 1 {
		t.Error("Incorrect latencies for m2")
	}

	if metricsStorage.client.client.Exists(
		"testPrefix.SPLITIO/go-test/instance123/latency.m1.bucket.13",
		"testPrefix.SPLITIO/go-test/instance123/latency.m1.bucket.1",
		"testPrefix.SPLITIO/go-test/instance123/latency.m2.bucket.1",
		"testPrefix.SPLITIO/go-test/instance123/latency.m2.bucket.2",
	).Val() != 0 {
		t.Error("Latency keys should have been deleted after PopAll()")
	}

	// Counters
	metricsStorage.IncCounter("count1")
	metricsStorage.IncCounter("count1")
	metricsStorage.IncCounter("count1")
	metricsStorage.IncCounter("count2")
	metricsStorage.IncCounter("count2")
	metricsStorage.IncCounter("count2")
	metricsStorage.IncCounter("count2")
	metricsStorage.IncCounter("count2")
	metricsStorage.IncCounter("count2")

	if metricsStorage.client.client.Exists(
		"testPrefix.SPLITIO/go-test/instance123/count.count1",
		"testPrefix.SPLITIO/go-test/instance123/count.count2",
	).Val() != 2 {
		t.Error("Incorrect counter keys stored in redis")
	}

	counters := metricsStorage.PopCounters()

	var c1, c2 dtos.CounterDTO
	if counters[0].MetricName == "count1" {
		c1 = counters[0]
		c2 = counters[1]
	} else if counters[0].MetricName == "count2" {
		c1 = counters[1]
		c2 = counters[0]
	} else {
		t.Error("Incorrect counters fetched")
	}

	if c1.Count != 3 {
		t.Error("Incorrect count for count1")
	}

	if c2.Count != 6 {
		t.Error("Incorrect count for count2")
	}

	if metricsStorage.client.client.Exists(
		"testPrefix.SPLITIO/go-test/instance123/count.count1",
		"testPrefix.SPLITIO/go-test/instance123/count.count2",
	).Val() != 0 {
		t.Error("Counter keys should have been removed after PopAll()")
	}
}

func TestTrafficTypeStorage(t *testing.T) {
	logger := NewMockedLogger()
	prefixedClient, err := NewPrefixedRedisClient(&conf.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Database: 1,
		Password: "",
		Prefix:   "testPrefix",
	})
	if err != nil {
		t.Error(err.Error())
		return
	}
	ttStorage := NewRedisSplitStorage(prefixedClient, logger)

	ttStorage.client.client.Del("testPrefix.SPLITIO.trafficType.mytraffictype")
	ttStorage.client.client.Incr("testPrefix.SPLITIO.trafficType.mytraffictype")

	if !ttStorage.TrafficTypeExists("mytraffictype") {
		t.Error("Traffic type should exists")
	}

	ttStorage.client.client.Del("testPrefix.SPLITIO.trafficType.mytraffictype")
}
