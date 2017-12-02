package main

import (
	"fmt"
	"math"
	"sort"
	//"gonum.org/v1/gonum/mat"
	//"math/rand"
)

func unmain() {
	fmt.Println("Start!")
	nodeMap := make(map[string][]int)
	nodeMap["n1"] = []int{1, 1, 2, 0, 0}
	nodeMap["n2"] = []int{2, 2, 0, 0, 2}
	nodeMap["n3"] = []int{1, 0, 0, 0, 2}
	nodeMap["me"] = []int{1, 0, 0, 0, 2}
	/*nodeMap["n1"] = []int{1, 1, 1, 1, 1}
	nodeMap["n2"] = []int{0, 0, 0, 0, 0}
	nodeMap["n3"] = []int{0, 0, 0, 0, 0}
	nodeMap["me"] = []int{0, 0, 0, 0, 0}*/

	steps, failed := getConvergenceSpeed(nodeMap, 2)
	cost1, _, _ := getCost(nodeMap, 2)
	fmt.Println("CS", cost1)
	fmt.Println("CS", steps)
	fmt.Println("No-converge?", failed)

}

func getCost(nodeMap map[string][]int, redLev int) (float64, float64, bool) {

	data := make([]float64, 20)
	i := 0

	for k, v := range nodeMap {

		// *** BEAWARE OF CALCULATING EXACT BANDWIDTH! ***
		var bandwidth float64 = 1

		for l := range v {

			var atrc float64 = 1
			if nodeMap["me"][l] > 0 {
				atrc = 0
			}

			cost := math.Inf(1)

			switch nodeMap[k][l] {
			case 1:
				cost = 1
			case 2:
				cost = 1.05
			case 3:
				cost = 1.11
			}

			data[i*5+l] = (1 / cost) * atrc * bandwidth
		}

		i++
	}

	// 각 row에서 가장 작은 걸 택하고, 모두 더한다.

	costVec := make([]float64, len(nodeMap))

	var totalSeqCost float64
	var totalPrlCost float64

	for nrow := 0; nrow < len(nodeMap); nrow++ {

		minCost := math.Inf(1)

		for ncol := 0; ncol < 5; ncol++ {

			// Do not check null valued chunk
			if data[nrow*5+ncol] == 0 {
				continue
			}

			cost := 1 / data[nrow*5+ncol]

			if minCost > cost {
				minCost = cost
			}
		}

		costVec[nrow] = minCost
	}

	sort.Float64s(costVec)

	// Update cost values
	broken := false

	for nrow := 0; nrow < len(costVec)-redLev/2; nrow++ {
		cost := costVec[nrow]

		if math.IsInf(cost, 1) {
			broken = true
			continue
		}

		totalSeqCost += cost
		if totalPrlCost < cost {
			totalPrlCost = cost
		}
	}

	return totalSeqCost, totalPrlCost, broken
}

func getConvergenceSpeed(nodeMap map[string][]int, redLev int) (int, bool) {

	convMat := make([][]int, 0, len(nodeMap))

	// Fill 2d array with node file availability data
	for i := range nodeMap {
		data := make([]int, len(nodeMap[i]))
		copy(data, nodeMap[i])
		convMat = append(convMat, data)
	}

	steps := 0
	failed := false

	fmt.Println(convMat)
	for !isConverged(convMat) {
		convMat, failed = converge(convMat, redLev)
		fmt.Println(convMat)
		// non-convergable...
		if failed {
			break
		}
		steps++
	}

	return steps, failed
}

// Let's roll!
func converge(mat [][]int, redLev int) ([][]int, bool) {

	//clone mat.
	cmat := make([][]int, len(mat))
	for i := range mat {
		cmat[i] = make([]int, len(mat[i]))
		copy(cmat[i], mat[i])
	}

	completedMarker := make([]bool, len(mat))
	changed := false

	for i := range mat { // Taker

		// Check if completed. if yes, continue
		if completedMarker[i] {
			continue
		} else {

			count := 0
			for j := range mat[i] {
				if mat[i][j] > 0 {
					count++
				}
			}

			if count >= len(mat[i])-redLev/2 {
				completedMarker[i] = true

				// Fill the rest
				for j := range mat[i] {
					if mat[i][j] == 0 {
						cmat[i][j] = 3
						changed = true
					}
				}

				continue
			}
		}

		for j := range mat { // Giver

			// Taking from myself? Nah.
			if i == j {
				continue
			}

			// Find needed chunks
			for k := range mat[j] {

				// Asymmetry found! Fill
				if mat[j][k] > 0 && mat[i][k] < 1 {
					cmat[i][k] = mat[j][k]
					changed = true
					break
				}
			}
		}
	}
	return cmat, !changed
}

// Convergence test
func isConverged(mat [][]int) bool {
	for i := range mat {
		for j := range mat[i] {
			if mat[i][j] < 1 {
				return false
			}
		}
	}
	return true
}
