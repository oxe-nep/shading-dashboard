package model

import "time"

type SwitchConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Username string `json:"username"`
	Password string `json:"password"`
	Port     int    `json:"port,omitempty"`
}

type PortGroupMember struct {
	SwitchID string `json:"switchId"`
	Port     string `json:"port"`
}

type PortGroupConfig struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Members []PortGroupMember `json:"members"`
}

type AppConfig struct {
	Switches     []SwitchConfig    `json:"switches"`
	PortGroups   []PortGroupConfig `json:"portGroups"`
	AllowedVlans []int             `json:"allowedVlans,omitempty"`
}

type VLANInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name,omitempty"`
	Label string `json:"label"`
	Title string `json:"title"`
}

type PortState struct {
	Name            string `json:"name"`
	OperState       string `json:"operState"`
	AccessVLAN      *int   `json:"accessVlan"`
	AccessVlanLabel string `json:"accessVlanLabel,omitempty"`
	AccessVlanTitle string `json:"accessVlanTitle,omitempty"`
	AdminDown       bool   `json:"adminDown"`
	Description     string `json:"description,omitempty"`
}

type SwitchRuntimeState struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	IP            string      `json:"ip"`
	Online        bool        `json:"online"`
	Polling       bool        `json:"polling"`
	LastPollAt    *time.Time  `json:"lastPollAt"`
	LastSuccessAt *time.Time  `json:"lastSuccessAt"`
	Ports         []PortState `json:"ports"`
	Vlans         []VLANInfo  `json:"vlans,omitempty"`
	LastError     string      `json:"lastError,omitempty"`
}

type PortGroupRuntime struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Members      []PortGroupMember `json:"members"`
	CurrentVLAN  *int     `json:"currentVlan"`
	VLANLabel    string   `json:"vlanLabel"`
	Mixed        bool     `json:"mixed"`
}

type PortApplyResult struct {
	SwitchID string `json:"switchId"`
	Port     string `json:"port"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
}

type SetVLANRequest struct {
	VLAN int `json:"vlan"`
}

type WSMessage struct {
	Type            string               `json:"type"`
	Switches        []SwitchRuntimeState `json:"switches,omitempty"`
	PortGroups      []PortGroupRuntime   `json:"portGroups,omitempty"`
	DiscoveredVlans []VLANInfo           `json:"discoveredVlans,omitempty"`
	SelectableVlans []VLANInfo           `json:"selectableVlans,omitempty"`
	SwitchID  string                 `json:"switchId,omitempty"`
	GroupID   string                 `json:"groupId,omitempty"`
	Port      string                 `json:"port,omitempty"`
	VLAN      int                    `json:"vlan,omitempty"`
	OK        bool                   `json:"ok,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Results   []PortApplyResult      `json:"results,omitempty"`
}

type RuntimeSnapshot struct {
	Switches        []SwitchRuntimeState `json:"switches"`
	PortGroups      []PortGroupRuntime   `json:"portGroups"`
	DiscoveredVlans []VLANInfo           `json:"discoveredVlans,omitempty"`
	SelectableVlans []VLANInfo           `json:"selectableVlans,omitempty"`
}
