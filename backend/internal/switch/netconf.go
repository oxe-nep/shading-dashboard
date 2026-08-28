package switchdrv

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/oxe-nep/shading-dashboard/internal/model"
	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/util"
)

const defaultNetconfPort = 830

type Client struct {
	mu     sync.Mutex
	driver *netconf.Driver
	cfg    model.SwitchConfig
}

func NewClient(cfg model.SwitchConfig) *Client {
	return &Client{cfg: cfg}
}

func driverOptions(cfg model.SwitchConfig) []util.Option {
	port := cfg.Port
	if port == 0 {
		port = defaultNetconfPort
	}

	opts := []util.Option{
		options.WithAuthUsername(cfg.Username),
		options.WithAuthPassword(cfg.Password),
		options.WithAuthNoStrictKey(),
		options.WithPort(port),
		options.WithTimeoutOps(90 * time.Second),
	}

	// "system" SSH wrapper is unreliable for NETCONF on Windows.
	if runtime.GOOS == "windows" {
		opts = append(opts, options.WithTransportType("standard"))
	}

	return opts
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.driver != nil {
		return nil
	}

	opts := driverOptions(c.cfg)

	d, err := netconf.NewDriver(c.cfg.IP, opts...)
	if err != nil {
		return fmt.Errorf("netconf driver: %w", err)
	}

	if err := d.Open(); err != nil {
		return fmt.Errorf("netconf open: %w", err)
	}

	c.driver = d
	return nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.driver != nil {
		_ = c.driver.Close()
		c.driver = nil
	}
}

func (c *Client) reconnect(ctx context.Context) error {
	c.Close()
	return c.Connect(ctx)
}

func (c *Client) withDriver(ctx context.Context, fn func(*netconf.Driver) error) error {
	c.mu.Lock()
	if c.driver == nil {
		c.mu.Unlock()
		if err := c.Connect(ctx); err != nil {
			return err
		}
		c.mu.Lock()
	}
	d := c.driver
	c.mu.Unlock()

	err := fn(d)
	if err == nil {
		return nil
	}

	if err2 := c.reconnect(ctx); err2 != nil {
		return err
	}

	c.mu.Lock()
	d = c.driver
	c.mu.Unlock()
	return fn(d)
}

func (c *Client) GetPorts(ctx context.Context) ([]model.PortState, []model.VLANInfo, error) {
	var nativeRaw, vlanRaw, operRaw string
	err := c.withDriver(ctx, func(d *netconf.Driver) error {
		nativeResp, err := d.Get(interfaceFilter())
		if err != nil {
			return err
		}
		nativeRaw = nativeResp.Result

		vlanResp, err := d.Get(vlanListFilter())
		if err != nil {
			return err
		}
		vlanRaw = vlanResp.Result

		operResp, err := d.Get(operInterfaceFilter())
		if err != nil {
			return err
		}
		operRaw = operResp.Result
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	vlans := parseVLANList(vlanRaw)
	ports := parsePortsFromGet(nativeRaw)
	if len(ports) == 0 {
		return nil, nil, fmt.Errorf("no interfaces returned from switch")
	}
	applyOperStates(ports, parseOperStates(operRaw))
	applyVLANLabels(ports, parseVLANNames(vlanRaw))
	return ports, vlans, nil
}

func (c *Client) GetPort(ctx context.Context, port string) (model.PortState, error) {
	ifType, ifName, display := NormalizePortQuery(port)
	var nativeRaw, operRaw string
	err := c.withDriver(ctx, func(d *netconf.Driver) error {
		nativeResp, err := d.Get(interfaceFilterFor(ifType, ifName))
		if err != nil {
			return err
		}
		nativeRaw = nativeResp.Result

		operResp, err := d.Get(operInterfaceFilterFor(ifType, ifName))
		if err != nil {
			return err
		}
		operRaw = operResp.Result
		return nil
	})
	if err != nil {
		return model.PortState{}, err
	}

	ports := parseRawPorts(nativeRaw)
	applyOperStates(ports, parseOperStates(operRaw))
	for _, p := range ports {
		if p.Name == display {
			return p, nil
		}
	}
	return model.PortState{}, fmt.Errorf("port %s not found in switch response", display)
}

func (c *Client) GetVLANIDs(ctx context.Context) (map[int]struct{}, error) {
	var raw string
	err := c.withDriver(ctx, func(d *netconf.Driver) error {
		resp, err := d.Get(vlanListFilter())
		if err != nil {
			return err
		}
		raw = resp.Result
		return nil
	})
	if err != nil {
		return nil, err
	}
	ids := parseVLANIDs(raw)
	if ids == nil {
		return map[int]struct{}{}, nil
	}
	return ids, nil
}

func (c *Client) SetAccessVLAN(ctx context.Context, port string, vlan int) error {
	ifType, ifName, _ := NormalizePortQuery(port)
	config := editAccessVLANConfig(ifType, ifName, vlan)

	return c.withDriver(ctx, func(d *netconf.Driver) error {
		resp, err := d.EditConfig("running", config)
		if err != nil {
			return err
		}
		if resp.Failed != nil {
			return resp.Failed
		}
		return nil
	})
}
