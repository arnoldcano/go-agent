package newrelic

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/cat"
)

func TestShouldSaveTrace(t *testing.T) {
	for _, tc := range []struct {
		name          string
		expected      bool
		synthetics    bool
		tracerEnabled bool
		collectTraces bool
		duration      time.Duration
		threshold     time.Duration
	}{
		{
			name:          "insufficient duration, all disabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: false,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, only synthetics enabled",
			expected:      false,
			synthetics:    true,
			tracerEnabled: false,
			collectTraces: false,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, only tracer enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: false,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, only collect traces enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: true,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, all normal flags enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: true,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, all flags enabled",
			expected:      true,
			synthetics:    true,
			tracerEnabled: true,
			collectTraces: true,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, all disabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: false,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, only synthetics enabled",
			expected:      false,
			synthetics:    true,
			tracerEnabled: false,
			collectTraces: false,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, only tracer enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: false,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, only collect traces enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: true,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, all normal flags enabled",
			expected:      true,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: true,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, all flags enabled",
			expected:      true,
			synthetics:    true,
			tracerEnabled: true,
			collectTraces: true,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
	} {
		txn := &txn{}

		cfg := defaultConfig()
		cfg.TransactionTracer.Enabled = tc.tracerEnabled
		cfg.TransactionTracer.Threshold.Duration = tc.threshold
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		reply := internal.ConnectReplyDefaults()
		reply.CollectTraces = tc.collectTraces
		txn.appRun = newAppRun(config{Config: cfg}, reply)

		txn.Duration = tc.duration
		if tc.synthetics {
			txn.CrossProcess.Synthetics = &cat.SyntheticsHeader{}
			txn.CrossProcess.SetSynthetics(tc.synthetics)
		}

		if actual := txn.shouldSaveTrace(); actual != tc.expected {
			t.Errorf("%s: unexpected shouldSaveTrace value; expected %v; got %v", tc.name, tc.expected, actual)
		}
	}
}

func TestLazilyCalculateSampledTrue(t *testing.T) {
	tx := &txn{}
	tx.appRun = &appRun{}
	tx.BetterCAT.Priority = 0.5
	tx.sampledCalculated = false
	tx.BetterCAT.Enabled = true
	tx.Reply = &internal.ConnectReply{}
	tx.Reply.SetSampleEverything()
	out := tx.lazilyCalculateSampled()
	if !out || !tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 1.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
	tx.Reply.SetSampleNothing()
	out = tx.lazilyCalculateSampled()
	if !out || !tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 1.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
}

func TestLazilyCalculateSampledFalse(t *testing.T) {
	tx := &txn{}
	tx.appRun = &appRun{}
	tx.BetterCAT.Priority = 0.5
	tx.sampledCalculated = false
	tx.BetterCAT.Enabled = true
	tx.Reply = &internal.ConnectReply{}
	tx.Reply.SetSampleNothing()
	out := tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
	tx.Reply.SetSampleEverything()
	out = tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
}

func TestLazilyCalculateSampledCATDisabled(t *testing.T) {
	tx := &txn{}
	tx.appRun = &appRun{}
	tx.BetterCAT.Priority = 0.5
	tx.sampledCalculated = false
	tx.BetterCAT.Enabled = false
	tx.Reply = &internal.ConnectReply{}
	tx.Reply.SetSampleEverything()
	out := tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
	out = tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
}

type expectTxnTimes struct {
	txn       *txn
	testName  string
	start     time.Time
	stop      time.Time
	duration  time.Duration
	totalTime time.Duration
}

func TestTransactionDurationTotalTime(t *testing.T) {
	// These tests touch internal txn structures rather than the public API:
	// Testing duration and total time is tough because our API functions do
	// not take fixed times.
	start := time.Now()
	testTxnTimes := func(expect expectTxnTimes) {
		if expect.txn.Start != expect.start {
			t.Error("start time", expect.testName, expect.txn.Start, expect.start)
		}
		if expect.txn.Stop != expect.stop {
			t.Error("stop time", expect.testName, expect.txn.Stop, expect.stop)
		}
		if expect.txn.Duration != expect.duration {
			t.Error("duration", expect.testName, expect.txn.Duration, expect.duration)
		}
		if expect.txn.TotalTime != expect.totalTime {
			t.Error("total time", expect.testName, expect.txn.TotalTime, expect.totalTime)
		}
	}

	// Basic transaction with no async activity.
	tx := &txn{}
	tx.markStart(start)
	segmentStart := internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(2*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), &tx.mainThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "basic transaction",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 3 * time.Second,
	})

	// Transaction with async activity.
	tx = &txn{}
	tx.markStart(start)
	segmentStart = internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(2*time.Second), "name")
	asyncThread := createThread(tx)
	asyncSegmentStart := internal.StartSegment(&tx.TxnData, asyncThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, asyncThread, asyncSegmentStart, start.Add(2*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), &tx.mainThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "transaction with async activity",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 4 * time.Second,
	})

	// Transaction ended on async thread.
	tx = &txn{}
	tx.markStart(start)
	segmentStart = internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(2*time.Second), "name")
	asyncThread = createThread(tx)
	asyncSegmentStart = internal.StartSegment(&tx.TxnData, asyncThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, asyncThread, asyncSegmentStart, start.Add(2*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), asyncThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "transaction ended on async thread",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 4 * time.Second,
	})

	// Duration exceeds TotalTime.
	tx = &txn{}
	tx.markStart(start)
	segmentStart = internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(0*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(1*time.Second), "name")
	asyncThread = createThread(tx)
	asyncSegmentStart = internal.StartSegment(&tx.TxnData, asyncThread, start.Add(2*time.Second))
	internal.EndBasicSegment(&tx.TxnData, asyncThread, asyncSegmentStart, start.Add(3*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), asyncThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "TotalTime should be at least Duration",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 3 * time.Second,
	})
}

func TestGetTraceMetadataDistributedTracingDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "" {
		t.Error(metadata.TraceID)
	}
}

func TestGetTraceMetadataSuccess(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "e71870997d57214c" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "1ae969564b34a33ecd1af05fe6923d6d" {
		t.Error(metadata.TraceID)
	}
	txn.StartSegment("name")
	// Span id should be different now that a segment has started.
	metadata = txn.GetTraceMetadata()
	if metadata.SpanID != "4259d74b863e2fba" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "1ae969564b34a33ecd1af05fe6923d6d" {
		t.Error(metadata.TraceID)
	}
}

func TestGetTraceMetadataEnded(t *testing.T) {
	// Test that GetTraceMetadata returns empty strings if the transaction
	// has been finished.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	txn.End()
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "" {
		t.Error(metadata.TraceID)
	}
}

func TestGetTraceMetadataNotSampled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleNothing()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "1ae969564b34a33ecd1af05fe6923d6d" {
		t.Error(metadata.TraceID)
	}
}

func TestGetTraceMetadataSpanEventsDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "1ae969564b34a33ecd1af05fe6923d6d" {
		t.Error(metadata.TraceID)
	}
}

func TestGetTraceMetadataInboundPayload(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		reply.AccountID = "account-id"
		reply.TrustedAccountKey = "123"
		reply.PrimaryAppID = "app-id"
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	hdrs := http.Header{}
	hdrs.Set(internal.DistributedTraceW3CTraceParentHeader, "00-12345678901234567890123456789012-9566c74d10037c4d-01")
	hdrs.Set(internal.DistributedTraceW3CTraceStateHeader, "123@nr=0-0-123-456-9566c74d10037c4d-52fdfc072182654f-1-0.390345-1563574856827")

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectNoLoggedErrors(t)
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "e71870997d57214c" {
		t.Errorf("Invalid Span ID, expected aeceb05d2fdcde0c but got %s", metadata.SpanID)
	}
	if metadata.TraceID != "12345678901234567890123456789012" {
		t.Errorf("Invalid Trace ID, expected 12345678901234567890123456789012 but got %s", metadata.TraceID)
	}
}

func TestGetLinkingMetadata(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.EntityGUID = "entities-are-guid"
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.AppName = "app-name"
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")

	metadata := txn.GetLinkingMetadata()
	host := txn.thread.appRun.Config.hostname
	if metadata.TraceID != "1ae969564b34a33ecd1af05fe6923d6d" {
		t.Error("wrong TraceID:", metadata.TraceID)
	}
	if metadata.SpanID != "e71870997d57214c" {
		t.Error("wrong SpanID:", metadata.SpanID)
	}
	if metadata.EntityName != "app-name" {
		t.Error("wrong EntityName:", metadata.EntityName)
	}
	if metadata.EntityType != "SERVICE" {
		t.Error("wrong EntityType:", metadata.EntityType)
	}
	if metadata.EntityGUID != "entities-are-guid" {
		t.Error("wrong EntityGUID:", metadata.EntityGUID)
	}
	if metadata.Hostname != host {
		t.Error("wrong Hostname:", metadata.Hostname)
	}
}

func TestGetLinkingMetadataAppNames(t *testing.T) {
	testcases := []struct {
		appName  string
		expected string
	}{
		{appName: "one-name", expected: "one-name"},
		{appName: "one-name;two-name;three-name", expected: "one-name"},
		{appName: "", expected: ""},
	}

	for _, test := range testcases {
		cfgfn := func(cfg *Config) {
			cfg.AppName = test.appName
		}
		app := testApp(nil, cfgfn, t)
		txn := app.StartTransaction("hello")

		metadata := txn.GetLinkingMetadata()
		if metadata.EntityName != test.expected {
			t.Errorf("wrong EntityName, actual=%s expected=%s", metadata.EntityName, test.expected)
		}
	}
}

func TestIsSampledFalse(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleNothing()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	sampled := txn.IsSampled()
	if sampled == true {
		t.Error("txn should not be sampled")
	}
}

func TestIsSampledTrue(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	sampled := txn.IsSampled()
	if sampled == false {
		t.Error("txn should be sampled")
	}
}

