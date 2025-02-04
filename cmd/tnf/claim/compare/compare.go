package compare

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/test-network-function/cnf-certification-test/cmd/tnf/claim/compare/testcases"
	"github.com/test-network-function/cnf-certification-test/cmd/tnf/pkg/claim"
)

var (
	Claim1FilePathFlag string
	Claim2FilePathFlag string

	claimCompareFiles = &cobra.Command{
		Use:   "compare",
		Short: "Compare two claim files.",
		RunE:  claimCompare,
	}
)

func NewCommand() *cobra.Command {
	claimCompareFiles.Flags().StringVarP(
		&Claim1FilePathFlag, "claim1", "1", "",
		"existing claim1 file. (Required) first file to compare",
	)
	claimCompareFiles.Flags().StringVarP(
		&Claim2FilePathFlag, "claim2", "2", "",
		"existing claim2 file. (Required) second file to compare",
	)
	err := claimCompareFiles.MarkFlagRequired("claim1")
	if err != nil {
		log.Errorf("Failed to mark flag claim1 as required: %v", err)
		return nil
	}
	err = claimCompareFiles.MarkFlagRequired("claim2")
	if err != nil {
		log.Errorf("Failed to mark flag claim2 as required: %v", err)
		return nil
	}

	return claimCompareFiles
}

func claimCompare(_ *cobra.Command, _ []string) error {
	err := claimCompareFilesfunc(Claim1FilePathFlag, Claim2FilePathFlag)
	if err != nil {
		log.Fatalf("Error comparing claim files: %v", err)
	}
	return nil
}

func claimCompareFilesfunc(claim1, claim2 string) error {
	// readfiles
	claimdata1, err := os.ReadFile(claim1)
	if err != nil {
		return fmt.Errorf("failed reading claim1 file: %v", err)
	}

	claimdata2, err := os.ReadFile(claim2)
	if err != nil {
		return fmt.Errorf("failed reading claim2 file: %v", err)
	}

	// unmarshal the files
	claimFile1Data, err := unmarshalClaimFile(claimdata1)
	if err != nil {
		return fmt.Errorf("failed to unmarshal claim1 file: %v", err)
	}

	claimFile2Data, err := unmarshalClaimFile(claimdata2)
	if err != nil {
		return fmt.Errorf("failed to unmarshal claim2 file: %v", err)
	}

	// compares function
	if compare2NodeList(claimFile1Data.Claim.Nodes.NodesHwInfo, claimFile2Data.Claim.Nodes.NodesHwInfo) {
		log.Info("we are comparing two different cluster, all the nodes are different in both claim")
	}

	compare2cni(claimFile1Data.Claim.Nodes.CniPlugins, claimFile2Data.Claim.Nodes.CniPlugins)

	// Get and show test cases results summary and differences
	tcsDiffReport := testcases.GetDiffReport(claimFile1Data.Claim.Results, claimFile2Data.Claim.Results)
	fmt.Println(&tcsDiffReport)

	return nil
}

func unmarshalClaimFile(claimdata []byte) (claim.Schema, error) {
	var claimDataResult claim.Schema
	err := json.Unmarshal(claimdata, &claimDataResult)
	if err != nil {
		return claim.Schema{}, err
	}
	return claimDataResult, nil
}

// function that receiving 2 hwinfo2 and prints
// print name of node that claim1 have and not have them in claim2
// print name of node that claim2 have and not have them in claim1
func compare2NodeList(hwinfo1, hwinfo2 map[string]interface{}) bool {
	var nodesIn1, nodesIn2, nodeNotIn1, nodeNotIn2 []string

	for key := range hwinfo1 {
		nodesIn1 = append(nodesIn1, key)
	}
	for key := range hwinfo2 {
		nodesIn2 = append(nodesIn2, key)
	}
	nodeNotIn1, nodeNotIn2 = missing(nodesIn2, nodesIn1)
	fmt.Println("nodes that claim2 have but claim1 do not have ", nodeNotIn1)
	fmt.Println("nodes that claim1 have but claim2 do not have ", nodeNotIn2)
	return compareEqual2String(nodeNotIn1, nodesIn2) && compareEqual2String(nodeNotIn2, nodesIn1)
}

