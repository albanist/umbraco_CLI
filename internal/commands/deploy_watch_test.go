package commands

import (
	"strings"
	"testing"
	"time"
)

func watchBaseline() watchObservation {
	return watchObservation{
		At:          time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
		MgmtAlive:   true,
		MgmtStatus:  401,
		ProcessID:   "1000",
		MachineName: "web-a",
		Health:      map[string]bool{"/": true},
		BadIndexes:  []string{},
	}
}

func phases(events []watchEvent) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Phase)
	}
	return names
}

func TestWatchMachineGreenDeploySequence(t *testing.T) {
	machine, err := newWatchMachine(watchBaseline(), 10*time.Minute, false)
	if err != nil {
		t.Fatalf("baseline failed: %v", err)
	}
	at := watchBaseline().At

	var all []string
	step := func(obs watchObservation, wantOutcome watchOutcome) {
		t.Helper()
		events, outcome := machine.observe(obs)
		all = append(all, phases(events)...)
		if outcome != wantOutcome {
			t.Fatalf("unexpected outcome %v at %s (events so far %v)", outcome, obs.At, all)
		}
	}

	// Origin goes down (503).
	step(watchObservation{At: at.Add(10 * time.Second), MgmtAlive: false, MgmtStatus: 503, Health: map[string]bool{"/": false}}, watchOutcomeNone)
	step(watchObservation{At: at.Add(40 * time.Second), MgmtAlive: false, MgmtStatus: 503, Health: map[string]bool{"/": false}}, watchOutcomeNone)
	// Token endpoint flips to 401 before public pages return; new ProcessId.
	step(watchObservation{At: at.Add(70 * time.Second), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": false}, BadIndexes: []string{"ExternalIndex"}}, watchOutcomeNone)
	// Public pages come back ~15s later, index still rebuilding.
	step(watchObservation{At: at.Add(85 * time.Second), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": true}, BadIndexes: []string{"ExternalIndex"}}, watchOutcomeNone)
	// Index rebuild finishes → verified.
	step(watchObservation{At: at.Add(17 * time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": true}, BadIndexes: []string{}}, watchOutcomeVerified)

	want := []string{"restarting", "app-alive", "landed", "serving", "verified"}
	if strings.Join(all, ",") != strings.Join(want, ",") {
		t.Fatalf("expected phase sequence %v, got %v", want, all)
	}
}

func TestWatchMachineIndexRebuildBlocksVerified(t *testing.T) {
	machine, _ := newWatchMachine(watchBaseline(), 10*time.Minute, false)
	at := watchBaseline().At
	obs := watchObservation{At: at.Add(time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": true}, BadIndexes: []string{"ExternalIndex"}}
	_, outcome := machine.observe(obs)
	if outcome != watchOutcomeNone {
		t.Fatalf("expected rebuild to block verified ('deploy succeeded' and 'the site works' are different questions), got %v", outcome)
	}
	obs.At = at.Add(2 * time.Minute)
	obs.BadIndexes = nil // index state unreadable this tick — never treated as clean
	if _, outcome := machine.observe(obs); outcome != watchOutcomeNone {
		t.Fatalf("expected unknown index state to block verified, got %v", outcome)
	}
	obs.At = at.Add(3 * time.Minute)
	obs.BadIndexes = []string{}
	if _, outcome := machine.observe(obs); outcome != watchOutcomeVerified {
		t.Fatalf("expected verified once indexes clean, got %v", outcome)
	}
}

func TestWatchMachineBaselineBadIndexIsNotASignal(t *testing.T) {
	baseline := watchBaseline()
	baseline.BadIndexes = []string{"BrokenSinceForever"}
	machine, _ := newWatchMachine(baseline, 10*time.Minute, false)
	obs := watchObservation{At: baseline.At.Add(time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": true}, BadIndexes: []string{"BrokenSinceForever"}}
	_, outcome := machine.observe(obs)
	if outcome != watchOutcomeVerified {
		t.Fatalf("expected pre-existing bad index to be excluded from verification, got %v", outcome)
	}
}

func TestWatchMachineEscalatesSustainedDowntime(t *testing.T) {
	machine, _ := newWatchMachine(watchBaseline(), 10*time.Minute, false)
	at := watchBaseline().At
	if _, outcome := machine.observe(watchObservation{At: at.Add(time.Minute), MgmtAlive: false, MgmtStatus: 503}); outcome != watchOutcomeNone {
		t.Fatalf("expected downtime under threshold to keep watching")
	}
	events, outcome := machine.observe(watchObservation{At: at.Add(11 * time.Minute), MgmtAlive: false, MgmtStatus: 503})
	if outcome != watchOutcomeFailed {
		t.Fatalf("expected sustained downtime to fail, got %v", outcome)
	}
	if phases(events)[len(events)-1] != "failed" || !strings.Contains(machine.failureReason, "down for") {
		t.Fatalf("expected failed event with downtime reason, got %v / %q", phases(events), machine.failureReason)
	}
}

func TestWatchMachineEscalatesPostLandingHealthFailure(t *testing.T) {
	machine, _ := newWatchMachine(watchBaseline(), 5*time.Minute, true)
	at := watchBaseline().At
	// Deploy lands but the site never comes back.
	if _, outcome := machine.observe(watchObservation{At: at.Add(time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": false}}); outcome != watchOutcomeNone {
		t.Fatalf("expected failing health under threshold to keep watching")
	}
	events, outcome := machine.observe(watchObservation{At: at.Add(7 * time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": false}})
	if outcome != watchOutcomeFailed {
		t.Fatalf("expected post-landing health failure to fail, got %v (%v)", outcome, phases(events))
	}
	if !strings.Contains(machine.failureReason, "after the deploy landed") {
		t.Fatalf("unexpected failure reason %q", machine.failureReason)
	}
}

func TestWatchMachineFastRecycleSkipsPhases(t *testing.T) {
	machine, _ := newWatchMachine(watchBaseline(), 10*time.Minute, true)
	at := watchBaseline().At
	// PID changed between polls without an observed down window.
	events, outcome := machine.observe(watchObservation{At: at.Add(time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": true}})
	if outcome != watchOutcomeVerified {
		t.Fatalf("expected fast recycle to reach verified, got %v", outcome)
	}
	want := []string{"landed", "serving", "verified"}
	if strings.Join(phases(events), ",") != strings.Join(want, ",") {
		t.Fatalf("expected skip-phase sequence %v, got %v", want, phases(events))
	}
}

func TestWatchMachineMachineNameChangeIsALandingSignal(t *testing.T) {
	machine, _ := newWatchMachine(watchBaseline(), 10*time.Minute, true)
	events, _ := machine.observe(watchObservation{At: watchBaseline().At.Add(time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "1000", MachineName: "web-b", Health: map[string]bool{"/": true}})
	if len(events) == 0 || events[0].Phase != "landed" {
		t.Fatalf("expected slot swap (MachineName change) to land, got %v", phases(events))
	}
}

func TestWatchMachineBaselineUnhealthyPathIsNotASignal(t *testing.T) {
	baseline := watchBaseline()
	baseline.Health = map[string]bool{"/": true, "/broken": false}
	machine, _ := newWatchMachine(baseline, 10*time.Minute, true)
	// /broken still failing after landing must not block serving/verified.
	_, outcome := machine.observe(watchObservation{At: baseline.At.Add(time.Minute), MgmtAlive: true, MgmtStatus: 401, ProcessID: "2000", MachineName: "web-a", Health: map[string]bool{"/": true, "/broken": false}})
	if outcome != watchOutcomeVerified {
		t.Fatalf("expected baseline-unhealthy path to be excluded, got %v", outcome)
	}
}

func TestWatchMachineRefusesToArmMidOutageOrBlind(t *testing.T) {
	down := watchBaseline()
	down.MgmtAlive = false
	if _, err := newWatchMachine(down, time.Minute, false); err == nil || !strings.Contains(err.Error(), "refusing to arm") {
		t.Fatalf("expected arming refusal mid-outage, got %v", err)
	}
	blind := watchBaseline()
	blind.ProcessID = ""
	if _, err := newWatchMachine(blind, time.Minute, false); err == nil || !strings.Contains(err.Error(), "ProcessId") {
		t.Fatalf("expected arming refusal without a landing signal, got %v", err)
	}
}
