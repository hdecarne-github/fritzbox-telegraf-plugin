// meshlist.go
//
// Copyright (C) 2022-2023 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.

package fritzbox

import (
	"github.com/google/uuid"
)

type meshList struct {
	SchemaVersion string         `json:"schema_version"`
	Nodes         []meshListNode `json:"nodes"`
	nodeTable     map[string]*meshListNode
}

func (meshList *meshList) lookupNode(uid string) *meshListNode {
	if meshList.nodeTable == nil {
		meshList.nodeTable = make(map[string]*meshListNode, 0)
		for nodeIndex, node := range meshList.Nodes {
			meshList.nodeTable[node.Uid] = &meshList.Nodes[nodeIndex]
		}
	}
	return meshList.nodeTable[uid]
}

type meshListNode struct {
	Uid            string                  `json:"uid"`
	DeviceName     string                  `json:"device_name"`
	IsMeshed       bool                    `json:"is_meshed"`
	MeshRole       string                  `json:"mesh_role"`
	NodeInterfaces []meshListNodeInterface `json:"node_interfaces"`
}

func (node *meshListNode) hasValidDeviceName() bool {
	if node.DeviceName == "" {
		return false
	}
	_, err := uuid.Parse(node.DeviceName)
	return err != nil
}

func (node *meshListNode) isMaster() bool {
	return node.IsMeshed && node.MeshRole == "master"
}

func (node *meshListNode) isSlave() bool {
	return node.IsMeshed && node.MeshRole == "slave"
}

type meshListNodeInterface struct {
	Uid       string             `json:"uid"`
	Name      string             `json:"name"`
	Type      string             `json:"type"`
	NodeLinks []meshListNodeLink `json:"node_links"`
}

type meshListNodeLink struct {
	State             string `json:"state"`
	Node1Uid          string `json:"node_1_uid"`
	Node2Uid          string `json:"node_2_uid"`
	NodeInterface1Uid string `json:"node_interface_1_uid"`
	NodeInterface2Uid string `json:"node_interface_2_uid"`
	MaxDataRateRx     int    `json:"max_data_rate_rx"`
	MaxDataRateTx     int    `json:"max_data_rate_tx"`
	CurDataRateRx     int    `json:"cur_data_rate_rx"`
	CurDataRateTx     int    `json:"cur_data_rate_tx"`
}

func (link *meshListNodeLink) isConnected() bool {
	return link.State == "CONNECTED"
}

func (link *meshListNodeLink) isConnectedTo(nodeInterface *meshListNodeInterface) bool {
	return link.isConnected() && (link.NodeInterface1Uid == nodeInterface.Uid || link.NodeInterface2Uid == nodeInterface.Uid)
}

type meshPath struct {
	parent        *meshPath
	node          *meshListNode
	nodeInterface *meshListNodeInterface
	nodeLink      *meshListNodeLink
}

func (path *meshPath) getRoot() *meshPath {
	currentPath := path
	for {
		if currentPath.parent == nil {
			break
		}
		currentPath = currentPath.parent
	}
	return currentPath
}

func (path *meshPath) getPeerNodeUid() string {
	if path.node.Uid == path.nodeLink.Node1Uid {
		return path.nodeLink.Node2Uid
	}
	return path.nodeLink.Node1Uid
}

func (path *meshPath) getDataRates() [4]int {
	if path.node.Uid == path.nodeLink.Node1Uid {
		return [4]int{path.nodeLink.MaxDataRateRx, path.nodeLink.MaxDataRateTx, path.nodeLink.CurDataRateRx, path.nodeLink.CurDataRateTx}
	}
	return [4]int{path.nodeLink.MaxDataRateTx, path.nodeLink.MaxDataRateRx, path.nodeLink.CurDataRateTx, path.nodeLink.CurDataRateRx}
}

