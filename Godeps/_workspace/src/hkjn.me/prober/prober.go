// Package prober provides black-box monitoring mechanisms.
//
// To use, define Probe() and Alert() on a type, then pass it to NewProbe:
//   struct FooProber{ someState int }
//
//   // Probe "Foo". E.g. do a network call and compare it to what
//   // was expected.
//   func (p FooProber) Probe() error {
//     // Returning non-nil indicates that the probe failed.
//   }
//   // Send an alert. Called if the probe fails too often.
//   func (p FooProber) Alert() error {
//   }
//   ...
//
//   // Create the probe.
//   p := prober.NewProbe(FooProber{1}, "FooProber", "Probes the Foo")
//
//   // Run the probe. This call blocks forever, so you may
//   // want to do this in a goroutine — you could e.g. register a web
//   // handler to show the contents of p.Records here.
//   go p.Run()
package prober // import "hkjn.me/prober"

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"hkjn.me/timeutils"
)

var (
	MaxAlertFrequency = time.Minute * 15 // never send alerts more often than this
	DefaultInterval   = flag.Duration("probe_interval", time.Second*61, "duration to pause between prober runs")
	logDir            = os.TempDir()          // logging directory
	logName           = "prober.outcomes.log" // name of logging file
	alertThreshold    = flag.Int("alert_threshold", 100, "level of 'badness' before alerting")
	alertsDisabled    = flag.Bool("no_alerts", false, "disables alerts when probes fail too often")
	disabledProbes    = make(selectedProbes)
	onlyProbes        = make(selectedProbes)
	defaultMinBadness = 0  // default minimum allowed `badness`
	defaultBadnessInc = 10 // default increment on failed probe
	defaultBadnessDec = 1  // default decrement on successful probe
	onceOpen          sync.Once
	logFile           *os.File
	bufferSize        = 200 // maximum number of results per prober to keep
	parseFlags        = sync.Once{}
)

// Prober is a mechanism that can probe some target(s).
type Prober interface {
	Probe() error                                                // probe target(s) once
	Alert(name, desc string, badness int, records Records) error // send alert
}

// selectedProbes is a set of probes to be enabled/disabled.
type selectedProbes map[string]bool

// String returns the flag's value.
func (d *selectedProbes) String() string {
	s := ""
	i := 0
	for p, _ := range *d {
		if i > 0 {
			s += ","
		}
		s += p
	}
	return s
}

// Get is part of the flag.Getter interface. It always returns nil for
// this flag type since the struct is not exported.
func (d *selectedProbes) Get() interface{} {
	return nil
}

// Syntax: -disable_probes=FooProbe,BarProbe
func (d *selectedProbes) Set(value string) error {
	vals := strings.Split(value, ",")
	m := *d
	for _, p := range vals {
		m[p] = true
	}
	return nil
}

// Probe is a stateful representation of repeated probe runs.
type Probe struct {
	Prober            // underlying prober mechanism
	Name, Desc string // name, description of the probe
	// If badness reaches alert threshold, an alert email is sent and
	// alertThreshold resets.
	Badness    int
	Interval   time.Duration // how often to probe
	Timeout    time.Duration // timeout for probe call, defaults to same as probing inteval
	Alerting   bool          // whether this probe is currently alerting
	LastAlert  time.Time     // time of last alert sent, if any
	Disabled   bool          // whether this probe is disabled
	Records    Records       // records of probe runs
	minBadness int           // minimum allowed `badness` value
	badnessInc int           // how much to increment `badness` with on failure
	badnessDec int           // how much to decrement `badness` with on success
}

