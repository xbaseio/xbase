package dispatcher

import (
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/endpoint"
	"github.com/xbaseio/xbase/registry"
)

type serviceEndpoint struct {
	insID      string
	state      string
	status     registry.ServiceStatus
	endpoint   *endpoint.Endpoint
	weight     int
	currWeight int
}

type abstract struct {
	endpoints1 []*serviceEndpoint
	endpoints2 []*serviceEndpoint
	endpoints3 []*serviceEndpoint
	endpoints4 []*serviceEndpoint
	endpoints5 map[string]*serviceEndpoint
	endpoints6 []*serviceEndpoint
	endpoints7 []*serviceEndpoint
	endpoints8 []*serviceEndpoint
	endpoints9 []*serviceEndpoint
}

func newAbstract() abstract {
	return abstract{
		endpoints1: make([]*serviceEndpoint, 0),
		endpoints2: make([]*serviceEndpoint, 0),
		endpoints3: make([]*serviceEndpoint, 0),
		endpoints4: make([]*serviceEndpoint, 0),
		endpoints5: make(map[string]*serviceEndpoint),
		endpoints6: make([]*serviceEndpoint, 0),
		endpoints7: make([]*serviceEndpoint, 0),
		endpoints8: make([]*serviceEndpoint, 0),
		endpoints9: make([]*serviceEndpoint, 0),
	}
}

func (a *abstract) addServiceEndpoint(se *serviceEndpoint, balance bool) {
	a.endpoints5[se.insID] = se
	if !balance {
		return
	}

	switch se.status {
	case registry.ServiceStatusTest:
		switch se.state {
		case cluster.Work.String():
			a.endpoints6 = append(a.endpoints6, se)
		case cluster.Busy.String():
			a.endpoints7 = append(a.endpoints7, se)
		case cluster.Hang.String():
			a.endpoints3 = append(a.endpoints3, se)
		case cluster.Shut.String():
			a.endpoints4 = append(a.endpoints4, se)
		}
	case registry.ServiceStatusGray:
		switch se.state {
		case cluster.Work.String():
			a.endpoints8 = append(a.endpoints8, se)
		case cluster.Busy.String():
			a.endpoints9 = append(a.endpoints9, se)
		case cluster.Hang.String():
			a.endpoints3 = append(a.endpoints3, se)
		case cluster.Shut.String():
			a.endpoints4 = append(a.endpoints4, se)
		}
	default:
		switch se.state {
		case cluster.Work.String():
			a.endpoints1 = append(a.endpoints1, se)
		case cluster.Busy.String():
			a.endpoints2 = append(a.endpoints2, se)
		case cluster.Hang.String():
			a.endpoints3 = append(a.endpoints3, se)
		case cluster.Shut.String():
			a.endpoints4 = append(a.endpoints4, se)
		}
	}
}