func (path *meshPath) contains(node *meshListNode) bool {
	currentPath := path
	for {
		if currentPath.node.Uid == node.Uid {
			return true
		}
		if currentPath.parent == nil {
			break
		}
		currentPath = currentPath.parent
	}
	return false
}

func (meshList *meshList) getMasterSlavePaths() []*meshPath {
	paths := make([]*meshPath, 0)
	for masterNodeIndex, masterNode := range meshList.Nodes {
		if masterNode.isMaster() {
			for masterInterfaceIndex, masterInterface := range masterNode.NodeInterfaces {
				for masterLinkIndex, masterLink := range masterInterface.NodeLinks {
					if masterLink.isConnected() {
						path := &meshPath{
							node:          &meshList.Nodes[masterNodeIndex],
							nodeInterface: &masterNode.NodeInterfaces[masterInterfaceIndex],
							nodeLink:      &masterInterface.NodeLinks[masterLinkIndex],
						}
						paths = meshList.collectMasterSlavePaths(paths, path)
					}
				}
			}
		}
	}
	return paths
}

func (meshList *meshList) collectMasterSlavePaths(paths []*meshPath, path *meshPath) []*meshPath {
	updatedPaths := paths
	peerNode := meshList.lookupNode(path.getPeerNodeUid())
	if peerNode != nil && !path.contains(peerNode) {
		if peerNode.isSlave() && peerNode.hasValidDeviceName() {
			for peerInterfaceIndex, peerInterface := range peerNode.NodeInterfaces {
				for peerLinkIndex, peerLink := range peerInterface.NodeLinks {
					if peerLink.isConnectedTo(path.nodeInterface) {
						updatedPath := &meshPath{
							parent:        path,
							node:          peerNode,
							nodeInterface: &peerNode.NodeInterfaces[peerInterfaceIndex],
							nodeLink:      &peerInterface.NodeLinks[peerLinkIndex],
						}
						updatedPaths = append(updatedPaths, updatedPath)
					}
				}
			}
		} else {
			for peerInterfaceIndex, peerInterface := range peerNode.NodeInterfaces {
				for peerLinkIndex, peerLink := range peerInterface.NodeLinks {
					if peerLink.isConnected() {
						updatedPath := &meshPath{
							parent:        path,
							node:          peerNode,
							nodeInterface: &peerNode.NodeInterfaces[peerInterfaceIndex],
							nodeLink:      &peerInterface.NodeLinks[peerLinkIndex],
						}
						updatedPaths = meshList.collectMasterSlavePaths(updatedPaths, updatedPath)
					}
				}
			}
		}
	}
	return updatedPaths
}

func (meshList *meshList) getClientPaths(clientTypes []string) []*meshPath {
	paths := make([]*meshPath, 0)
	for clientNodeIndex, clientNode := range meshList.Nodes {
		if !clientNode.IsMeshed && clientNode.hasValidDeviceName() {
			for clientInterfaceIndex, clientInterface := range clientNode.NodeInterfaces {
				includeClient := len(clientTypes) == 0
				for _, clientType := range clientTypes {
					if clientInterface.Type == clientType {
						includeClient = true
						break
					}
				}

				if includeClient {
					for clientLinkIndex, clientLink := range clientInterface.NodeLinks {
						if clientLink.isConnected() {
							client := &meshPath{
								node:          &meshList.Nodes[clientNodeIndex],
								nodeInterface: &clientNode.NodeInterfaces[clientInterfaceIndex],
								nodeLink:      &clientInterface.NodeLinks[clientLinkIndex],
							}
							peerNode := meshList.lookupNode(client.getPeerNodeUid())
							if peerNode != nil {
								for peerInterfaceIndex, peerInterface := range peerNode.NodeInterfaces {
									if clientLink.isConnectedTo(&peerInterface) {
										peer := &meshPath{
											node:          peerNode,
											nodeInterface: &peerNode.NodeInterfaces[peerInterfaceIndex],
											nodeLink:      client.nodeLink,
										}
										client.parent = peer
										paths = append(paths, client)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return paths
}