// NewProbe returns a new probe from given prober implementation.
func NewProbe(p Prober, name, desc string, options ...func(*Probe)) *Probe {
	parseFlags.Do(func() {
		if !flag.Parsed() {
			flag.Parse()
		}
	})
	probe := &Probe{
		Prober:     p,
		Name:       name,
		Desc:       desc,
		Badness:    defaultMinBadness,
		Interval:   *DefaultInterval,
		Timeout:    *DefaultInterval,
		Records:    Records{},
		minBadness: defaultMinBadness,
		badnessInc: defaultBadnessInc,
		badnessDec: defaultBadnessDec,
	}
	for _, opt := range options {
		opt(probe)
	}
	return probe
}

// Interval sets the interval option for the prober.
func Interval(interval time.Duration) func(*Probe) {
	return func(p *Probe) {
		p.Interval = interval
	}
}

// Timeout sets the timeout option for the prober.
func Timeout(timeout time.Duration) func(*Probe) {
	return func(p *Probe) {
		p.Timeout = timeout
	}
}

// FailurePenalty sets the amount `badness` is incremented on failure for the prober.
func FailurePenalty(badnessInc int) func(*Probe) {
	return func(p *Probe) {
		p.badnessInc = badnessInc
	}
}

// SuccessReward sets the amount `badness` is decremented on success for the prober.
func SuccessReward(badnessDec int) func(*Probe) {
	return func(p *Probe) {
		p.badnessDec = badnessDec
	}
}

// Run repeatedly runs the probe, blocking forever.
func (p *Probe) Run() {
	glog.Infof("[%s] Starting..\n", p.Name)

	for {
		if p.enabled() {
			p.runProbe()
		} else {
			p.Disabled = true
			glog.Infof("[%s] is disabled, will now exit", p.Name)
			return
		}
	}
}

func (p *Probe) enabled() bool {
	if len(onlyProbes) > 0 {
		if _, ok := onlyProbes[p.Name]; ok {
			// We only want specific probes, but we do want this one.
			return true
		}
		return false
	}

	if _, ok := disabledProbes[p.Name]; ok {
		// This probe is explicitly disabled.
		return false
	}
	return true
}

// runProbe runs the probe once.
func (p *Probe) runProbe() {
	c := make(chan error, 1)
	start := time.Now().UTC()
	go func() {
		glog.Infof("[%s] Probing..\n", p.Name)
		c <- p.Probe()
	}()
	select {
	case err := <-c:
		// We got a result of some sort from the prober.
		p.handleResult(err)
		wait := p.Timeout - time.Since(start)
		glog.V(2).Infof("[%s] needs to sleep %v more here\n", p.Name, wait)
		time.Sleep(wait)
	case <-time.After(p.Interval):
		// Probe didn't finish in time for us to run the next one, report as failure.
		glog.Errorf("[%s] Timed out\n", p.Name)
		p.handleResult(fmt.Errorf("%s timed out (with probe interval %1.1f sec)", p.Name, p.Interval.Seconds()))
	}
}

// add appends the record to the buffer for the probe, keeping it within bufferSize.
func (p *Probe) addRecord(r Record) {
	p.Records = append(p.Records, r)
	if len(p.Records) >= bufferSize {
		over := len(p.Records) - bufferSize
		glog.V(2).Infof("[%s] buffer is over %d, reslicing it\n", p.Name, bufferSize)
		p.Records = p.Records[over:]
	}
	glog.V(2).Infof("[%s] buffer is now %d elements\n", p.Name, len(p.Records))
}

// Records is a grouping of probe records that implements sort.Interface.
type Records []Record

func (pr Records) Len() int           { return len(pr) }
func (pr Records) Swap(i, j int)      { pr[i], pr[j] = pr[j], pr[i] }
func (pr Records) Less(i, j int) bool { return pr[i].Timestamp.Before(pr[j].Timestamp) }

// RecentFailures returns only recent probe failures among the records.
func (pr Records) RecentFailures() Records {
	failures := make(Records, 0)
	for _, r := range pr {
		if !r.Passed && !r.Timestamp.Before(time.Now().Add(-time.Hour)) {
			failures = append(failures, r)
		}
	}
	sort.Sort(sort.Reverse(failures))
	return failures
}