func TestIsSampledEnded(t *testing.T) {
	// Test that Transaction.IsSampled returns false if the transaction has
	// already ended.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	txn.End()
	sampled := txn.IsSampled()
	if sampled == true {
		t.Error("finished txn should not be sampled")
	}
}

func TestNilTransaction(t *testing.T) {
	var txn *Transaction

	txn.End()
	txn.Ignore()
	txn.SetName("hello")
	txn.NoticeError(errors.New("something"))
	txn.AddAttribute("myKey", "myValue")
	txn.SetWebRequestHTTP(helloRequest)
	var x dummyResponseWriter
	if w := txn.SetWebResponse(x); w != x {
		t.Error(w)
	}
	if start := txn.StartSegmentNow(); !reflect.DeepEqual(start, SegmentStartTime{}) {
		t.Error(start)
	}
	if seg := txn.StartSegment("hello"); !reflect.DeepEqual(seg, &Segment{Name: "hello"}) {
		t.Error(seg)
	}
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) > 0 {
		t.Error(hdrs)
	}
	txn.AcceptDistributedTraceHeaders(TransportHTTP, nil)
	if app := txn.Application(); app != nil {
		t.Error(app)
	}
	if hdr := txn.BrowserTimingHeader(); hdr.WithTags() != nil {
		t.Error(hdr)
	}
	if tx := txn.NewGoroutine(); tx != nil {
		t.Error(tx)
	}
	if m := txn.GetTraceMetadata(); !reflect.DeepEqual(m, TraceMetadata{}) {
		t.Error(m)
	}
	if m := txn.GetLinkingMetadata(); !reflect.DeepEqual(m, LinkingMetadata{}) {
		t.Error(m)
	}
	if s := txn.IsSampled(); s {
		t.Error(s)
	}
}

func TestEmptyTransaction(t *testing.T) {
	txn := &Transaction{}

	txn.End()
	txn.Ignore()
	txn.SetName("hello")
	txn.NoticeError(errors.New("something"))
	txn.AddAttribute("myKey", "myValue")
	txn.SetWebRequestHTTP(helloRequest)
	var x dummyResponseWriter
	if w := txn.SetWebResponse(x); w != x {
		t.Error(w)
	}
	if start := txn.StartSegmentNow(); !reflect.DeepEqual(start, SegmentStartTime{}) {
		t.Error(start)
	}
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) > 0 {
		t.Error(hdrs)
	}
	txn.AcceptDistributedTraceHeaders(TransportHTTP, nil)
	if app := txn.Application(); app != nil {
		t.Error(app)
	}
	if hdr := txn.BrowserTimingHeader(); hdr.WithTags() != nil {
		t.Error(hdr)
	}
	if tx := txn.NewGoroutine(); tx != nil {
		t.Error(tx)
	}
	if m := txn.GetTraceMetadata(); !reflect.DeepEqual(m, TraceMetadata{}) {
		t.Error(m)
	}
	if m := txn.GetLinkingMetadata(); !reflect.DeepEqual(m, LinkingMetadata{}) {
		t.Error(m)
	}
	if s := txn.IsSampled(); s {
		t.Error(s)
	}
}

func TestDTPriority(t *testing.T) {
	type testCase struct {
		name                       string
		incomingSampledAndPriority string
		expectedPriority           string
	}
	// We expect to either receive both a priority and a sampled field, or neither - not one without the other.
	cases := []testCase{
		{
			name:                       "IncludesIncomingPriority",
			incomingSampledAndPriority: `,"sa":true,"pr":1.5`,
			expectedPriority:           "1.5",
		},
		{
			name:                       "NoIncomingPriority",
			incomingSampledAndPriority: "",
			expectedPriority:           "1.315222",
		},
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		reply.DistributedTraceTimestampGenerator = func() time.Time {
			return time.Unix(1577830891, 900000000)
		}
		reply.AccountID = "123"
		reply.TrustedAccountKey = "123"

	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgfn := func(cfg *Config) {
				cfg.DistributedTracer.Enabled = true
			}
			app := testApp(replyfn, cfgfn, t)
			txn := app.StartTransaction("hello")

			inboundHdrs := map[string][]string{
				DistributedTraceNewRelicHeader: {`{"v":[0,1],"d":{"ty":"App","ap":"456","ac":"123","id":"myid","tr":"mytrip","ti":1574881875872` +
					tc.incomingSampledAndPriority + "}}",
				},
			}

			txn.AcceptDistributedTraceHeaders(TransportHTTP, inboundHdrs)
			outboundHdrs := http.Header{}
			txn.InsertDistributedTraceHeaders(outboundHdrs)
			traceState := outboundHdrs.Get(DistributedTraceW3CTraceStateHeader)
			if traceState != "123@nr=0-0-123--e71870997d57214c-1ae969564b34a33e-1-"+tc.expectedPriority+"-1577830891900" {
				t.Error(tc.expectedPriority, traceState)
			}
		})

	}
}
