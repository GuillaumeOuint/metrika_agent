// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !nonetclass && linux
// +build !nonetclass,linux

package collector

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/sysfs"
	"go.uber.org/zap"
)

var (
	// netclassIgnoredDevices Regexp of net devices to ignore for netclass collector.
	// collector.netclass.ignored-devices
	netclassIgnoredDevices = "^$"

	// netclassInvalidSpeed Ignore devices where the speed is invalid. This will be
	// the default behavior in 2.x.
	// collector.netclass.ignore-invalid-speed
	netclassInvalidSpeed = false
)

type netClassCollector struct {
	fs                    sysfs.FS
	subsystem             string
	ignoredDevicesPattern *regexp.Regexp
	metricDescs           map[string]*prometheus.Desc
}

// NewNetClassCollector returns a new Collector exposing network class stats.
func NewNetClassCollector() (prometheus.Collector, error) {
	fs, err := sysfs.NewFS(sysPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sysfs: %w", err)
	}
	pattern := regexp.MustCompile(netclassIgnoredDevices)
	return &netClassCollector{
		fs:                    fs,
		subsystem:             "network",
		ignoredDevicesPattern: pattern,
		metricDescs:           map[string]*prometheus.Desc{},
	}, nil
}

func (c *netClassCollector) Collect(ch chan<- prometheus.Metric) {
	netClass, err := c.getNetClassInfo()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			zap.S().Debugw("Could not read netclass file", "err", err)
			zap.S().Error(ErrNoData)

			return
		}
		err = fmt.Errorf("could not get net class info: %w", err)
		zap.S().Error(err)

		return
	}
	for _, ifaceInfo := range netClass {
		upDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, c.subsystem, "up"),
			"Value is 1 if operstate is 'up', 0 otherwise.",
			[]string{"device"},
			nil,
		)
		upValue := 0.0
		if ifaceInfo.OperState == "up" {
			upValue = 1.0
		}

		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, upValue, ifaceInfo.Name)

		infoDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, c.subsystem, "info"),
			"Non-numeric data from /sys/class/net/<iface>, value is always 1.",
			[]string{"device", "address", "broadcast", "duplex", "operstate", "ifalias"},
			nil,
		)
		infoValue := 1.0

		ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.GaugeValue, infoValue, ifaceInfo.Name, ifaceInfo.Address, ifaceInfo.Broadcast, ifaceInfo.Duplex, ifaceInfo.OperState, ifaceInfo.IfAlias)

		if ifaceInfo.AddrAssignType != nil {
			pushMetric(ch, c.subsystem, "address_assign_type", *ifaceInfo.AddrAssignType, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.Carrier != nil {
			pushMetric(ch, c.subsystem, "carrier", *ifaceInfo.Carrier, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.CarrierChanges != nil {
			pushMetric(ch, c.subsystem, "carrier_changes_total", *ifaceInfo.CarrierChanges, ifaceInfo.Name, prometheus.CounterValue)
		}

		if ifaceInfo.CarrierUpCount != nil {
			pushMetric(ch, c.subsystem, "carrier_up_changes_total", *ifaceInfo.CarrierUpCount, ifaceInfo.Name, prometheus.CounterValue)
		}

		if ifaceInfo.CarrierDownCount != nil {
			pushMetric(ch, c.subsystem, "carrier_down_changes_total", *ifaceInfo.CarrierDownCount, ifaceInfo.Name, prometheus.CounterValue)
		}

		if ifaceInfo.DevID != nil {
			pushMetric(ch, c.subsystem, "device_id", *ifaceInfo.DevID, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.Dormant != nil {
			pushMetric(ch, c.subsystem, "dormant", *ifaceInfo.Dormant, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.Flags != nil {
			pushMetric(ch, c.subsystem, "flags", *ifaceInfo.Flags, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.IfIndex != nil {
			pushMetric(ch, c.subsystem, "iface_id", *ifaceInfo.IfIndex, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.IfLink != nil {
			pushMetric(ch, c.subsystem, "iface_link", *ifaceInfo.IfLink, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.LinkMode != nil {
			pushMetric(ch, c.subsystem, "iface_link_mode", *ifaceInfo.LinkMode, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.MTU != nil {
			pushMetric(ch, c.subsystem, "mtu_bytes", *ifaceInfo.MTU, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.NameAssignType != nil {
			pushMetric(ch, c.subsystem, "name_assign_type", *ifaceInfo.NameAssignType, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.NetDevGroup != nil {
			pushMetric(ch, c.subsystem, "net_dev_group", *ifaceInfo.NetDevGroup, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.Speed != nil {
			// Some devices return -1 if the speed is unknown.
			if *ifaceInfo.Speed >= 0 || !netclassInvalidSpeed {
				speedBytes := int64(*ifaceInfo.Speed * 1000 * 1000 / 8)
				pushMetric(ch, c.subsystem, "speed_bytes", speedBytes, ifaceInfo.Name, prometheus.GaugeValue)
			}
		}

		if ifaceInfo.TxQueueLen != nil {
			pushMetric(ch, c.subsystem, "transmit_queue_length", *ifaceInfo.TxQueueLen, ifaceInfo.Name, prometheus.GaugeValue)
		}

		if ifaceInfo.Type != nil {
			pushMetric(ch, c.subsystem, "protocol_type", *ifaceInfo.Type, ifaceInfo.Name, prometheus.GaugeValue)
		}
	}

	return
}

func pushDesc(ch chan<- *prometheus.Desc, subsystem string, name string) {
	fieldDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, name),
		fmt.Sprintf("%s value of /sys/class/net/<iface>.", name),
		[]string{"device"},
		nil,
	)

	ch <- fieldDesc
}

func pushMetric(ch chan<- prometheus.Metric, subsystem string, name string, value int64, ifaceName string, valueType prometheus.ValueType) {
	fieldDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, name),
		fmt.Sprintf("%s value of /sys/class/net/<iface>.", name),
		[]string{"device"},
		nil,
	)

	ch <- prometheus.MustNewConstMetric(fieldDesc, valueType, float64(value), ifaceName)
}

func (c *netClassCollector) getNetClassInfo() (sysfs.NetClass, error) {
	netClass := sysfs.NetClass{}
	netDevices, err := c.fs.NetClassDevices()
	if err != nil {
		return netClass, err
	}

	for _, device := range netDevices {
		if c.ignoredDevicesPattern.MatchString(device) {
			continue
		}
		interfaceClass, err := c.fs.NetClassByIface(device)
		if err != nil {
			return netClass, err
		}
		netClass[device] = *interfaceClass
	}

	return netClass, nil
}

func (c *netClassCollector) Describe(ch chan<- *prometheus.Desc) {
	netClass, err := c.getNetClassInfo()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			zap.S().Debugw("Could not read netclass file", "err", err)
			zap.S().Error(ErrNoData)

			return
		}
		err = fmt.Errorf("could not get net class info: %w", err)
		zap.S().Error(err)

		return
	}
	for _, ifaceInfo := range netClass {
		upDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, c.subsystem, "up"),
			"Value is 1 if operstate is 'up', 0 otherwise.",
			[]string{"device"},
			nil,
		)

		ch <- upDesc

		infoDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, c.subsystem, "info"),
			"Non-numeric data from /sys/class/net/<iface>, value is always 1.",
			[]string{"device", "address", "broadcast", "duplex", "operstate", "ifalias"},
			nil,
		)

		ch <- infoDesc

		if ifaceInfo.AddrAssignType != nil {
			pushDesc(ch, c.subsystem, "address_assign_type")
		}

		if ifaceInfo.Carrier != nil {
			pushDesc(ch, c.subsystem, "carrier")
		}

		if ifaceInfo.CarrierChanges != nil {
			pushDesc(ch, c.subsystem, "carrier_changes_total")
		}

		if ifaceInfo.CarrierUpCount != nil {
			pushDesc(ch, c.subsystem, "carrier_up_changes_total")
		}

		if ifaceInfo.CarrierDownCount != nil {
			pushDesc(ch, c.subsystem, "carrier_down_changes_total")
		}

		if ifaceInfo.DevID != nil {
			pushDesc(ch, c.subsystem, "device_id")
		}

		if ifaceInfo.Dormant != nil {
			pushDesc(ch, c.subsystem, "dormant")
		}

		if ifaceInfo.Flags != nil {
			pushDesc(ch, c.subsystem, "flags")
		}

		if ifaceInfo.IfIndex != nil {
			pushDesc(ch, c.subsystem, "iface_id")
		}

		if ifaceInfo.IfLink != nil {
			pushDesc(ch, c.subsystem, "iface_link")
		}

		if ifaceInfo.LinkMode != nil {
			pushDesc(ch, c.subsystem, "iface_link_mode")
		}

		if ifaceInfo.MTU != nil {
			pushDesc(ch, c.subsystem, "mtu_bytes")
		}

		if ifaceInfo.NameAssignType != nil {
			pushDesc(ch, c.subsystem, "name_assign_type")
		}

		if ifaceInfo.NetDevGroup != nil {
			pushDesc(ch, c.subsystem, "net_dev_group")
		}

		if ifaceInfo.Speed != nil {
			// Some devices return -1 if the speed is unknown.
			if *ifaceInfo.Speed >= 0 || !netclassInvalidSpeed {
				pushDesc(ch, c.subsystem, "speed_bytes")
			}
		}

		if ifaceInfo.TxQueueLen != nil {
			pushDesc(ch, c.subsystem, "transmit_queue_length")
		}

		if ifaceInfo.Type != nil {
			pushDesc(ch, c.subsystem, "protocol_type")
		}
	}

	return
}