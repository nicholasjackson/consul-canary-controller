package canary

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupPlugin(t *testing.T, config string) (*Plugin, *mocks.MonitorMock) {
	log := hclog.NewNullLogger()
	_, m := mocks.BuildMocks(t)

	p, _ := New(log, m.MonitorMock)

	// set the traffic to initial traffic, this is the state after the first run
	p.currentTraffic = 10

	err := p.Configure([]byte(config))
	require.NoError(t, err)

	return p, m.MonitorMock
}

func TestSetsIntitialDelayToIntervalWhenNotSet(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategyWithoutInitialDelay)
	require.Equal(t, p.config.InitialDelay, p.config.Interval)
}

func TestValidatesConfig(t *testing.T) {
	log := hclog.NewNullLogger()
	_, m := mocks.BuildMocks(t)

	p, _ := New(log, m.MonitorMock)

	// set the traffic to initial traffic, this is the state after the first run
	p.currentTraffic = 10

	err := p.Configure([]byte(canaryStrategyWithValidationErrors))
	require.Error(t, err)

	require.Contains(t, err.Error(), ErrInvalidInitialDelay.Error())
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
	require.Contains(t, err.Error(), ErrTrafficStep.Error())
	require.Contains(t, err.Error(), ErrMaxTraffic.Error())
	require.Contains(t, err.Error(), ErrThreshold.Error())
}

func TestSetsIntitialTraficAndReturnsFirstRun(t *testing.T) {
	p, mm := setupPlugin(t, canaryStrategy)
	// reset the traffic to the initial state
	p.currentTraffic = -1

	status, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(status))
	require.Equal(t, 10, traffic)

	mm.AssertNotCalled(t, "Check", mock.Anything)
}

func TestSetsIntitialTraficToTrafficStepWhenNotSetAndReturnsFirstRun(t *testing.T) {
	p, mm := setupPlugin(t, canaryStrategyWithoutInitial)
	// reset the traffic to the initial state
	p.currentTraffic = -1

	status, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(status))
	require.Equal(t, 20, traffic)

	mm.AssertNotCalled(t, "Check", mock.Anything)
}

func TestCallsMonitorCheckAndReturnsWhenNoError(t *testing.T) {
	st := time.Now()
	p, mm := setupPlugin(t, canaryStrategy)

	_, _, err := p.Execute(context.Background())
	require.NoError(t, err)

	mm.AssertCalled(t, "Check", mock.Anything, 30*time.Millisecond)

	et := time.Since(st)
	require.Greater(t, et, 30*time.Millisecond, "Execute should sleep for interval before check")
}

func TestExecuteReturnsIncrementsTrafficSubsequentRuns(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.currentTraffic = 10

	state, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(state))
	require.Equal(t, 20, traffic)
}

func TestExecuteReturnsCompleteWhenAllChecksComplete(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.currentTraffic = 80

	state, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusComplete, string(state))
	require.Equal(t, 100, traffic)
}

func TestReturnsErrorWhenChecksFail(t *testing.T) {
	p, mm := setupPlugin(t, canaryStrategy)
	testutils.ClearMockCall(&mm.Mock, "Check")

	mm.On("Check", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	state, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusFail, string(state))
	require.Equal(t, traffic, 0)

	// should call check 5 times due to error threshold
	mm.AssertNumberOfCalls(t, "Check", 5)
}

func TestGetPrimaryTrafficReturnsCorrectValue(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.currentTraffic = 80

	traf := p.GetPrimaryTraffic()

	require.Equal(t, 20, traf)
}

func TestGetCandidateTrafficReturnsCorrectValue(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.currentTraffic = 80

	traf := p.GetCandidateTraffic()

	require.Equal(t, 80, traf)
}

const canaryStrategyWithoutInitialDelay = `
{
  "interval": "30ms",
  "initial_traffic": 10,
  "traffic_step": 10,
  "max_traffic": 90,
  "error_threshold": 5
}
`

const canaryStrategy = `
{
  "interval": "30ms",
  "initial_traffic": 10,
  "initial_delay": "30ms",
  "traffic_step": 10,
  "max_traffic": 90,
  "error_threshold": 5
}
`

const canaryStrategyWithoutInitial = `
{
  "interval": "30ms",
  "initial_delay": "30ms",
  "traffic_step": 20,
  "max_traffic": 90,
  "error_threshold": 5
}
`

const canaryStrategyWithValidationErrors = `
{
  "initial_delay": "acs",
  "interval": "30",
  "initial_traffic": 101,
  "traffic_step": 1100,
  "max_traffic": -3,
  "error_threshold": -1
}
`
