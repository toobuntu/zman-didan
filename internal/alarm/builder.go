// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

// Package alarm strips existing alarms from all events and rebuilds them
// according to the didan alarm policy.
package alarm

import "github.com/toobuntu/zman-didan/internal/types"

// Rebuild clears all existing Alarms and re-applies the policy table:
//
//	candles    → −2h, −15min
//	havdalah   → −10min, at event time
//	fast-begin → −2h, −30min
//	fast-end   → −15min, at event time
//	all-day    → none
//	all others → none
//
// All alarm DESCRIPTION values use the generic "Event reminder" string,
// consistent with Hebcal's iCal output. Apple Calendar does not display
// the DESCRIPTION field of VALARM components in its UI.
func Rebuild(events []types.HebcalEvent) {
	const reminder = "Event reminder"
	for i := range events {
		ev := &events[i]
		if ev.AllDay {
			ev.Alarms = nil
			continue
		}
		switch ev.Category {
		case "candles":
			ev.Alarms = []types.Alarm{
				{TriggerMinutes: -120, Description: reminder},
				{TriggerMinutes: -15, Description: reminder},
			}
		case "havdalah":
			ev.Alarms = []types.Alarm{
				{TriggerMinutes: -10, Description: reminder},
				{TriggerMinutes: 0, Description: reminder},
			}
		case "fast-begin":
			ev.Alarms = []types.Alarm{
				{TriggerMinutes: -120, Description: reminder},
				{TriggerMinutes: -30, Description: reminder},
			}
		case "fast-end":
			ev.Alarms = []types.Alarm{
				{TriggerMinutes: -15, Description: reminder},
				{TriggerMinutes: 0, Description: reminder},
			}
		default:
			ev.Alarms = nil
		}
	}
}