func compareEqual2String(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// empty struct (0 bytes)
type void struct{}

// missing compares two slices and returns slice of differences, between 2 sides
func missing(a, b []string) (diffsAfromB, diffsBfromA []string) {
	// create map with length of the 'a' slice
	ma := make(map[string]void, len(a))
	mb := make(map[string]void, len(b))
	// Convert first slice to map with empty struct (0 bytes)
	for _, ka := range a {
		ma[ka] = void{}
	}
	// Convert first slice to map with empty struct (0 bytes)
	for _, ka := range b {
		mb[ka] = void{}
	}
	// find missing values in b
	for _, kb := range b {
		if _, ok := ma[kb]; !ok {
			diffsAfromB = append(diffsAfromB, kb)
		}
	}
	for _, ka := range a {
		if _, ok := mb[ka]; !ok {
			diffsBfromA = append(diffsBfromA, ka)
		}
	}
	return diffsAfromB, diffsBfromA
}

func removeDuplicateValues(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// compare between 2 cni objects and print the difference
func compare2cni(cni1, cni2 map[string][]claim.Cni) {
	for node, val := range cni1 {
		for node2, val2 := range cni2 {
			if node != node2 {
				continue
			}
			diffPlugIn, notFoundNamesIn1, notFoundNamesIn2 := compare2cniHelper(val, val2, node2)
			if len(notFoundNamesIn1) != 0 && notFoundNamesIn1 != nil {
				log.Infof("in node %s  CNIs found in claim1 but not present in claim2: %v ", node2, notFoundNamesIn1)
			}
			if len(notFoundNamesIn2) != 0 && notFoundNamesIn2 != nil {
				log.Infof("in node %s  CNIs found in claim2 but not present in claim1: %v", node2, notFoundNamesIn2)
			}
			if len(diffPlugIn) != 0 {
				log.Infof("in node %s  CNIs present in both claim 1 and 2 but with different plugins: %v", node2, diffPlugIn)
			}
			break
		}
	}
}

// receiving 2 cnistruct and return :
// 1. name of cni's that have same name but the plugin value are different - diffPlugins
// 2. name of cni's that found on claim2 but not in claim1 - notFoundNamesIn1
// 3. name of cni's that found on claim1 but not in claim2 - notFoundNamesIn3
func compare2cniHelper(cniList1, cniList2 []claim.Cni, node string) (diffPlugins []claim.Cni, notFoundNamesIn1, notFoundNamesIn2 []string) {
	var cniList1Name, cniList2Name []string
	if len(cniList1) == 0 {
		log.Infof("in node %s CNIs present in claim2 and on claim1 that node do not have cni values: %v", node, cniList2)
		return nil, nil, nil
	}
	if len(cniList2) == 0 {
		log.Infof("in node %s CNIs present in claim1 and on claim2 that node do not have cni values: %v", node, cniList1)
		return nil, nil, nil
	}
	for _, plugin1 := range cniList1 {
		cniList1Name = append(cniList1Name, plugin1.Name)
		for _, plugin2 := range cniList2 {
			cniList2Name = append(cniList2Name, plugin2.Name)
			if plugin2.Name != plugin1.Name {
				continue
			}
			if plugin2.Plugins != nil {
				if len(plugin2.Plugins) != len(plugin1.Plugins) {
					diffPlugins = append(diffPlugins, plugin1)
				}
			}
		}
	}
	notFoundNamesIn1, notFoundNamesIn2 = missing(cniList2Name, cniList1Name)
	return diffPlugins, removeDuplicateValues(notFoundNamesIn1), removeDuplicateValues(notFoundNamesIn2)
}