// Record is the result of a single probe run.
type Record struct {
	Timestamp  time.Time `yaml: "-"`
	TimeMillis string    // same as Timestamp but makes it into the YAML logs
	Passed     bool
	Details    string `yaml: "omitempty"`
}

// Ago describes the duration since the record occured.
func (r Record) Ago() string {
	return timeutils.DescDuration(time.Since(r.Timestamp))
}

// newRecord returns a new record.
func newRecord(passed bool, line string) Record {
	now := time.Now().UTC()
	return Record{
		Timestamp:  now,
		TimeMillis: now.Format(time.StampMilli),
		Passed:     passed,
		Details:    line,
	}
}

// Marshal returns the record in YAML form.
func (r Record) marshal() []byte {
	b, err := yaml.Marshal(r)
	if err != nil {
		glog.Fatalf("failed to marshal record %+v: %v", r, err)
	}
	return b
}

// openLog opens the log file.
func openLog() {
	logPath := filepath.Join(logDir, logName)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		glog.Fatalf("failed to open %q: %v\n", logPath, err)
	}
	logFile = f
}

// handleResult handles a return value from a probe() run.
func (p *Probe) handleResult(err error) {
	if err != nil {
		p.Badness += p.badnessInc
		glog.Errorf("[%s] Failed while probing, badness is now %d: %v\n", p.Name, p.Badness, err)
		p.logFail(err)
	} else {
		if p.Badness > p.minBadness {
			p.Badness -= p.badnessDec
		}
		glog.Infof("[%s] Pass, badness is now %d.\n", p.Name, p.Badness)
		p.logPass()
	}

	if p.Badness < *alertThreshold {
		p.Alerting = false
		return
	}

	p.Alerting = true
	if *alertsDisabled {
		glog.Infof("[%s] would now be alerting, but alerts are supressed\n", p.Name)
		return
	}

	glog.Infof("[%s] is alerting\n", p.Name)
	if time.Since(p.LastAlert) < MaxAlertFrequency {
		glog.V(1).Infof("[%s] will not alert, since last alert was sent %v back\n", p.Name, time.Since(p.LastAlert))
		return
	}
	// Send alert notification in goroutine to not block further
	// probing.
	// TODO: There is a race condition here, if email sending takes long
	// enough for further Probe() runs to finish, which would queue up
	// several duplicate alert emails. This shouldn't often happen, but
	// technically should be bounded by a timeout to prevent the
	// possibility.
	go p.sendAlert()
}

// sendAlert calls the Alert() implementation and handles the outcome.
func (p *Probe) sendAlert() {
	err := p.Alert(p.Name, p.Desc, p.Badness, p.Records)
	if err != nil {
		glog.Errorf("[%s] failed to alert: %v", p.Name, err)
		// Note: We don't reset badness here; next cycle we'll keep
		// trying to send the alert.
	} else {
		glog.Infof("[%s] sent alert email, resetting badness to 0\n", p.Name)
		p.LastAlert = time.Now().UTC()
		p.Badness = p.minBadness
	}
}

// logFail logs a failed probe run.
func (p *Probe) logFail(err error) {
	onceOpen.Do(openLog)
	r := newRecord(false, err.Error())
	p.addRecord(r)
	_, err = logFile.Write(r.marshal())
	if err != nil {
		glog.Fatalf("failed to write failure to log: %v", err)
	}
}

// logPass logs a successful probe run.
func (p *Probe) logPass() {
	onceOpen.Do(openLog)
	r := newRecord(true, "")
	p.addRecord(r)
	_, err := logFile.Write(r.marshal())
	if err != nil {
		glog.Fatalf("failed to write success to log: %v", err)
	}
}

func init() {
	flag.Var(&disabledProbes, "disabled_probes", "comma-separated list of probes to disable")
	flag.Var(&onlyProbes, "only_probes", "comma-separated list of the only probes to enable")
}
