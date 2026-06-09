/*
 * SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package alarm

import (
	"testing"

	"github.com/toobuntu/zman-didan/internal/types"
)

func makeEvent(category string, allDay bool) types.HebcalEvent {
	return types.HebcalEvent{
		Category: category,
		AllDay:   allDay,
		Alarms:   []types.Alarm{{TriggerMinutes: -999, Description: "old"}},
	}
}

func TestRebuild_Candles(t *testing.T) {
	events := []types.HebcalEvent{makeEvent("candles", false)}
	Rebuild(events)
	if len(events[0].Alarms) != 2 {
		t.Fatalf("want 2 alarms, got %d", len(events[0].Alarms))
	}
	if events[0].Alarms[0].TriggerMinutes != -120 {
		t.Errorf("alarm[0] = %d, want -120", events[0].Alarms[0].TriggerMinutes)
	}
	if events[0].Alarms[1].TriggerMinutes != -15 {
		t.Errorf("alarm[1] = %d, want -15", events[0].Alarms[1].TriggerMinutes)
	}
}

func TestRebuild_Havdalah(t *testing.T) {
	events := []types.HebcalEvent{makeEvent("havdalah", false)}
	Rebuild(events)
	if len(events[0].Alarms) != 2 {
		t.Fatalf("want 2 alarms, got %d", len(events[0].Alarms))
	}
	if events[0].Alarms[0].TriggerMinutes != -10 {
		t.Errorf("alarm[0] = %d, want -10", events[0].Alarms[0].TriggerMinutes)
	}
	if events[0].Alarms[1].TriggerMinutes != 0 {
		t.Errorf("alarm[1] = %d, want 0", events[0].Alarms[1].TriggerMinutes)
	}
}

func TestRebuild_FastBegin(t *testing.T) {
	events := []types.HebcalEvent{makeEvent("fast-begin", false)}
	Rebuild(events)
	if len(events[0].Alarms) != 2 {
		t.Fatalf("want 2 alarms, got %d", len(events[0].Alarms))
	}
	if events[0].Alarms[0].TriggerMinutes != -120 {
		t.Errorf("alarm[0] = %d, want -120", events[0].Alarms[0].TriggerMinutes)
	}
	if events[0].Alarms[1].TriggerMinutes != -30 {
		t.Errorf("alarm[1] = %d, want -30", events[0].Alarms[1].TriggerMinutes)
	}
}

func TestRebuild_FastEnd(t *testing.T) {
	events := []types.HebcalEvent{makeEvent("fast-end", false)}
	Rebuild(events)
	if len(events[0].Alarms) != 2 {
		t.Fatalf("want 2 alarms, got %d", len(events[0].Alarms))
	}
	if events[0].Alarms[0].TriggerMinutes != -15 {
		t.Errorf("alarm[0] = %d, want -15", events[0].Alarms[0].TriggerMinutes)
	}
	if events[0].Alarms[1].TriggerMinutes != 0 {
		t.Errorf("alarm[1] = %d, want 0", events[0].Alarms[1].TriggerMinutes)
	}
}

func TestRebuild_AllDayCleared(t *testing.T) {
	events := []types.HebcalEvent{makeEvent("holiday", true)}
	Rebuild(events)
	if events[0].Alarms != nil {
		t.Errorf("all-day event should have nil alarms, got %v", events[0].Alarms)
	}
}

func TestRebuild_UnknownCategoryCleared(t *testing.T) {
	events := []types.HebcalEvent{makeEvent("roshchodesh", false)}
	Rebuild(events)
	if events[0].Alarms != nil {
		t.Errorf("unknown category should have nil alarms, got %v", events[0].Alarms)
	}
}

func TestRebuild_AlarmDescription(t *testing.T) {
	events := []types.HebcalEvent{makeEvent("candles", false)}
	Rebuild(events)
	for _, a := range events[0].Alarms {
		if a.Description != "Event reminder" {
			t.Errorf("alarm description = %q, want %q", a.Description, "Event reminder")
		}
	}
}

func TestRebuild_ClearsExistingAlarms(t *testing.T) {
	ev := makeEvent("candles", false)
	ev.Alarms = []types.Alarm{
		{TriggerMinutes: -999, Description: "stale"},
		{TriggerMinutes: -888, Description: "stale2"},
		{TriggerMinutes: -777, Description: "stale3"},
	}
	events := []types.HebcalEvent{ev}
	Rebuild(events)
	if len(events[0].Alarms) != 2 {
		t.Fatalf("want 2 alarms after rebuild, got %d", len(events[0].Alarms))
	}
	if events[0].Alarms[0].TriggerMinutes != -120 {
		t.Errorf("stale alarm not replaced: %d", events[0].Alarms[0].TriggerMinutes)
	}
}
