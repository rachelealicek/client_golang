package prometheus

import (
	"fmt"
	"sync"
)

type metricMap struct {
	mtx       sync.RWMutex
	metrics   map[uint64][]metricWithLabels
	desc      *Desc
	newMetric func(labelValues ...string) Metric
}

type metricWithLabels struct {
	values []string
	metric Metric
}

func (m *metricMap) getOrCreateMetricWithLabelValues(lvs []string) Metric {
	h := hashLvs(lvs)

	m.mtx.RLock()
	metric, ok := m.getMetricWithHashAndLabelValues(h, lvs)
	m.mtx.RUnlock()
	if ok {
		return metric
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	metric, ok = m.getMetricWithHashAndLabelValues(h, lvs)
	if !ok {
		metric = m.newMetric(lvs...)
		// Copy slice to avoid race condition with deleteLabelValues modifying the slice in-place
		oldMetrics := m.metrics[h]
		newMetrics := make([]metricWithLabels, len(oldMetrics), len(oldMetrics)+1)
		copy(newMetrics, oldMetrics)
		newMetrics = append(newMetrics, metricWithLabels{values: lvs, metric: metric})
		m.metrics[h] = newMetrics
	}
	return metric
}

func (m *metricMap) getMetricWithHashAndLabelValues(h uint64, lvs []string) (Metric, bool) {
	metrics, ok := m.metrics[h]
	if !ok {
		return nil, false
	}
	for _, val := range metrics {
		if equalLV(val.values, lvs) {
			return val.metric, true
		}
	}
	return nil, false
}

func (m *metricMap) deleteLabelValues(lvs []string) bool {
	h := hashLvs(lvs)

	m.mtx.Lock()
	defer m.mtx.Unlock()

	metrics, ok := m.metrics[h]
	if !ok {
		return false
	}
	for i, val := range metrics {
		if equalLV(val.values, lvs) {
			// Create a new slice instead of modifying the existing one in-place with append
			newMetrics := make([]metricWithLabels, 0, len(metrics)-1)
			newMetrics = append(newMetrics, metrics[:i]...)
			newMetrics = append(newMetrics, metrics[i+1:]...)
			if len(newMetrics) == 0 {
				delete(m.metrics, h)
			} else {
				m.metrics[h] = newMetrics
			}
			return true
		}
	}
	return false
}
