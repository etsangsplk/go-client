// Package conf contains configuration structures used to setup the SDK
package conf

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os/user"
	"path"
	"strings"

	impressionlistener "github.com/splitio/go-client/splitio/impressionListener"
	"github.com/splitio/go-toolkit/datastructures/set"
	"github.com/splitio/go-toolkit/logging"
	"github.com/splitio/go-toolkit/nethelpers"
)

// SplitSdkConfig struct ...
// struct used to setup a Split.io SDK client.
//
// Parameters:
// - OperationMode (Required) Must be one of ["inmemory-standalone", "redis-consumer", "redis-standalone"]
// - InstanceName (Optional) Name to be used when submitting metrics & impressions to split servers
// - IPAddress (Optional) Address to be used when submitting metrics & impressions to split servers
// - BlockUntilReady (Optional) How much to wait until the sdk is ready
// - SplitFile (Optional) File with splits to use when running in localhost mode
// - LabelsEnabled (Optional) Can be used to disable labels if the user does not want to send that info to split servers.
// - Logger: (Optional) Custom logger complying with logging.LoggerInterface
// - LoggerConfig: (Optional) Options to setup the sdk's own logger
// - TaskPeriods: (Optional) How often should each task run
// - Redis: (Required for "redis-consumer" & "redis-standalone" operation modes. Sets up Redis config
// - Advanced: (Optional) Sets up various advanced options for the sdk
type SplitSdkConfig struct {
	OperationMode      string
	InstanceName       string
	IPAddress          string
	IPAddressesEnabled bool
	BlockUntilReady    int
	SplitFile          string
	LabelsEnabled      bool
	SplitSyncProxyURL  string
	Logger             logging.LoggerInterface
	LoggerConfig       logging.LoggerOptions
	TaskPeriods        TaskPeriods
	Advanced           AdvancedConfig
	Redis              RedisConfig
}

// TaskPeriods struct is used to configure the period for each synchronization task
type TaskPeriods struct {
	SplitSync      int
	SegmentSync    int
	ImpressionSync int
	GaugeSync      int
	CounterSync    int
	LatencySync    int
	EventsSync     int
}

// RedisConfig struct is used to cofigure the redis parameters
type RedisConfig struct {
	Host      string
	Port      int
	Database  int
	Password  string
	Prefix    string
	TLSConfig *tls.Config
}

// AdvancedConfig exposes more configurable parameters that can be used to further tailor the sdk to the user's needs
// - ImpressionListener - struct that will be notified each time an impression bulk is ready
// - HTTPTimeout - Timeout for HTTP requests when doing synchronization
// - SegmentQueueSize - How many segments can be queued for updating (should be >= # segments the user has)
// - SegmentWorkers - How many workers will be used when performing segments sync.
type AdvancedConfig struct {
	ImpressionListener   impressionlistener.ImpressionListener
	HTTPTimeout          int
	SegmentQueueSize     int
	SegmentWorkers       int
	SdkURL               string
	EventsURL            string
	EventsBulkSize       int64
	EventsQueueSize      int
	ImpressionsQueueSize int
	ImpressionsBulkSize  int64
}

// Default returns a config struct with all the default values
func Default() *SplitSdkConfig {

	instanceName := "unknown"
	ipAddress, err := nethelpers.ExternalIP()
	if err != nil {
		ipAddress = "unknown"
	} else {
		instanceName = fmt.Sprintf("ip-%s", strings.Replace(ipAddress, ".", "-", -1))
	}

	var splitFile string
	usr, err := user.Current()
	if err != nil {
		splitFile = "splits"
	} else {
		splitFile = path.Join(usr.HomeDir, ".splits")
	}

	return &SplitSdkConfig{
		OperationMode:      "inmemory-standalone",
		LabelsEnabled:      true,
		IPAddress:          ipAddress,
		IPAddressesEnabled: true,
		InstanceName:       instanceName,
		Logger:             nil,
		LoggerConfig:       logging.LoggerOptions{},
		SplitFile:          splitFile,
		Redis: RedisConfig{
			Database:  0,
			Host:      "localhost",
			Password:  "",
			Port:      6379,
			Prefix:    "",
			TLSConfig: nil,
		},
		TaskPeriods: TaskPeriods{
			CounterSync:    defaultTaskPeriod,
			GaugeSync:      defaultTaskPeriod,
			LatencySync:    defaultTaskPeriod,
			ImpressionSync: defaultTaskPeriod,
			SegmentSync:    defaultTaskPeriod,
			SplitSync:      defaultFeatureRefreshRate,
			EventsSync:     defaultTaskPeriod,
		},
		Advanced: AdvancedConfig{
			EventsURL:            "",
			SdkURL:               "",
			HTTPTimeout:          0,
			ImpressionListener:   nil,
			SegmentQueueSize:     500,
			SegmentWorkers:       10,
			EventsBulkSize:       5000,
			EventsQueueSize:      10000,
			ImpressionsQueueSize: 10000,
			ImpressionsBulkSize:  5000,
		},
	}
}

// Normalize checks that the parameters passed by the user are correct and updates parameters if necessary.
// returns an error if something is wrong
func Normalize(apikey string, cfg *SplitSdkConfig) error {
	// Fail if no apikey is provided
	if apikey == "" && cfg.OperationMode != "localhost" {
		return errors.New("Factory instantiation: you passed an empty apikey, apikey must be a non-empty string")
	}

	// To keep the interface consistent with other sdks we accept "localhost" as an apikey,
	// which sets the operation mode to localhost
	if apikey == "localhost" {
		cfg.OperationMode = "localhost"
	}

	// Fail if an invalid operation-mode is provided
	operationModes := set.NewSet(
		"localhost",
		"inmemory-standalone",
		"redis-consumer",
		"redis-standalone",
	)

	if !operationModes.Has(cfg.OperationMode) {
		return fmt.Errorf("OperationMode parameter must be one of: %v", operationModes.List())
	}

	if cfg.SplitSyncProxyURL != "" {
		cfg.Advanced.SdkURL = cfg.SplitSyncProxyURL
		cfg.Advanced.EventsURL = cfg.SplitSyncProxyURL
	}

	if !cfg.IPAddressesEnabled {
		cfg.IPAddress = "NA"
		cfg.InstanceName = "NA"
	}

	return nil
}
