package store

import "time"

type Counts struct {
	Devices   int       `json:"devices"`
	Streams   int       `json:"streams"`
	Policies  int       `json:"policies"`
	Samples   int       `json:"samples"`
	Anomalies int       `json:"anomalies"`
	Incidents int       `json:"incidents"`
	Actions   int       `json:"actions"`
	Evidence  int       `json:"evidence"`
	At        time.Time `json:"at"`
}

func (m *Memory) Counts(now time.Time) Counts {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Counts{
		Devices: len(m.devices), Streams: len(m.streams), Policies: len(m.policies),
		Samples: len(m.samples), Anomalies: len(m.anomalies), Incidents: len(m.incidents),
		Actions: len(m.actions), Evidence: len(m.evidence), At: now,
	}
}
