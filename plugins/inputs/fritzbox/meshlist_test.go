// meshlist_test.go
//
// Copyright (C) 2022-2024 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.

package fritzbox

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testMeshList1 = "testdata/meshlist1.json"
const testMeshList2 = "testdata/meshlist2.json"

func TestGetMasterSlavePaths1(t *testing.T) {
	meshList := loadTestMeshList(t, testMeshList1)
	masterSlavePaths := meshList.getMasterSlavePaths()
	require.Equal(t, 2, len(masterSlavePaths))
	require.Equal(t, [4]int{216000, 216000, 216000, 216000}, masterSlavePaths[0].getRoot().getDataRates())
	require.Equal(t, [4]int{1300000, 1300000, 1300000, 975000}, masterSlavePaths[1].getRoot().getDataRates())
}
func TestGetMasterSlavePaths2(t *testing.T) {
	meshList := loadTestMeshList(t, testMeshList2)
	masterSlavePaths := meshList.getMasterSlavePaths()
	require.Equal(t, 2, len(masterSlavePaths))
	require.Equal(t, [4]int{1000004, 1000003, 1000002, 1000001}, masterSlavePaths[0].getRoot().getDataRates())
	require.Equal(t, [4]int{1000004, 1000003, 1000002, 1000001}, masterSlavePaths[1].getRoot().getDataRates())
}
func TestGetClientPaths1(t *testing.T) {
	meshList := loadTestMeshList(t, testMeshList1)
	clientPaths := meshList.getClientPaths([]string{})
	require.Equal(t, 20, len(clientPaths))
}
func TestGetClientPaths2(t *testing.T) {
	meshList := loadTestMeshList(t, testMeshList2)
	clientPaths := meshList.getClientPaths([]string{})
	require.Equal(t, 12, len(clientPaths))
}

func loadTestMeshList(t *testing.T, filename string) *meshList {
	meshListBytes, err := os.ReadFile(filename)
	require.NoError(t, err)

	var meshList meshList

	err = json.Unmarshal(meshListBytes, &meshList)
	require.NoError(t, err)
	return &meshList
}
